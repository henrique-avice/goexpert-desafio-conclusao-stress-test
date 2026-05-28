package entity

import "time"

// RequestResult represents the result of a single HTTP request
type RequestResult struct {
	StatusCode int
	Latency    time.Duration
	Error      error
	Timestamp  time.Time
}
