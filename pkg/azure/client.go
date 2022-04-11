package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
)

var (
	defaultBufferSize     = 3 * 1024 * 1024
	defaultMaxBuffers     = 4
	coverageContainerName = "coverage"
)

// Store represents the azure storage
type Store struct {
	containerClient azblob.ContainerClient
	httpClient      http.Client
	logger          lumber.Logger
}

// request body for getting SAS URL API.
type request struct {
	BlobPath string             `json:"blob_path"`
	BlobType core.ContainerType `json:"blob_type"`
}

//  response body for  get SAS URL API.
type response struct {
	SASURL string `json:"sas_url"`
}

// NewAzureBlobEnv returns a new Azure blob store.
func NewAzureBlobEnv(cfg *config.NucleusConfig, logger lumber.Logger) (core.AzureClient, error) {
	// if non coverage mode then use Azure SAS Token
	if !cfg.CoverageMode {
		return &Store{
			logger: logger,
			httpClient: http.Client{
				Timeout: global.DefaultHTTPTimeout,
			},
		}, nil
	}
	// FIXME: Hack for synapse
	if cfg.LocalRunner {
		cfg.Azure.StorageAccountName = "dummy"
		cfg.Azure.StorageAccessKey = "dummy"
	}
	if len(cfg.Azure.StorageAccountName) == 0 || len(cfg.Azure.StorageAccessKey) == 0 {
		return nil, errors.New("either the storage account or storage access key environment variable is not set")
	}
	credential, err := azblob.NewSharedKeyCredential(cfg.Azure.StorageAccountName, cfg.Azure.StorageAccessKey)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", cfg.Azure.StorageAccountName, cfg.Azure.ContainerName))
	if err != nil {
		return nil, err
	}

	serviceClient, err := azblob.NewServiceClientWithSharedKey(u.String(), credential, nil)
	if err != nil {
		logger.Errorf("Failed to create azure service client, error: %v", err)
		return nil, err
	}

	return &Store{
		logger: logger,
		httpClient: http.Client{
			Timeout: global.DefaultHTTPTimeout,
		},
		containerClient: serviceClient.NewContainerClient(coverageContainerName),
	}, nil
}

// FindUsingSASUrl download object based on sasURL
func (s *Store) FindUsingSASUrl(ctx context.Context, sasURL string) (io.ReadCloser, error) {
	u, err := url.Parse(sasURL)
	if err != nil {
		return nil, err
	}
	blobClient, err := azblob.NewBlockBlobClientWithNoCredential(u.String(), &azblob.ClientOptions{})
	if err != nil {
		s.logger.Errorf("failed to create blob client, error: %v", err)
		return nil, err
	}
	s.logger.Debugf("Downloading blob from %s", blobClient.URL())
	out, err := blobClient.Download(ctx, &azblob.DownloadBlobOptions{})
	if err != nil {
		return nil, handleError(err)
	}

	return out.Body(&azblob.RetryReaderOptions{MaxRetryRequests: 5}), nil
}

// CreateUsingSASURL creates object using sasURL
func (s *Store) CreateUsingSASURL(ctx context.Context, sasURL string, reader io.Reader, mimeType string) (string, error) {
	u, err := url.Parse(sasURL)
	if err != nil {
		return "", err
	}
	blobClient, err := azblob.NewBlockBlobClientWithNoCredential(u.String(), &azblob.ClientOptions{})
	if err != nil {
		s.logger.Errorf("failed to create blob client, error: %v", err)
		return "", err
	}
	s.logger.Debugf("Uploading blob to %s", blobClient.URL())

	_, err = blobClient.UploadStreamToBlockBlob(ctx, reader, azblob.UploadStreamToBlockBlobOptions{
		HTTPHeaders: &azblob.BlobHTTPHeaders{BlobContentType: &mimeType},
		BufferSize:  defaultBufferSize,
		MaxBuffers:  defaultMaxBuffers,
	})

	return blobClient.URL(), err
}

// Find function downloads blob based on URI
func (s *Store) Find(ctx context.Context, path string) (io.ReadCloser, error) {
	blobClient := s.containerClient.NewBlockBlobClient(path)
	out, err := blobClient.Download(ctx, &azblob.DownloadBlobOptions{})
	if err != nil {
		return nil, handleError(err)
	}
	defer out.RawResponse.Body.Close()

	return out.Body(&azblob.RetryReaderOptions{MaxRetryRequests: 5}), nil
}

// Create function ulploads blob to URI
func (s *Store) Create(ctx context.Context, path string, reader io.Reader, mimeType string) (string, error) {
	blobClient := s.containerClient.NewBlockBlobClient(path)
	_, err := blobClient.UploadStreamToBlockBlob(ctx, reader, azblob.UploadStreamToBlockBlobOptions{
		HTTPHeaders: &azblob.BlobHTTPHeaders{BlobContentType: &mimeType},
		BufferSize:  defaultBufferSize,
		MaxBuffers:  defaultMaxBuffers,
	})

	return blobClient.URL(), err
}

// GetSASURL calls request neuron to get the SAS url
func (s *Store) GetSASURL(ctx context.Context, containerPath string, containerType core.ContainerType) (string, error) {
	reqPayload := &request{
		BlobPath: containerPath,
		BlobType: containerType,
	}
	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		s.logger.Errorf("failed to marshal request body %v", err)
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/%s", global.NeuronHost, "internal/sas-token"), bytes.NewBuffer(reqBody))
	if err != nil {
		s.logger.Errorf("error while creating http request, error %v", err)
		return "", err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Errorf("error while getting SAS URL, error %v", err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		s.logger.Errorf("error while getting SAS Token, status code %d", resp.StatusCode)
		return "", errs.ErrAPIStatus
	}

	rawBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.logger.Errorf("error while reading SAS token response, error %v", err)
		return "", err
	}
	payload := new(response)
	err = json.Unmarshal(rawBytes, payload)
	if err != nil {
		s.logger.Errorf("Error while unmarshalling json, error %v", err)
		return "", err
	}
	return payload.SASURL, nil
}

// Exists checks the blob if exists
func (s *Store) Exists(ctx context.Context, path string) (bool, error) {
	blobClient := s.containerClient.NewBlockBlobClient(path)
	get, err := blobClient.GetProperties(ctx, &azblob.GetBlobPropertiesOptions{})
	if err != nil {
		return false, fmt.Errorf("check if object exists, %w", err)
	}
	statusCode := get.RawResponse.StatusCode
	defer get.RawResponse.Body.Close()
	return statusCode == http.StatusOK, nil
}

func handleError(err error) error {
	if err == nil {
		return nil
	}
	var errResp *azblob.StorageError
	if internalErr, ok := err.(*azblob.InternalError); ok && internalErr.As(&errResp) {
		switch errResp.ErrorCode { // Compare serviceCode to ServiceCodeXxx constants
		case azblob.StorageErrorCodeBlobNotFound:
			return errs.ErrNotFound
		}
	}
	return err

}
