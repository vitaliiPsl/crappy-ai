package server

import "github.com/vitaliiPsl/crappy-ai/internal/background"

func (s *Server) BackgroundJobs() []background.Job {
	if s.background == nil {
		return nil
	}

	jobs, err := s.background.List()
	if err != nil {
		return nil
	}

	return jobs
}

func (s *Server) CancelBackgroundJob(id string) {
	if s.background == nil {
		return
	}

	_, _ = s.background.Cancel(id)
}
