package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
	"github.com/GustavoGutierrez/celador/internal/ports"
)

type clock interface{ Now() time.Time }

type FileCache struct {
	fs   ports.FileSystem
	root string
	clk  clock
}

func NewFileCache(fs ports.FileSystem, root string, clk clock) *FileCache {
	return &FileCache{fs: fs, root: root, clk: clk}
}

type osvEntry struct {
	Findings  []shared.Finding `json:"findings"`
	ExpiresAt time.Time        `json:"expiresAt"`
}

func (c *FileCache) GetScan(ctx context.Context, key string) (shared.ScanResult, bool, error) {
	path := filepath.Join(c.root, "scan-"+key+".json")
	body, err := c.fs.ReadFile(ctx, path)
	if err != nil {
		if ok, statErr := c.fs.Stat(ctx, path); statErr == nil && !ok {
			return shared.ScanResult{}, false, nil
		}
		return shared.ScanResult{}, false, err
	}
	var result shared.ScanResult
	if err := json.Unmarshal(body, &result); err != nil {
		return shared.ScanResult{}, false, err
	}
	return result, true, nil
}

func (c *FileCache) PutScan(ctx context.Context, key string, result shared.ScanResult) error {
	body, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return c.fs.WriteFile(ctx, filepath.Join(c.root, "scan-"+key+".json"), body)
}

func (c *FileCache) GetOSV(ctx context.Context, key string) ([]shared.Finding, bool, time.Time, error) {
	path := filepath.Join(c.root, "osv-"+key+".json")
	body, err := c.fs.ReadFile(ctx, path)
	if err != nil {
		if ok, statErr := c.fs.Stat(ctx, path); statErr == nil && !ok {
			return nil, false, time.Time{}, nil
		}
		return nil, false, time.Time{}, err
	}
	var entry osvEntry
	if err := json.Unmarshal(body, &entry); err != nil {
		return nil, false, time.Time{}, err
	}
	return entry.Findings, true, entry.ExpiresAt, nil
}

func (c *FileCache) PutOSV(ctx context.Context, key string, findings []shared.Finding, ttl time.Duration) error {
	entry := osvEntry{Findings: findings, ExpiresAt: c.clk.Now().Add(ttl)}
	body, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal osv cache: %w", err)
	}
	return c.fs.WriteFile(ctx, filepath.Join(c.root, "osv-"+key+".json"), body)
}
