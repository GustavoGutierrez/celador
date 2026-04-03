package ports

import "context"

type NodeVersionDetector interface {
	Detect(ctx context.Context, root string) (string, bool)
}
