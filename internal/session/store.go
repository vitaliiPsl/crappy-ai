package session

import "context"

type Store interface {
	Create(ctx context.Context, title, cwd string) (*Session, error)
	Save(ctx context.Context, session *Session) error
	Get(ctx context.Context, id string) (*Session, error)
	List(ctx context.Context) ([]*Session, error)
	Delete(ctx context.Context, id string) error

	AppendEvents(ctx context.Context, id string, events ...Event) error
	LoadEvents(ctx context.Context, id string) ([]Event, error)
}

type ArtifactStore interface {
	SaveArtifact(ctx context.Context, id, name string, value any) error
	LoadArtifact(ctx context.Context, id, name string, value any) (bool, error)
	ListArtifacts(ctx context.Context, id string) ([]string, error)
	DeleteArtifact(ctx context.Context, id, name string) error
}
