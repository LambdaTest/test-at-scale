package cachemanager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/LambdaTest/synapse/pkg/core"
	"github.com/LambdaTest/synapse/pkg/errs"
	"github.com/LambdaTest/synapse/pkg/fileutils"
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/LambdaTest/synapse/pkg/lumber"
)

const (
	yarnLock                    = "yarn.lock"
	packageLock                 = "package-lock.json"
	npmShrinkwrap               = "npm-shrinkwrap.json"
	nodeModules                 = "node_modules"
	defaultCompressedFileName   = "cache.tzst"
	workspaceCompressedFilename = "workspace.tzst"
)

// cache represents the files/dirs that will be cached
type cache struct {
	azureClient core.AzureClient
	logger      lumber.Logger
	once        sync.Once
	zstd        core.ZstdCompressor
	skipUpload  bool
	homeDir     string
}

var cacheBlobURL string
var apiErr error

// New returns a new CacheStore
func New(z core.ZstdCompressor, azureClient core.AzureClient, logger lumber.Logger) (core.CacheStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &cache{
		azureClient: azureClient,
		zstd:        z,
		logger:      logger,
		homeDir:     homeDir,
	}, nil
}

func (c *cache) getCacheSASURL(ctx context.Context, containerPath string) (string, error) {
	c.once.Do(func() {
		cacheBlobURL, apiErr = c.azureClient.GetSASURL(ctx, containerPath, core.CacheContainer)
	})
	return cacheBlobURL, apiErr
}

func (c *cache) Download(ctx context.Context, cacheKey string) error {
	containerPath := fmt.Sprintf("%s/%s", cacheKey, defaultCompressedFileName)
	sasURL, err := c.getCacheSASURL(ctx, containerPath)
	if err != nil {
		c.logger.Errorf("Error while generating SAS Token, error %v", err)
		return err
	}
	resp, err := c.azureClient.FindUsingSASUrl(ctx, sasURL)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			c.logger.Infof("Cache not found for key: %s", cacheKey)
			return nil
		}
		c.logger.Errorf("Error while downloading cache for key: %s, error %v", cacheKey, err)
		return err
	}
	c.skipUpload = true
	defer resp.Close()

	cachedFilePath := filepath.Join(os.TempDir(), defaultCompressedFileName)
	out, err := os.Create(cachedFilePath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp); err != nil {
		return err
	}
	//decompress
	return c.zstd.Decompress(ctx, cachedFilePath, true, global.RepoDir)

}

func (c *cache) Upload(ctx context.Context, cacheKey string, itemsToCompress ...string) error {
	if c.skipUpload {
		c.logger.Infof("Cache hit occurred on the key %s, not saving cache.", cacheKey)
		return nil
	}

	validatedItems := make([]string, 0, len(itemsToCompress))
	if len(itemsToCompress) == 0 {
		dir, err := c.getDefaultDirs()
		if err != nil {
			c.logger.Errorf("failed to get default cache directories, error %v", err)
			return nil
		}
		itemsToCompress = append(itemsToCompress, dir)
	}
	// validate the file or dir paths if it exists.
	for _, item := range itemsToCompress {
		exists, err := fileutils.CheckIfExists(item)
		if err != nil {
			return err
		}
		if exists {
			validatedItems = append(validatedItems, item)
		} else {
			c.logger.Debugf("%s does not exist, skipping upload", item)
		}
	}
	if len(validatedItems) == 0 {
		c.logger.Debugf("No valid files/dirs found to cache")
		return nil
	}

	err := c.zstd.Compress(ctx, defaultCompressedFileName, true, global.RepoDir, validatedItems...)
	if err != nil {
		c.logger.Errorf("error while compressing files with key %s, error: %v", cacheKey, err)
		return err
	}

	f, err := os.Open(filepath.Join(global.RepoDir, defaultCompressedFileName))
	if err != nil {
		c.logger.Errorf("error while opening compressed file with key %s, error: %v", cacheKey, err)
		return err
	}

	defer f.Close()
	containerPath := fmt.Sprintf("%s/%s", cacheKey, defaultCompressedFileName)
	sasURL, err := c.getCacheSASURL(ctx, containerPath)
	if err != nil {
		c.logger.Errorf("Error while generating SAS Token, error %v", err)
		return err
	}
	_, err = c.azureClient.CreateUsingSASURL(ctx, sasURL, f, "application/zstd")
	if err != nil {
		c.logger.Errorf("error while uploading cached file %s with key %s, error: %v", defaultCompressedFileName, cacheKey, err)
		return err
	}
	return nil
}

func (c *cache) CacheWorkspace(ctx context.Context) error {
	tmpDir := os.TempDir()
	if err := c.zstd.Compress(ctx, workspaceCompressedFilename, true, tmpDir, global.HomeDir); err != nil {
		return err
	}
	src := filepath.Join(tmpDir, workspaceCompressedFilename)
	dst := filepath.Join(global.WorkspaceCacheDir, workspaceCompressedFilename)
	if err := fileutils.CopyFile(src, dst, false); err != nil {
		return err
	}
	return nil
}

func (c *cache) ExtractWorkspace(ctx context.Context) error {
	tmpDir := os.TempDir()
	src := filepath.Join(global.WorkspaceCacheDir, workspaceCompressedFilename)
	dst := filepath.Join(tmpDir, workspaceCompressedFilename)
	if err := fileutils.CopyFile(src, dst, false); err != nil {
		return err
	}
	if err := c.zstd.Decompress(ctx, filepath.Join(tmpDir, workspaceCompressedFilename), true, global.HomeDir); err != nil {
		return err
	}
	return nil
}

func (c *cache) getDefaultDirs() (string, error) {
	f, err := os.Open(global.RepoDir)
	if err != nil {
		return "", err
	}

	dirs, err := f.ReadDir(-1)
	if err != nil {
		return "", err
	}

	for _, d := range dirs {
		// if yarn.lock present cache yarn folder
		if d.Name() == yarnLock {
			return filepath.Join(c.homeDir, ".cache", "yarn"), nil
		}
		// if package-lock.json or npm-shrinkwrap.json cache .npm cache
		if d.Name() == packageLock || d.Name() == npmShrinkwrap {
			return filepath.Join(c.homeDir, ".npm"), nil
		}
	}
	// If none present cache node_modules
	return nodeModules, nil
}
