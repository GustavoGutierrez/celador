package ports

import "context"

type ReleaseVersionSource interface {
	Latest(ctx context.Context) (string, error)
}
