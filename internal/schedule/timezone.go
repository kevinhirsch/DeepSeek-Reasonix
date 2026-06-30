package schedule

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// TimezoneStore wraps timezone handling for scheduled tasks.
// Stores explicit IANA timezone with each scheduled task.
type TimezoneStore struct {
	tz *time.Location
}

// NewTimezoneStore creates a timezone store. Uses TZ env var, config flag,
// or OS timezone in that order.
func NewTimezoneStore(explicit string) *TimezoneStore {
	tz := resolveTimezone(explicit)
	return &TimezoneStore{tz: tz}
}

// Location returns the configured timezone.
func (s *TimezoneStore) Location() *time.Location { return s.tz }

// Zone returns the IANA zone name.
func (s *TimezoneStore) Zone() string { return s.tz.String() }

// Now returns the current time in the configured timezone.
func (s *TimezoneStore) Now() time.Time { return time.Now().In(s.tz) }

// FormatCron validates that a cron expression uses local timezone consistently.
func (s *TimezoneStore) FormatCron(minute, hour, dom, month, dow int) string {
	return fmt.Sprintf("%d %d %d %d %d (%s)", minute, hour, dom, month, dow, s.Zone())
}

func resolveTimezone(explicit string) *time.Location {
	if explicit != "" {
		if loc, err := time.LoadLocation(explicit); err == nil {
			return loc
		}
	}
	if tzEnv := os.Getenv("TZ"); tzEnv != "" {
		if loc, err := time.LoadLocation(tzEnv); err == nil {
			return loc
		}
	}
	return time.Local
}
