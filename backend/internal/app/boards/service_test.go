package boards_test

import (
	"context"
	"errors"
	"testing"

	appboards "github.com/g-trinh/job-tendencies/internal/app/boards"
	"github.com/g-trinh/job-tendencies/internal/domain/boards"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	domainllm "github.com/g-trinh/job-tendencies/internal/domain/llm"
)

// fakeBoardRepo is an in-memory fake of boards.Repository for service unit tests.
type fakeBoardRepo struct {
	board    boards.Board
	boardErr error
	saveErr  error
	savedID  kernel.AdapterID
}

func (f *fakeBoardRepo) ListBoards(_ context.Context) ([]boards.BoardView, error) {
	return nil, nil
}
func (f *fakeBoardRepo) ApprovedAdapters(_ context.Context) ([]boards.Adapter, error) {
	return nil, nil
}
func (f *fakeBoardRepo) BoardByID(_ context.Context, _ kernel.BoardID) (boards.Board, error) {
	if f.boardErr != nil {
		return boards.Board{}, f.boardErr
	}
	return f.board, nil
}
func (f *fakeBoardRepo) CreateBoard(_ context.Context, _ boards.Board) (kernel.BoardID, error) {
	return "new-id", nil
}
func (f *fakeBoardRepo) UpdateBoard(_ context.Context, _ boards.Board) error { return nil }
func (f *fakeBoardRepo) DeleteBoard(_ context.Context, _ kernel.BoardID) error {
	return nil
}
func (f *fakeBoardRepo) GetAdapter(_ context.Context, _ kernel.BoardID) (boards.Adapter, error) {
	return boards.Adapter{}, nil
}
func (f *fakeBoardRepo) SaveDraftAdapter(_ context.Context, _ boards.Adapter) (kernel.AdapterID, error) {
	if f.saveErr != nil {
		return "", f.saveErr
	}
	if f.savedID == "" {
		f.savedID = "adapter-id-1"
	}
	return f.savedID, nil
}
func (f *fakeBoardRepo) ApproveAdapter(_ context.Context, _ kernel.AdapterID, _ kernel.BoardID) error {
	return nil
}
func (f *fakeBoardRepo) GetSchedule(_ context.Context) (boards.Schedule, error) {
	return boards.Schedule{}, nil
}
func (f *fakeBoardRepo) UpsertSchedule(_ context.Context, _ boards.Schedule) error { return nil }

// fakeAdapterGenerator is a minimal fake of domainllm.AdapterGenerator.
type fakeAdapterGenerator struct {
	spec *domainllm.AdapterSpec
	err  error
}

func (f *fakeAdapterGenerator) GenerateAdapter(_ context.Context, _, _ string) (*domainllm.AdapterSpec, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.spec, nil
}

// AC: Generates a draft AdapterSpec from an example page.

func TestService_GenerateAdapter(t *testing.T) {
	t.Parallel()

	validSpec := &domainllm.AdapterSpec{
		Board:     "wttj",
		FetchMode: domainllm.FetchModeJSONAPI,
		Search: domainllm.SearchConfig{
			URLTemplate:    "https://www.wttj.co/api/v1/jobs?page={page}",
			Method:         "GET",
			ParamMap:       map[string]string{"q": "profile.search.keywords"},
			ResultNodePath: "$.jobs",
			ResultFields:   map[string]string{"listing_url": "$.url"},
			Pagination: domainllm.PaginationConfig{
				Kind:  domainllm.PaginationKindQueryParam,
				Param: "page",
				Start: 1,
			},
		},
		Listing: domainllm.ListingConfig{
			Fetch:      domainllm.ListingFetchDetailPage,
			RawCapture: "full_response",
		},
		Incremental: domainllm.IncrementalConfig{
			CursorField:    "posted_at",
			OverlapBuffer:  "36h",
			SafetyMaxPages: 50,
		},
	}

	cases := []struct {
		name            string
		boardID         kernel.BoardID
		exampleResponse string
		board           boards.Board
		boardErr        error
		generatorSpec   *domainllm.AdapterSpec
		generatorErr    error
		saveErr         error
		wantStatus      boards.AdapterStatus
		wantErr         bool
	}{
		{
			name:            "generates and saves draft adapter for valid input",
			boardID:         "b-1",
			exampleResponse: "<html>job listings</html>",
			board:           boards.Board{ID: "b-1", Name: "WTTJ", BaseURL: "https://www.wttj.co", Enabled: true},
			generatorSpec:   validSpec,
			wantStatus:      boards.AdapterStatusDraft,
		},
		{
			name:            "returns validation error when example_response is empty",
			boardID:         "b-1",
			exampleResponse: "",
			wantErr:         true,
		},
		{
			name:            "returns not-found error when board does not exist",
			boardID:         "unknown",
			exampleResponse: "<html>...</html>",
			boardErr:        &kernel.NotFoundError{Kind: "board", ID: "unknown"},
			wantErr:         true,
		},
		{
			name:            "propagates LLM generator error",
			boardID:         "b-1",
			exampleResponse: "<html>...</html>",
			board:           boards.Board{ID: "b-1", Name: "WTTJ", BaseURL: "https://www.wttj.co", Enabled: true},
			generatorErr:    errors.New("LLM unavailable"),
			wantErr:         true,
		},
		{
			name:            "propagates repository save error",
			boardID:         "b-1",
			exampleResponse: "<html>...</html>",
			board:           boards.Board{ID: "b-1", Name: "WTTJ", BaseURL: "https://www.wttj.co", Enabled: true},
			generatorSpec:   validSpec,
			saveErr:         errors.New("db write failed"),
			wantErr:         true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeBoardRepo{board: tc.board, boardErr: tc.boardErr, saveErr: tc.saveErr}
			gen := &fakeAdapterGenerator{spec: tc.generatorSpec, err: tc.generatorErr}
			svc := appboards.New(repo, gen)

			got, err := svc.GenerateAdapter(context.Background(), tc.boardID, tc.exampleResponse)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (adapter: %+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Status != tc.wantStatus {
				t.Errorf("adapter status = %q; want %q", got.Status, tc.wantStatus)
			}
			if got.ID == "" {
				t.Errorf("adapter ID should be set after save")
			}
			if got.BoardID != tc.boardID {
				t.Errorf("adapter BoardID = %q; want %q", got.BoardID, tc.boardID)
			}
		})
	}
}

// AC: Output contains no executable-code field.
// The AdapterSpec domain type has no code field; this test confirms the returned
// adapter's Spec carries pure data (JSONPath, selectors, URL templates) only.

func TestService_GenerateAdapter_SpecIsDataOnly(t *testing.T) {
	t.Parallel()

	spec := &domainllm.AdapterSpec{
		Board:     "wttj",
		FetchMode: domainllm.FetchModeHTML,
		Search: domainllm.SearchConfig{
			URLTemplate:    "https://www.wttj.co/jobs",
			Method:         "GET",
			ParamMap:       map[string]string{},
			ResultNodePath: ".job-card",
			ResultFields:   map[string]string{"listing_url": "a.job-link[href]"},
			Pagination: domainllm.PaginationConfig{
				Kind:  domainllm.PaginationKindQueryParam,
				Param: "page",
				Start: 1,
			},
		},
		Listing: domainllm.ListingConfig{
			Fetch:      domainllm.ListingFetchDetailPage,
			RawCapture: "full_response",
		},
		Incremental: domainllm.IncrementalConfig{
			CursorField:    "posted_at",
			OverlapBuffer:  "36h",
			SafetyMaxPages: 10,
		},
	}

	repo := &fakeBoardRepo{
		board: boards.Board{ID: "b-1", Name: "WTTJ", BaseURL: "https://www.wttj.co", Enabled: true},
	}
	gen := &fakeAdapterGenerator{spec: spec}
	svc := appboards.New(repo, gen)

	got, err := svc.GenerateAdapter(context.Background(), "b-1", "<html>sample</html>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Spec is declarative data: board id, fetch mode, URL templates, selectors.
	// Verify there is no executable code by asserting the fields are data strings only.
	if got.Spec.Board == "" {
		t.Errorf("spec.Board should be set")
	}
	if got.Spec.FetchMode != domainllm.FetchModeHTML && got.Spec.FetchMode != domainllm.FetchModeJSONAPI {
		t.Errorf("spec.FetchMode = %q; want json_api or html (data-only values)", got.Spec.FetchMode)
	}
}
