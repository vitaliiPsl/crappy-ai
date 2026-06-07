package background

import "context"

type Jobs struct {
	manager   *Manager
	sessionID string
}

func (m *Manager) ForSession(sessionID string) Jobs {
	return Jobs{
		manager:   m,
		sessionID: sessionID,
	}
}

func (j Jobs) Start(toolName string, run func(context.Context) (string, error)) (Job, error) {
	return j.manager.Start(j.sessionID, toolName, run)
}

func (j Jobs) Get(id string) (Job, error) {
	return j.manager.Get(j.sessionID, id)
}

func (j Jobs) List() ([]Job, error) {
	return j.manager.List(j.sessionID)
}

func (j Jobs) Wait(ctx context.Context, id string) (Job, error) {
	return j.manager.Wait(ctx, j.sessionID, id)
}

func (j Jobs) Cancel(id string) (Job, error) {
	return j.manager.Cancel(j.sessionID, id)
}
