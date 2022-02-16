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

	"github.com/Azure/azure-pipeline-go/pipeline"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/LambdaTest/synapse/config"
	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/errs"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
)

var (
	defaultBufferSize    = 3 * 1024 * 1024
	defaultMaxBuffers    = 4
	defaultContainerName = "cache"
)

// Store represents the azure storage
type Store struct {
	containerName      string
	storageAccountName string
	storageAccessKey   string
	containerURL       *azblob.ContainerURL
	azurePipeLine      *pipeline.Pipeline
	httpClient         http.Client
	logger             lumber.Logger
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
			logger:        logger,
			containerName: defaultContainerName,
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

	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	URL, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", cfg.Azure.StorageAccountName, cfg.Azure.ContainerName))
	if err != nil {
		return nil, err
	}
	containerURL := azblob.NewContainerURL(*URL, p)
	return &Store{
		containerName:      defaultContainerName,
		storageAccountName: cfg.Azure.StorageAccountName,
		storageAccessKey:   cfg.Azure.StorageAccessKey,
		containerURL:       &containerURL,
		azurePipeLine:      &p,
	}, nil
}

// FindUsingSASUrl download object based on sasURL
func (s *Store) FindUsingSASUrl(ctx context.Context, sasURL string) (io.ReadCloser, error) {
	u, err := url.Parse(sasURL)
	if err != nil {
		return nil, err
	}
	blobURL := azblob.NewBlobURL(*u, azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{}))

	out, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return nil, handleError(err)
	}
	return out.Body(azblob.RetryReaderOptions{MaxRetryRequests: 5}), nil
}

// CreateUsingSASURL creates object using sasURL
func (s *Store) CreateUsingSASURL(ctx context.Context, sasURL string, reader io.Reader, mimeType string) (string, error) {
	u, err := url.Parse(sasURL)
	if err != nil {
		return "", err
	}
	blobURL := azblob.NewBlockBlobURL(*u, azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{}))
	_, err = azblob.UploadStreamToBlockBlob(ctx, reader, blobURL, azblob.UploadStreamToBlockBlobOptions{
		BlobHTTPHeaders: azblob.BlobHTTPHeaders{ContentType: mimeType},
		BufferSize:      defaultBufferSize,
		MaxBuffers:      defaultMaxBuffers,
	})

	return blobURL.String(), err
}

// Find function downloads blob based on URI
func (s *Store) Find(ctx context.Context, path string) (io.ReadCloser, error) {
	blobURL := s.containerURL.NewBlockBlobURL(path)
	out, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return nil, handleError(err)
	}
	return out.Body(azblob.RetryReaderOptions{MaxRetryRequests: 5}), nil
}

// Create function ulploads blob to URI
func (s *Store) Create(ctx context.Context, path string, reader io.Reader, mimeType string) (string, error) {
	blobURL := s.containerURL.NewBlockBlobURL(path)
	_, err := azblob.UploadStreamToBlockBlob(ctx, reader, blobURL, azblob.UploadStreamToBlockBlobOptions{
		BlobHTTPHeaders: azblob.BlobHTTPHeaders{ContentType: mimeType},
		BufferSize:      defaultBufferSize,
		MaxBuffers:      defaultMaxBuffers,
	})

	return blobURL.String(), err
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
		return "", errs.ErrApiStatus
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
	blobURL := s.containerURL.NewBlockBlobURL(path)
	get, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return false, fmt.Errorf("check if object exists, %w", err)
	}

	return get.StatusCode() == http.StatusOK, nil
}

func handleError(err error) error {
	if err == nil {
		return nil
	}
	if serr, ok := err.(azblob.StorageError); ok { // This error is a Service-specific
		switch serr.ServiceCode() { // Compare serviceCode to ServiceCodeXxx constants
		case azblob.ServiceCodeBlobNotFound:
			return errs.ErrNotFound
		}
	}
	return err

}
