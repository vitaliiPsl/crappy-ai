package model

import (
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
)

const (
	OptionAllowOnce    = "allow_once"
	OptionAllowExact   = "allow_exact"
	OptionAllowPattern = "allow_pattern"
	OptionDenyOnce     = "deny_once"
)

type Prompt struct {
	Call    kit.ToolCall
	Input   string
	Request ask.Request
	Options []Option
}

type Option struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Decision Decision `json:"decision"`
	Scope    Scope    `json:"scope"`
	Rule     *Rule    `json:"rule,omitempty"`
}

func NewPrompt(call kit.ToolCall, input string, suggested []Option) Prompt {
	options := []Option{allowOnceOption()}
	options = append(options, suggested...)
	options = append(options, denyOnceOption())

	return Prompt{
		Call:  call,
		Input: input,
		Request: ask.Request{
			ID:      call.ID,
			Title:   fmt.Sprintf("Allow %s?", toolLabel(call)),
			Detail:  input,
			Options: askOptions(options),
		},
		Options: options,
	}
}

func (r Prompt) Option(id string) (Option, bool) {
	for _, option := range r.Options {
		if option.ID == id {
			return option, true
		}
	}

	return Option{}, false
}

func allowOnceOption() Option {
	return Option{
		ID:       OptionAllowOnce,
		Label:    "Allow once",
		Decision: Allow,
		Scope:    ScopeOnce,
	}
}

func denyOnceOption() Option {
	return Option{
		ID:       OptionDenyOnce,
		Label:    "Deny",
		Decision: Deny,
		Scope:    ScopeOnce,
	}
}

func askOptions(options []Option) []ask.Option {
	out := make([]ask.Option, len(options))
	for i, option := range options {
		out[i] = ask.Option{
			ID:     option.ID,
			Label:  option.Label,
			Detail: optionDetail(option),
		}
	}

	return out
}

func optionDetail(option Option) string {
	if option.Rule == nil {
		return ""
	}

	return option.Rule.Pattern
}

func toolLabel(call kit.ToolCall) string {
	switch {
	case toolArg(call, "command") != "":
		return fmt.Sprintf("%s: $ %s", call.Name, toolArg(call, "command"))
	case toolArg(call, "path") != "":
		return fmt.Sprintf("%s: %s", call.Name, toolArg(call, "path"))
	case toolArg(call, "url") != "":
		return fmt.Sprintf("%s: %s", call.Name, toolArg(call, "url"))
	default:
		return call.Name
	}
}

func toolArg(call kit.ToolCall, key string) string {
	value, _ := call.Arguments[key].(string)

	return value
}
