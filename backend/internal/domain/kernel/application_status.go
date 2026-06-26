package kernel

import "fmt"

// ApplicationStatus tracks the user's progression through a job application kanban.
// Values are ordered by pipeline stage: Saved → Applied → Interview → Offer → Rejected.
type ApplicationStatus string

const (
	// ApplicationStatusSaved means the job is bookmarked but not yet applied to.
	ApplicationStatusSaved ApplicationStatus = "saved"
	// ApplicationStatusApplied means the application has been submitted.
	ApplicationStatusApplied ApplicationStatus = "applied"
	// ApplicationStatusInterview means an interview has been scheduled or completed.
	ApplicationStatusInterview ApplicationStatus = "interview"
	// ApplicationStatusOffer means an offer has been received.
	ApplicationStatusOffer ApplicationStatus = "offer"
	// ApplicationStatusRejected means the application was rejected at any stage.
	ApplicationStatusRejected ApplicationStatus = "rejected"
)

var validApplicationStatuses = map[ApplicationStatus]bool{
	ApplicationStatusSaved:     true,
	ApplicationStatusApplied:   true,
	ApplicationStatusInterview: true,
	ApplicationStatusOffer:     true,
	ApplicationStatusRejected:  true,
}

// ParseApplicationStatus parses an ApplicationStatus from a raw string, returning
// an error if the value is not recognised.
func ParseApplicationStatus(s string) (ApplicationStatus, error) {
	as := ApplicationStatus(s)
	if !validApplicationStatuses[as] {
		return "", fmt.Errorf(
			"unknown application status %q; valid values: saved, applied, interview, offer, rejected", s,
		)
	}
	return as, nil
}

// IsValid reports whether a is a known ApplicationStatus value.
func (a ApplicationStatus) IsValid() bool { return validApplicationStatuses[a] }
