package profiles_test

import (
	"context"
	"errors"
	"testing"

	appprofiles "github.com/g-trinh/job-tendencies/internal/app/profiles"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	domainllm "github.com/g-trinh/job-tendencies/internal/domain/llm"
	"github.com/g-trinh/job-tendencies/internal/domain/profiles"
)

// fakeImportRepo is a minimal profiles.Repository fake for ImportIdentity tests.
// Only ProfileByID, UpdateIdentityFromImport, and UpdateIdentity are needed.
type fakeImportRepo struct {
	profile profiles.Profile
	getErr  error
	saveErr error
	saved   *importedIdentity
}

type importedIdentity struct {
	skills        []string
	seniority     kernel.Seniority
	rawExperience string
}

func (f *fakeImportRepo) ActiveProfile(_ context.Context) (profiles.Profile, error) {
	panic("not used")
}
func (f *fakeImportRepo) ProfileByID(_ context.Context, _ kernel.ProfileID) (profiles.Profile, error) {
	return f.profile, f.getErr
}
func (f *fakeImportRepo) List(_ context.Context) ([]profiles.Profile, error) {
	panic("not used")
}
func (f *fakeImportRepo) Create(_ context.Context, _ profiles.Profile) (kernel.ProfileID, error) {
	panic("not used")
}
func (f *fakeImportRepo) Update(_ context.Context, _ profiles.Profile) error {
	panic("not used")
}
func (f *fakeImportRepo) Delete(_ context.Context, _ kernel.ProfileID) error {
	panic("not used")
}
func (f *fakeImportRepo) Activate(_ context.Context, _ kernel.ProfileID) error {
	panic("not used")
}
func (f *fakeImportRepo) UpdateIdentity(_ context.Context, _ kernel.ProfileID, _ []string, _ kernel.Seniority) error {
	panic("not used")
}
func (f *fakeImportRepo) UpdateIdentityFromImport(_ context.Context, _ kernel.ProfileID, skills []string, seniority kernel.Seniority, rawExperience string) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = &importedIdentity{skills: skills, seniority: seniority, rawExperience: rawExperience}
	// Simulate the post-save read by updating the profile in-memory.
	f.profile.Skills = skills
	f.profile.Seniority = seniority
	f.profile.RawExperience = rawExperience
	return nil
}
func (f *fakeImportRepo) UpdateConditions(_ context.Context, _ kernel.ProfileID, _ profiles.ProfileConditions) error {
	panic("not used")
}
func (f *fakeImportRepo) UpdateWeights(_ context.Context, _ kernel.ProfileID, _ profiles.FitWeights) error {
	panic("not used")
}

// fakeExtractor is an in-memory implementation of appprofiles.IdentityExtractor.
type fakeExtractor struct {
	identity *domainllm.ExtractedIdentity
	err      error
}

func (f *fakeExtractor) ExtractIdentity(_ context.Context, _ []byte) (*domainllm.ExtractedIdentity, error) {
	return f.identity, f.err
}

// AC: ImportIdentity returns the populated profile when identity is empty.
// AC: ImportIdentity returns ErrConflict when identity is already populated.
// AC: ImportIdentity returns ErrNotFound when profile does not exist.

func TestImportIdentity_Guard(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		profile     profiles.Profile
		extractor   *fakeExtractor
		wantErr     error
		wantSkills  []string
		wantSenior  string
		wantExpBody string
	}{
		{
			name: "populates identity when profile has empty skills seniority and experience",
			profile: profiles.Profile{
				ID:            "p-1",
				Skills:        []string{},
				Seniority:     "",
				RawExperience: "",
			},
			extractor: &fakeExtractor{
				identity: &domainllm.ExtractedIdentity{
					Skills:        []string{"Go", "PostgreSQL"},
					RawExperience: "Engineer at Corp (2020-2024)",
					Seniority:     kernel.Seniority("senior"),
				},
			},
			wantSkills:  []string{"Go", "PostgreSQL"},
			wantSenior:  "senior",
			wantExpBody: "Engineer at Corp (2020-2024)",
		},
		{
			name: "returns ErrConflict when skills are already populated",
			profile: profiles.Profile{
				ID:     "p-1",
				Skills: []string{"Go"},
			},
			extractor: &fakeExtractor{},
			wantErr:   kernel.ErrConflict,
		},
		{
			name: "returns ErrConflict when seniority is already set",
			profile: profiles.Profile{
				ID:        "p-1",
				Skills:    []string{},
				Seniority: kernel.Seniority("senior"),
			},
			extractor: &fakeExtractor{},
			wantErr:   kernel.ErrConflict,
		},
		{
			name: "returns ErrConflict when raw experience is already set",
			profile: profiles.Profile{
				ID:            "p-1",
				Skills:        []string{},
				RawExperience: "some previous import",
			},
			extractor: &fakeExtractor{},
			wantErr:   kernel.ErrConflict,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeImportRepo{profile: tc.profile}
			svc := appprofiles.NewWithExtractor(repo, tc.extractor)

			got, err := svc.ImportIdentity(context.Background(), tc.profile.ID, []byte("%PDF fake"))

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("error = %v; want wrapping %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got.Skills) != len(tc.wantSkills) {
				t.Errorf("Skills = %v; want %v", got.Skills, tc.wantSkills)
			}
			if string(got.Seniority) != tc.wantSenior {
				t.Errorf("Seniority = %q; want %q", got.Seniority, tc.wantSenior)
			}
			if got.RawExperience != tc.wantExpBody {
				t.Errorf("RawExperience = %q; want %q", got.RawExperience, tc.wantExpBody)
			}
		})
	}
}

func TestImportIdentity_ProfileNotFound(t *testing.T) {
	t.Parallel()

	// AC: returns ErrNotFound when profile does not exist.
	repo := &fakeImportRepo{
		getErr: &kernel.NotFoundError{Kind: "profile", ID: "missing"},
	}
	svc := appprofiles.NewWithExtractor(repo, &fakeExtractor{})

	_, err := svc.ImportIdentity(context.Background(), "missing", []byte("%PDF"))
	if !errors.Is(err, kernel.ErrNotFound) {
		t.Errorf("error = %v; want wrapping ErrNotFound", err)
	}
}

func TestImportIdentity_ExtractorError(t *testing.T) {
	t.Parallel()

	// AC: LLM extraction errors are propagated to the caller.
	extractErr := errors.New("llm unavailable")
	repo := &fakeImportRepo{profile: profiles.Profile{ID: "p-1"}}
	svc := appprofiles.NewWithExtractor(repo, &fakeExtractor{err: extractErr})

	_, err := svc.ImportIdentity(context.Background(), "p-1", []byte("%PDF"))
	if err == nil {
		t.Fatal("expected error; got nil")
	}
	if !errors.Is(err, extractErr) {
		t.Errorf("error = %v; want wrapping %v", err, extractErr)
	}
}
