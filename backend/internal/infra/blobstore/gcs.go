// Package blobstore provides the GCS implementation of the domain blobstore port.
// Raw HTML/JSON captured from job boards is stored verbatim and never translated.
// Retained payloads allow re-extraction when extraction logic improves.
package blobstore

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
)

// GCSBlobStore implements domain/blobstore.BlobStore using Google Cloud Storage.
// Construct via NewGCSBlobStore at the composition root; the GCS client authenticates
// using Application Default Credentials.
type GCSBlobStore struct {
	bucket *storage.BucketHandle
}

// NewGCSBlobStore constructs a GCSBlobStore for the given bucket. The GCS client
// authenticates using Application Default Credentials; in Cloud Run this is the
// service account; locally it uses gcloud credentials.
func NewGCSBlobStore(ctx context.Context, bucketName string) (*GCSBlobStore, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("gcs blobstore: bucketName is required")
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating gcs client for bucket %q: %w", bucketName, err)
	}

	return &GCSBlobStore{bucket: client.Bucket(bucketName)}, nil
}

// Store writes data verbatim to the given GCS object path. Existing objects are
// overwritten (idempotent — writing the same raw listing twice is safe).
func (g *GCSBlobStore) Store(ctx context.Context, path string, data []byte) error {
	w := g.bucket.Object(path).NewWriter(ctx)
	if _, err := w.Write(data); err != nil {
		_ = w.Close()
		return fmt.Errorf("writing to gcs object %q: %w", path, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("closing gcs writer for object %q: %w", path, err)
	}
	return nil
}

// Load reads and returns the raw bytes stored at the given GCS object path.
// Returns a non-nil error wrapping storage.ErrObjectNotExist when the path does not exist.
func (g *GCSBlobStore) Load(ctx context.Context, path string) (data []byte, err error) {
	r, openErr := g.bucket.Object(path).NewReader(ctx)
	if openErr != nil {
		return nil, fmt.Errorf("opening gcs object %q: %w", path, openErr)
	}
	defer func() {
		if cerr := r.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing gcs reader for object %q: %w", path, cerr)
		}
	}()

	data, err = io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading gcs object %q: %w", path, err)
	}
	return data, nil
}
