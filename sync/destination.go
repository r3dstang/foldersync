package sync

import (
	"context"
	"io"
	"time"
)

// ObjectMeta holds metadata about a stored object.
type ObjectMeta struct {
	Size    int64
	ModTime time.Time
}

// Destination is a write target for synced files.
type Destination interface {
	// Put uploads a file to the destination at the given relative key.
	Put(ctx context.Context, key string, r io.Reader, size int64, modTime time.Time) error
	// Stat returns metadata for an existing object, or (nil, nil) if absent.
	Stat(ctx context.Context, key string) (*ObjectMeta, error)
	// List returns all keys currently held by the destination.
	List(ctx context.Context) ([]string, error)
	// Delete removes an object by key.
	Delete(ctx context.Context, key string) error
}
