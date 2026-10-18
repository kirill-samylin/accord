package github

import "time"

type Ordering string

const (
	Older         Ordering = "older"
	Same          Ordering = "same"
	Newer         Ordering = "newer"
	OrderConflict Ordering = "order_conflict"
)

// CompareVersions never uses arrival time, delivery ID, or suite ID as ordering.
// Distinct facts lacking comparable source timestamps conflict conservatively.
func CompareVersions(currentHash string, currentTime *time.Time, incomingHash string, incomingTime *time.Time) Ordering {
	if currentHash == incomingHash {
		return Same
	}
	if currentTime == nil || incomingTime == nil || currentTime.IsZero() || incomingTime.IsZero() {
		return OrderConflict
	}
	if incomingTime.Before(*currentTime) {
		return Older
	}
	if incomingTime.After(*currentTime) {
		return Newer
	}
	return OrderConflict
}
