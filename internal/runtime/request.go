package runtime

import "github.com/vitaliiPsl/crappy-ai/internal/session"

type Request struct {
	Text      string
	Skill     *SkillInvocation
	MCPPrompt *MCPPromptInvocation
}

type SkillInvocation struct {
	Name string
	Args []string
}

type MCPPromptInvocation struct {
	Server string
	Name   string
	Args   map[string]string
}

type QueuedRequest struct {
	ID      string
	Request Request
}

func queueSnapshot(queue []QueuedRequest) []session.QueuedRequest {
	out := make([]session.QueuedRequest, len(queue))
	for i, queued := range queue {
		req := session.Request{Text: queued.Request.Text}
		if queued.Request.Skill != nil {
			req.Skill = &session.SkillInvocation{
				Name: queued.Request.Skill.Name,
				Args: append([]string(nil), queued.Request.Skill.Args...),
			}
		}

		if queued.Request.MCPPrompt != nil {
			req.MCPPrompt = &session.MCPPromptInvocation{
				Server: queued.Request.MCPPrompt.Server,
				Name:   queued.Request.MCPPrompt.Name,
				Args:   queued.Request.MCPPrompt.Args,
			}
		}

		out[i] = session.QueuedRequest{ID: queued.ID, Request: req}
	}

	return out
}
