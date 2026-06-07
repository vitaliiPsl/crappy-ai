package background

import (
	"errors"
	"time"
)

const (
	ArgName    = "background"
	ToolStatus = "job_status"
	ToolResult = "job_result"
	ToolCancel = "job_cancel"
	ToolList   = "job_list"
)

var ErrNotFound = errors.New("job not found")

type Status string

const (
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

type Job struct {
	ID          string     `json:"job_id"`
	SessionID   string     `json:"session_id,omitempty"`
	Tool        string     `json:"tool"`
	Status      Status     `json:"status"`
	Output      string     `json:"output,omitempty"`
	Error       string     `json:"error,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
