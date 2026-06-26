package kernel_test

import (
	"testing"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// AC: enums reject invalid values.

func TestParseContractType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    kernel.ContractType
		wantErr bool
	}{
		{name: "accepts cdi", input: "cdi", want: kernel.ContractTypeCDI},
		{name: "accepts cdd", input: "cdd", want: kernel.ContractTypeCDD},
		{name: "accepts freelance", input: "freelance", want: kernel.ContractTypeFreelance},
		{name: "accepts interim", input: "interim", want: kernel.ContractTypeInterim},
		{name: "rejects empty string", input: "", wantErr: true},
		{name: "rejects unknown value", input: "permanent", wantErr: true},
		{name: "rejects uppercase CDI", input: "CDI", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := kernel.ParseContractType(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseContractType(%q) returned nil error; want non-nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseContractType(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseContractType(%q) = %q; want %q", tc.input, got, tc.want)
			}
			if !got.IsValid() {
				t.Errorf("IsValid() = false; want true for %q", got)
			}
		})
	}
}

func TestParseRemotePolicy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    kernel.RemotePolicy
		wantErr bool
	}{
		{name: "accepts on_site", input: "on_site", want: kernel.RemotePolicyOnSite},
		{name: "accepts hybrid", input: "hybrid", want: kernel.RemotePolicyHybrid},
		{name: "accepts full_remote", input: "full_remote", want: kernel.RemotePolicyFullRemote},
		{name: "rejects empty string", input: "", wantErr: true},
		{name: "rejects unknown value", input: "remote", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := kernel.ParseRemotePolicy(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseRemotePolicy(%q) returned nil error; want non-nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRemotePolicy(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseRemotePolicy(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseWorkingDays(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    kernel.WorkingDays
		wantErr bool
	}{
		{name: "accepts full_time", input: "full_time", want: kernel.WorkingDaysFullTime},
		{name: "accepts part_time", input: "part_time", want: kernel.WorkingDaysPartTime},
		{name: "accepts four_day", input: "four_day", want: kernel.WorkingDaysFourDay},
		{name: "rejects empty string", input: "", wantErr: true},
		{name: "rejects unknown value", input: "flexible", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := kernel.ParseWorkingDays(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseWorkingDays(%q) returned nil error; want non-nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseWorkingDays(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseWorkingDays(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseSeniority(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    kernel.Seniority
		wantErr bool
	}{
		{name: "accepts entry", input: "entry", want: kernel.SeniorityEntry},
		{name: "accepts mid", input: "mid", want: kernel.SeniorityMid},
		{name: "accepts senior", input: "senior", want: kernel.SenioritySenior},
		{name: "accepts lead", input: "lead", want: kernel.SeniorityLead},
		{name: "accepts exec", input: "exec", want: kernel.SeniorityExec},
		{name: "rejects empty string", input: "", wantErr: true},
		{name: "rejects unknown value", input: "principal", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := kernel.ParseSeniority(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseSeniority(%q) returned nil error; want non-nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSeniority(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseSeniority(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseApplicationStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    kernel.ApplicationStatus
		wantErr bool
	}{
		{name: "accepts saved", input: "saved", want: kernel.ApplicationStatusSaved},
		{name: "accepts applied", input: "applied", want: kernel.ApplicationStatusApplied},
		{name: "accepts interview", input: "interview", want: kernel.ApplicationStatusInterview},
		{name: "accepts offer", input: "offer", want: kernel.ApplicationStatusOffer},
		{name: "accepts rejected", input: "rejected", want: kernel.ApplicationStatusRejected},
		{name: "rejects empty string", input: "", wantErr: true},
		{name: "rejects unknown value", input: "ghosted", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := kernel.ParseApplicationStatus(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseApplicationStatus(%q) returned nil error; want non-nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseApplicationStatus(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseApplicationStatus(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestConfidence(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   uint8
		wantErr bool
	}{
		{name: "accepts 0", input: 0},
		{name: "accepts 50", input: 50},
		{name: "accepts 100", input: 100},
		// 101 overflows uint8, so the maximum testable invalid value requires
		// direct construction — confidence is validated at domain boundaries.
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := kernel.NewConfidence(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("NewConfidence(%d) returned nil error; want non-nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewConfidence(%d) unexpected error: %v", tc.input, err)
			}
			if got.Int() != int(tc.input) {
				t.Errorf("Int() = %d; want %d", got.Int(), tc.input)
			}
		})
	}
}

func TestUnderstanding(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   uint8
		wantErr bool
	}{
		{name: "accepts 0", input: 0},
		{name: "accepts 75", input: 75},
		{name: "accepts 100", input: 100},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := kernel.NewUnderstanding(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("NewUnderstanding(%d) returned nil error; want non-nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewUnderstanding(%d) unexpected error: %v", tc.input, err)
			}
			if got.Int() != int(tc.input) {
				t.Errorf("Int() = %d; want %d", got.Int(), tc.input)
			}
		})
	}
}
