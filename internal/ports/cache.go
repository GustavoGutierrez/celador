package ports

import (
	"context"
	"time"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

type ScanCache interface {
	GetScan(ctx context.Context, key string) (shared.ScanResult, bool, error)
	PutScan(ctx context.Context, key string, result shared.ScanResult) error
	GetOSV(ctx context.Context, key string) ([]shared.Finding, bool, time.Time, error)
	PutOSV(ctx context.Context, key string, findings []shared.Finding, ttl time.Duration) error
}
