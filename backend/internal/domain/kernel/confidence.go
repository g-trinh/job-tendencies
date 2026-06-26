package kernel

import "fmt"

// Confidence is a per-field LLM extraction confidence score in the range [0, 100].
// A value of 0 means the field was absent or unrecognisable; 100 means maximum certainty.
// Scores are produced by the extraction model and stored on job.field_confidence.
type Confidence uint8

// NewConfidence constructs a Confidence value, returning an error if score > 100.
func NewConfidence(score uint8) (Confidence, error) {
	if score > 100 {
		return 0, fmt.Errorf("confidence score %d is out of range [0, 100]", score)
	}
	return Confidence(score), nil
}

// Int returns the confidence as an int in [0, 100].
func (c Confidence) Int() int { return int(c) }

// Understanding is the overall per-listing LLM parse quality score in the range [0, 100].
// It expresses how well the model understood the raw listing as a whole.
// Stored on job.understanding_score; surfaced as a badge and threshold filter.
type Understanding uint8

// NewUnderstanding constructs an Understanding value, returning an error if score > 100.
func NewUnderstanding(score uint8) (Understanding, error) {
	if score > 100 {
		return 0, fmt.Errorf("understanding score %d is out of range [0, 100]", score)
	}
	return Understanding(score), nil
}

// Int returns the understanding as an int in [0, 100].
func (u Understanding) Int() int { return int(u) }
