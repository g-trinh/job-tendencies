// Package db provides the Cloud SQL pgx connection pool for Job Tendencies.
// All three binaries (api, scrape-worker, extract-worker) use this pool for
// database access. Authentication is passwordless IAM via Application Default
// Credentials — no password is ever stored in configuration or code.
//
// Local development:
//   - Ensure you are authenticated: gcloud auth application-default login
//   - Your Google account must have an IAM DB user in Cloud SQL (see NewPool for
//     the exact gcloud command to create it).
//   - Set CLOUD_SQL_INSTANCE, DB_IAM_USER, DB_NAME in your environment.
//
// In Cloud Run: the per-binary service account is the ADC identity; it has an
// IAM DB user created by the database OpenTofu module.
package db

import (
	"context"
	"fmt"
	"net"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool constructs a pgx connection pool that connects to Cloud SQL via the
// Cloud SQL Go connector using IAM (passwordless) authentication.
//
// # Parameters
//
//   - instanceConnName: Cloud SQL instance connection name
//     (format project:region:instance, e.g. job-tendencies-dev:europe-west9:jt-dev-pg)
//   - iamUser: IAM DB user name — the service account email for deployed workers,
//     or the developer's Google account email for local dev
//   - dbName: Postgres database name (e.g. job_tendencies)
//
// # Human developer prerequisite
//
// The human developer's Google account must exist as an IAM DB user in Cloud SQL.
// The Terraform module creates SA users only. Run the following once per developer:
//
//	gcloud sql users create DEVELOPER_EMAIL \
//	    --instance=jt-dev-pg \
//	    --project=job-tendencies-dev \
//	    --type=CLOUD_IAM_USER
//
//	gcloud projects add-iam-policy-binding job-tendencies-dev \
//	    --member="user:DEVELOPER_EMAIL" \
//	    --role="roles/cloudsql.client"
//
// Then authenticate locally:
//
//	gcloud auth application-default login
//
// # Returns
//
// The pool and a cleanup function that must be called on shutdown to close the pool
// and the underlying Cloud SQL dialer.
func NewPool(ctx context.Context, instanceConnName, iamUser, dbName string) (*pgxpool.Pool, func(), error) {
	if instanceConnName == "" {
		return nil, nil, fmt.Errorf("db pool: instanceConnName is required")
	}
	if iamUser == "" {
		return nil, nil, fmt.Errorf("db pool: iamUser is required")
	}
	if dbName == "" {
		return nil, nil, fmt.Errorf("db pool: dbName is required")
	}

	// Create a Cloud SQL dialer with IAM authentication enabled.
	// WithIAMAuthN injects an OAuth2 token (from ADC) as the Postgres password,
	// making the connection passwordless.
	dialer, err := cloudsqlconn.NewDialer(ctx, cloudsqlconn.WithIAMAuthN())
	if err != nil {
		return nil, nil, fmt.Errorf("creating cloud sql dialer: %w", err)
	}

	// sslmode=disable is intentional — the cloudsqlconn connector handles TLS itself.
	dsn := fmt.Sprintf("user=%s dbname=%s sslmode=disable", iamUser, dbName)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		_ = dialer.Close()
		return nil, nil, fmt.Errorf("parsing pgx pool config: %w", err)
	}

	// Substitute the Cloud SQL connector's Dial in place of the default TCP dialer
	// so all connections go through the managed connector (handles TLS, IAM token refresh).
	poolCfg.ConnConfig.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) {
		return dialer.Dial(ctx, instanceConnName)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		_ = dialer.Close()
		return nil, nil, fmt.Errorf("creating pgx pool for %q: %w", instanceConnName, err)
	}

	cleanup := func() {
		pool.Close()
		_ = dialer.Close()
	}

	return pool, cleanup, nil
}
