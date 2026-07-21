package memory

import (
	"context"
	"time"
)

type Kind string

const (
	KindProfile     Kind = "profile"
	KindPreference  Kind = "preference"
	KindInstruction Kind = "instruction"
)

func (k Kind) Valid() bool {
	switch k {
	case KindProfile, KindPreference, KindInstruction:
		return true
	default:
		return false
	}
}

type Memory struct {
	ID        string    `json:"id"`
	Kind      Kind      `json:"kind"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateParams struct {
	Kind    Kind
	Content string
}

type UpdateParams struct {
	ID      string
	Kind    Kind
	Content string
}

type Store interface {
	List(ctx context.Context) ([]Memory, error)
	Create(ctx context.Context, params CreateParams) (Memory, error)
	Update(ctx context.Context, params UpdateParams) (Memory, error)
	Delete(ctx context.Context, id string) error
}
