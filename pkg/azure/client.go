package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/LambdaTest/test-at-scale/config"
	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
	"github.com/LambdaTest/test-at-scale/pkg/utils"
)

var (
	defaultBufferSize     = 3 * 1024 * 1024
	defaultMaxBuffers     = 4
	coverageContainerName = "coverage"
	maxRetry              = 10
)

// store represents the azure storage
type store struct {
	requests        core.Requests
	containerClient azblob.ContainerClient
	logger          lumber.Logger
	endpoint        string
}

// request body for getting SAS URL API.
type request struct {
	Purpose core.SASURLPurpose `json:"purpose" validate:"oneof=cache workspace_cache pre_run_logs post_run_logs execution_logs"`
}

//  response body for  get SAS URL API.
type response struct {
	SASURL string `json:"sas_url"`
}

// NewAzureBlobEnv returns a new Azure blob store.
func NewAzureBlobEnv(cfg *config.NucleusConfig, requests core.Requests, logger lumber.Logger) (core.AzureClient, error) {
	// if non coverage mode then use Azure SAS Token
	if !cfg.CoverageMode {
		return &store{
			requests: requests,
			logger:   logger,
			endpoint: global.NeuronHost + "/internal/sas-token",
		}, nil
	}
	// FIXME: Hack for synapse
	if cfg.LocalRunner {
		cfg.Azure.StorageAccountName = "dummy-account"
		cfg.Azure.StorageAccessKey = "dummy-access-key"
	}
	if cfg.Azure.StorageAccountName == "" || cfg.Azure.StorageAccessKey == "" {
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
	serviceClient, err := azblob.NewServiceClientWithSharedKey(u.String(), credential, getClientOptions())
	if err != nil {
		logger.Errorf("Failed to create azure service client, error: %v", err)
		return nil, err
	}

	return &store{
		requests:        requests,
		logger:          logger,
		endpoint:        global.NeuronHost + "/internal/sas-token",
		containerClient: serviceClient.NewContainerClient(coverageContainerName),
	}, nil
}

// FindUsingSASUrl download object based on sasURL
func (s *store) FindUsingSASUrl(ctx context.Context, sasURL string) (io.ReadCloser, error) {
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
func (s *store) CreateUsingSASURL(ctx context.Context, sasURL string, reader io.Reader, mimeType string) (string, error) {
	u, err := url.Parse(sasURL)
	if err != nil {
		return "", err
	}
	blobClient, err := azblob.NewBlockBlobClientWithNoCredential(u.String(), getClientOptions())
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
func (s *store) Find(ctx context.Context, path string) (io.ReadCloser, error) {
	blobClient := s.containerClient.NewBlockBlobClient(path)
	out, err := blobClient.Download(ctx, &azblob.DownloadBlobOptions{})
	if err != nil {
		return nil, handleError(err)
	}
	defer out.RawResponse.Body.Close()

	return out.Body(&azblob.RetryReaderOptions{MaxRetryRequests: 5}), nil
}

// Create function ulploads blob to URI
func (s *store) Create(ctx context.Context, path string, reader io.Reader, mimeType string) (string, error) {
	blobClient := s.containerClient.NewBlockBlobClient(path)
	_, err := blobClient.UploadStreamToBlockBlob(ctx, reader, azblob.UploadStreamToBlockBlobOptions{
		HTTPHeaders: &azblob.BlobHTTPHeaders{BlobContentType: &mimeType},
		BufferSize:  defaultBufferSize,
		MaxBuffers:  defaultMaxBuffers,
	})

	return blobClient.URL(), err
}

// GetSASURL calls request neuron to get the SAS url
func (s *store) GetSASURL(ctx context.Context, purpose core.SASURLPurpose, query map[string]interface{}) (string, error) {
	reqPayload := &request{Purpose: purpose}
	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		s.logger.Errorf("failed to marshal request body %v", err)
		return "", err
	}
	defaultQuery, headers := utils.GetDefaultQueryAndHeaders()
	for key, val := range defaultQuery {
		if query == nil {
			query = make(map[string]interface{})
		}
		query[key] = val
	}
	rawBytes, _, err := s.requests.MakeAPIRequest(ctx, http.MethodPost, s.endpoint, reqBody, query, headers)
	if err != nil {
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
func (s *store) Exists(ctx context.Context, path string) (bool, error) {
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
		if errResp.ErrorCode == azblob.StorageErrorCodeBlobNotFound {
			return errs.ErrNotFound
		}
	}
	return err
}

func getClientOptions() *azblob.ClientOptions {
	return &azblob.ClientOptions{
		Retry: policy.RetryOptions{
			MaxRetries: int32(maxRetry),
			TryTimeout: global.DefaultAPITimeout,
		},
	}
}
