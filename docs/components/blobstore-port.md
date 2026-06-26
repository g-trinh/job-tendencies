# Component: BlobStore Port

Domain port for storing and loading raw HTML/JSON payloads verbatim. GCS implementation in `infra/blobstore`.

## Interfaces

### BlobStore (= Storer + Loader)

```go
Store(ctx context.Context, path string, data []byte) error
Load(ctx context.Context, path string) ([]byte, error)
```

Store is idempotent: writing the same path twice overwrites. Load wraps `storage.ErrObjectNotExist` when path is missing.

## Notes

- Package path: `internal/domain/blobstore`
- Implementation: `internal/infra/blobstore.GCSBlobStore` (uses `cloud.google.com/go/storage`).
- Raw payloads are never translated; retained for re-extraction.
- Referenced from `raw_listing.raw_ref` in the data model.
