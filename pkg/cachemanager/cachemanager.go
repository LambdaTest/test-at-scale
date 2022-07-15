package cachemanager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/fileutils"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
)

const (
	pnpmLock                      = "pnpm-lock.yaml"
	yarnLock                      = "yarn.lock"
	packageLock                   = "package-lock.json"
	npmShrinkwrap                 = "npm-shrinkwrap.json"
	nodeModules                   = "node_modules"
	defaultCompressedFileName     = "cache.tzst"
	workspaceCompressedFilenameV1 = "workspace.tzst"
	workspaceCompressedFilenameV2 = "workspace-%s.tzst"
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

func (c *cache) getCacheSASURL(ctx context.Context, cacheKey string) (string, error) {
	c.once.Do(func() {
		query := map[string]interface{}{"key": cacheKey}
		cacheBlobURL, apiErr = c.azureClient.GetSASURL(ctx, core.PurposeCache, query)
	})
	return cacheBlobURL, apiErr
}

func (c *cache) Download(ctx context.Context, cacheKey string) error {
	sasURL, err := c.getCacheSASURL(ctx, cacheKey)
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
	return c.zstd.Decompress(ctx, cachedFilePath, true, global.RepoDir)
}

func (c *cache) Upload(ctx context.Context, cacheKey string, itemsToCompress ...string) error {
	if c.skipUpload {
		c.logger.Infof("Cache hit occurred on the key %s, not saving cache.", cacheKey)
		return nil
	}

	validatedItems := make([]string, 0, len(itemsToCompress))
	if len(itemsToCompress) == 0 {
		dirs, err := c.getDefaultDirs()
		c.logger.Debugf("Dirs: %+v", dirs)
		if err != nil {
			c.logger.Errorf("failed to get default cache directories, error %v", err)
			return nil
		}
		itemsToCompress = append(itemsToCompress, dirs...)
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
	sasURL, err := c.getCacheSASURL(ctx, cacheKey)
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

func (c *cache) CacheWorkspace(ctx context.Context, subModule string) error {
	tmpDir := os.TempDir()
	workspaceCompressedFilename := workspaceCompressedFilenameV1
	if subModule != "" {
		workspaceCompressedFilename = fmt.Sprintf(workspaceCompressedFilenameV2, subModule)
	}
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

func (c *cache) ExtractWorkspace(ctx context.Context, subModule string) error {
	tmpDir := os.TempDir()
	workspaceCompressedFilename := workspaceCompressedFilenameV1
	if subModule != "" {
		workspaceCompressedFilename = fmt.Sprintf(workspaceCompressedFilenameV2, subModule)
	}
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

func (c *cache) getDefaultDirs() ([]string, error) {
	defaultDirs := []string{}
	f, err := os.Open(global.RepoDir)
	if err != nil {
		return defaultDirs, err
	}

	dirs, err := f.ReadDir(-1)
	if err != nil {
		return defaultDirs, err
	}

	defaultDirs = append(defaultDirs, global.RepoCacheDir)
	for _, d := range dirs {
		// if yarn.lock present cache yarn folder
		if d.Name() == yarnLock {
			defaultDirs = append(defaultDirs, filepath.Join(c.homeDir, ".cache", "yarn"))
			return defaultDirs, nil
		}
		// if package-lock.json or npm-shrinkwrap.json cache .npm cache
		if d.Name() == packageLock || d.Name() == npmShrinkwrap {
			defaultDirs = append(defaultDirs, filepath.Join(c.homeDir, ".npm"))
			return defaultDirs, nil
		}
		// if pnmpm-lock.yaml is present, cache .pnpm-store cache
		if d.Name() == pnpmLock {
			defaultDirs = append(defaultDirs, filepath.Join(c.homeDir, ".local", "share", "pnpm", "store"))
			return defaultDirs, nil
		}
	}
	// If none present cache node_modules
	defaultDirs = append(defaultDirs, nodeModules)
	return defaultDirs, nil
}
