package config

import (
	"fmt"
	"strconv"
	"time"
)

var StandardTimeZone = time.FixedZone("JST", 9*60*60)

func ParseTime(s string) (time.Time, error) {
	ts, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp string: %w", err)
	}
	return time.Unix(ts, 0).In(StandardTimeZone), nil
}

func FormatTime(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 16)
}
