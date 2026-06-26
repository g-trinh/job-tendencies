package kernel

import "fmt"

// WorkingDays describes the weekly work schedule of a job listing.
type WorkingDays string

const (
	// WorkingDaysFullTime is a standard full-time schedule (temps plein).
	WorkingDaysFullTime WorkingDays = "full_time"
	// WorkingDaysPartTime is a part-time schedule.
	WorkingDaysPartTime WorkingDays = "part_time"
	// WorkingDaysFourDay is a four-day working week.
	WorkingDaysFourDay WorkingDays = "four_day"
)

var validWorkingDays = map[WorkingDays]bool{
	WorkingDaysFullTime: true,
	WorkingDaysPartTime: true,
	WorkingDaysFourDay:  true,
}

// ParseWorkingDays parses a WorkingDays value from a raw string, returning an error
// if the value is not recognised.
func ParseWorkingDays(s string) (WorkingDays, error) {
	wd := WorkingDays(s)
	if !validWorkingDays[wd] {
		return "", fmt.Errorf("unknown working days %q; valid values: full_time, part_time, four_day", s)
	}
	return wd, nil
}

// IsValid reports whether w is a known WorkingDays value.
func (w WorkingDays) IsValid() bool { return validWorkingDays[w] }
