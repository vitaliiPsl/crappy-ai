package model

import "github.com/vitaliiPsl/crappy-adk/kit"

const (
	OptionAllowOnce    = "allow_once"
	OptionAllowExact   = "allow_exact"
	OptionAllowPattern = "allow_pattern"
	OptionDenyOnce     = "deny_once"
)

type AskRequest struct {
	Call    kit.ToolCall `json:"call"`
	Input   string       `json:"input,omitempty"`
	Options []AskOption  `json:"options"`
}

type AskOption struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Decision Decision `json:"decision"`
	Scope    Scope    `json:"scope"`
	Rule     *Rule    `json:"rule,omitempty"`
}

type AskResponse struct {
	OptionID string `json:"option_id"`
}

func NewAskRequest(call kit.ToolCall, input string, suggested []AskOption) AskRequest {
	options := []AskOption{allowOnceOption()}
	options = append(options, suggested...)
	options = append(options, denyOnceOption())

	return AskRequest{
		Call:    call,
		Input:   input,
		Options: options,
	}
}

func (r AskRequest) Option(id string) (AskOption, bool) {
	for _, option := range r.Options {
		if option.ID == id {
			return option, true
		}
	}

	return AskOption{}, false
}

func allowOnceOption() AskOption {
	return AskOption{
		ID:       OptionAllowOnce,
		Label:    "Allow once",
		Decision: Allow,
		Scope:    ScopeOnce,
	}
}

func denyOnceOption() AskOption {
	return AskOption{
		ID:       OptionDenyOnce,
		Label:    "Deny",
		Decision: Deny,
		Scope:    ScopeOnce,
	}
}
