package server

import "github.com/vitaliiPsl/crappy-ai/internal/background"

func (s *Server) BackgroundJobs(sessionID string) []background.Job {
	if s.background == nil {
		return nil
	}

	jobs, err := s.background.List(sessionID)
	if err != nil {
		return nil
	}

	return jobs
}

func (s *Server) CancelBackgroundJob(sessionID, id string) {
	if s.background == nil {
		return
	}

	_, _ = s.background.Cancel(sessionID, id)
}
