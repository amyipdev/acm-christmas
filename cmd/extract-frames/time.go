package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Time is a time in HH:MM:SS format.
type Time time.Duration

var timeRe = regexp.MustCompile(`(?:(\d+):)?(?:(\d+):)?(\d+)(?:\.(\d+))?$`)

// ParseTime parses a time in H:M:S, M:S or S format.
func ParseTime(v string) (Time, error) {
	if count := strings.Count(v, ":"); count < 2 {
		v = strings.Repeat("0:", 2-count) + v
	}

	var h, m, s int
	var trailing string
	n, err := fmt.Sscanf(v, "%d:%d:%d%s", &h, &m, &s, &trailing)
	if n < 3 && err != nil {
		return 0, fmt.Errorf("failed to parse time: %w", err)
	}

	d := 0 +
		time.Duration(h)*time.Hour +
		time.Duration(m)*time.Minute +
		time.Duration(s)*time.Second

	if trailing != "" {
		if !strings.HasPrefix(trailing, ".") {
			return 0, fmt.Errorf("invalid trailing seconds: %q", trailing)
		}

		frac, err := strconv.ParseFloat("0"+trailing, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse fractional seconds: %w", err)
		}
		d += time.Duration(frac * float64(time.Second))
	}

	return Time(d), nil
}

// MustParseTime parses a time in HH:MM:SS format. It panics if the time cannot
// be parsed.
func MustParseTime(v string) Time {
	t, err := ParseTime(v)
	if err != nil {
		panic(err)
	}
	return t
}

// AsDuration returns the time as a time.Duration.
func (t Time) AsDuration() time.Duration {
	return time.Duration(t)
}

// String implements fmt.Stringer.
func (t Time) String() string {
	h := int(t.AsDuration() / time.Hour)
	t -= Time(h) * Time(time.Hour)
	m := int(t.AsDuration() / time.Minute)
	t -= Time(m) * Time(time.Minute)
	s := int(t.AsDuration() / time.Second)
	t -= Time(s) * Time(time.Second)

	part := fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	if t != 0 {
		frac := fmt.Sprintf("%f", t.AsDuration().Seconds())
		part += strings.TrimPrefix(frac, "0")
	}

	return part
}

// Set sets the value of the flag given a string.
func (t *Time) Set(v string) error {
	new, err := ParseTime(v)
	if err != nil {
		return err
	}
	*t = new
	return nil
}

// Type implements pflag.Value.
func (t Time) Type() string {
	return "time"
}
