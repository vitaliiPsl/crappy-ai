package session

import (
	"time"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type Session struct {
	ID    string `json:"id"`
	Title string `json:"title"`

	Usage kit.Usage `json:"usage"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
