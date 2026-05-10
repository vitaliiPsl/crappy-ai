package session

import "context"

type Store interface {
	Create(ctx context.Context, title string) (*Session, error)
	Get(ctx context.Context, id string) (*Session, error)
	List(ctx context.Context) ([]*Session, error)
	Delete(ctx context.Context, id string) error
	Save(ctx context.Context, session *Session) error
	AppendEvents(ctx context.Context, id string, events ...Event) error
	LoadEvents(ctx context.Context, id string) ([]Event, error)
}
