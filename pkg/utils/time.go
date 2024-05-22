package utils

import "time"

func RoundToNearest5Minutes(t time.Time) time.Time {
	roundedMinutes := (t.Minute() / 5) * 5
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), roundedMinutes, 0, 0, t.Location())
}

// RemoveTimezone removes the timezone information from a time.Time object
// without changing the hour, minute, second, and nanosecond values.
func RemoveTimezone(t time.Time) time.Time {
	// Extract the year, month, day, hour, minute, second, and nanosecond
	year, month, day := t.Date()
	hour, minute, second := t.Clock()
	nanosecond := t.Nanosecond()

	// Create a new time.Time object without timezone (local time)
	localTime := time.Date(year, month, day, hour, minute, second, nanosecond, time.UTC)

	return localTime
}
