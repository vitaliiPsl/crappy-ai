package oauth

import "time"

type Limits struct {
	Plan      string
	Snapshots []LimitSnapshot
}

type LimitSnapshot struct {
	Name    string
	Windows []LimitWindow
}

type LimitWindow struct {
	UsedPercent float64
	Duration    time.Duration
	ResetsAt    time.Time
}
