package session

import "context"

type CreateParams struct {
	Title    string
	Cwd      string
	ParentID string
}

type Store interface {
	Create(ctx context.Context, params CreateParams) (*Session, error)
	Save(ctx context.Context, session *Session) error
	Get(ctx context.Context, id string) (*Session, error)
	List(ctx context.Context) ([]*Session, error)
	Delete(ctx context.Context, id string) error

	AppendEvents(ctx context.Context, id string, events ...Event) error
	LoadEvents(ctx context.Context, id string) ([]Event, error)

	ArtifactStore
}

type ArtifactStore interface {
	SaveArtifact(ctx context.Context, id, name string, value any) error
	LoadArtifact(ctx context.Context, id, name string, value any) (bool, error)
	ListArtifacts(ctx context.Context, id string) ([]string, error)
	DeleteArtifact(ctx context.Context, id, name string) error
}
