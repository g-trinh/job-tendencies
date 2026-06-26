// Package blobstore defines the port for storing and loading raw payloads (HTML/JSON)
// captured verbatim from job boards. The GCS implementation lives in infra/blobstore.
// Raw payloads are never translated; they are retained so listings can be re-extracted
// when extraction logic improves.
package blobstore

import "context"

// Storer writes a raw payload to the blobstore at the given path.
// Implementations must be idempotent: writing the same path twice overwrites the first.
type Storer interface {
	Store(ctx context.Context, path string, data []byte) error
}

// Loader reads a raw payload from the blobstore at the given path.
type Loader interface {
	Load(ctx context.Context, path string) ([]byte, error)
}

// BlobStore combines Storer and Loader. The GCS adapter in infra/blobstore implements
// this interface.
type BlobStore interface {
	Storer
	Loader
}
