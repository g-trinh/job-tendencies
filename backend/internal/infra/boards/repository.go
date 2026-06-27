// Package boards provides the Postgres implementation of the board-manager
// repository. AdapterSpec is stored verbatim as JSONB in the adapter.spec column.
package boards

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/g-trinh/job-tendencies/internal/domain/boards"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/llm"
)

// Repository reads and writes boards and adapters in Postgres. It satisfies
// domain/boards.Repository. Construct via NewRepository at the composition root.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Postgres board repository over the given pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListBoards returns every board left-joined to its approved adapter.
func (r *Repository) ListBoards(ctx context.Context) ([]boards.BoardView, error) {
	const query = `
		SELECT b.id, b.name, b.base_url, b.enabled,
		       a.id, a.status, a.fetch_mode, a.spec, a.version
		FROM board b
		LEFT JOIN adapter a ON a.board_id = b.id AND a.status = 'approved'
		ORDER BY b.name`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying boards: %w", err)
	}
	defer rows.Close()

	var views []boards.BoardView
	for rows.Next() {
		var (
			b         boards.Board
			adapterID *string
			status    *string
			fetchMode *string
			specJSON  []byte
			version   *int
		)
		if err := rows.Scan(&b.ID, &b.Name, &b.BaseURL, &b.Enabled,
			&adapterID, &status, &fetchMode, &specJSON, &version); err != nil {
			return nil, fmt.Errorf("scanning board row: %w", err)
		}

		view := boards.BoardView{Board: b}
		if adapterID != nil {
			adapter, err := buildAdapter(b.ID, *adapterID, *status, *fetchMode, specJSON, *version)
			if err != nil {
				return nil, err
			}
			view.Adapter = &adapter
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating board rows: %w", err)
	}
	return views, nil
}

// ApprovedAdapters returns the approved adapter for every enabled board.
func (r *Repository) ApprovedAdapters(ctx context.Context) ([]boards.Adapter, error) {
	const query = `
		SELECT a.id, a.board_id, a.status, a.fetch_mode, a.spec, a.version
		FROM adapter a
		JOIN board b ON b.id = a.board_id
		WHERE a.status = 'approved' AND b.enabled = true
		ORDER BY a.board_id`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying approved adapters: %w", err)
	}
	defer rows.Close()

	var adapters []boards.Adapter
	for rows.Next() {
		var (
			id, boardID, status, fetchMode string
			specJSON                       []byte
			version                        int
		)
		if err := rows.Scan(&id, &boardID, &status, &fetchMode, &specJSON, &version); err != nil {
			return nil, fmt.Errorf("scanning adapter row: %w", err)
		}
		adapter, err := buildAdapter(kernel.BoardID(boardID), id, status, fetchMode, specJSON, version)
		if err != nil {
			return nil, err
		}
		adapters = append(adapters, adapter)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating adapter rows: %w", err)
	}
	return adapters, nil
}

// BoardByID returns one board or a kernel.NotFoundError.
func (r *Repository) BoardByID(ctx context.Context, id kernel.BoardID) (boards.Board, error) {
	const query = `SELECT id, name, base_url, enabled FROM board WHERE id = $1`
	var b boards.Board
	err := r.pool.QueryRow(ctx, query, string(id)).Scan(&b.ID, &b.Name, &b.BaseURL, &b.Enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return boards.Board{}, &kernel.NotFoundError{Kind: "board", ID: string(id)}
	}
	if err != nil {
		return boards.Board{}, fmt.Errorf("querying board %q: %w", id, err)
	}
	return b, nil
}

// CreateBoard inserts a new board and returns the assigned id.
func (r *Repository) CreateBoard(ctx context.Context, b boards.Board) (kernel.BoardID, error) {
	const query = `
		INSERT INTO board (name, base_url, enabled)
		VALUES ($1, $2, $3)
		RETURNING id`

	var id string
	err := r.pool.QueryRow(ctx, query, b.Name, b.BaseURL, b.Enabled).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("inserting board: %w", err)
	}
	return kernel.BoardID(id), nil
}

// UpdateBoard persists name, base_url, and enabled changes for the board.
func (r *Repository) UpdateBoard(ctx context.Context, b boards.Board) error {
	const query = `UPDATE board SET name = $1, base_url = $2, enabled = $3 WHERE id = $4`
	tag, err := r.pool.Exec(ctx, query, b.Name, b.BaseURL, b.Enabled, string(b.ID))
	if err != nil {
		return fmt.Errorf("updating board %q: %w", b.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return &kernel.NotFoundError{Kind: "board", ID: string(b.ID)}
	}
	return nil
}

// DeleteBoard removes a board by id.
func (r *Repository) DeleteBoard(ctx context.Context, id kernel.BoardID) error {
	const query = `DELETE FROM board WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, string(id))
	if err != nil {
		return fmt.Errorf("deleting board %q: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return &kernel.NotFoundError{Kind: "board", ID: string(id)}
	}
	return nil
}

// buildAdapter decodes the JSONB spec column into a domain Adapter.
func buildAdapter(boardID kernel.BoardID, id, status, fetchMode string, specJSON []byte, version int) (boards.Adapter, error) {
	var spec llm.AdapterSpec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return boards.Adapter{}, fmt.Errorf("unmarshalling adapter spec for %q: %w", id, err)
	}
	return boards.Adapter{
		ID:        kernel.AdapterID(id),
		BoardID:   boardID,
		Status:    boards.AdapterStatus(status),
		FetchMode: llm.FetchMode(fetchMode),
		Spec:      spec,
		Version:   version,
	}, nil
}

var _ boards.Repository = (*Repository)(nil)
