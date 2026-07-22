package server

import (
	"context"
	"fmt"

	"github.com/vitaliiPsl/crappy-ai/internal/memory"
)

func (s *Server) ListMemories(ctx context.Context) ([]memory.Memory, error) {
	if s.memoryStore == nil {
		return nil, fmt.Errorf("memory store is not configured")
	}

	return s.memoryStore.List(ctx)
}

func (s *Server) CreateMemory(ctx context.Context, params memory.CreateParams) (memory.Memory, error) {
	if s.memoryStore == nil {
		return memory.Memory{}, fmt.Errorf("memory store is not configured")
	}

	return s.memoryStore.Create(ctx, params)
}

func (s *Server) UpdateMemory(ctx context.Context, params memory.UpdateParams) (memory.Memory, error) {
	if s.memoryStore == nil {
		return memory.Memory{}, fmt.Errorf("memory store is not configured")
	}

	return s.memoryStore.Update(ctx, params)
}

func (s *Server) DeleteMemory(ctx context.Context, id string) error {
	if s.memoryStore == nil {
		return fmt.Errorf("memory store is not configured")
	}

	return s.memoryStore.Delete(ctx, id)
}
