package model

type Decision string

const (
	Allow Decision = "allow"
	Deny  Decision = "deny"
	Ask   Decision = "ask"
)

type Scope string

const (
	ScopeOnce   Scope = "once"
	ScopeGlobal Scope = "global"
)
