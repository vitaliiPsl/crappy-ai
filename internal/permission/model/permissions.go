package model

import "slices"

type Rule struct {
	Tool    string `yaml:"tool" json:"tool"`
	Pattern string `yaml:"pattern,omitempty" json:"pattern,omitempty"`
}

type Permissions struct {
	Default Decision `yaml:"default,omitempty" json:"default,omitempty"`
	Deny    []Rule   `yaml:"deny,omitempty" json:"deny,omitempty"`
	Ask     []Rule   `yaml:"ask,omitempty" json:"ask,omitempty"`
	Allow   []Rule   `yaml:"allow,omitempty" json:"allow,omitempty"`
}

func (p *Permissions) Add(decision Decision, rule Rule) {
	switch decision {
	case Deny:
		p.Deny = appendRule(p.Deny, rule)
	case Ask:
		p.Ask = appendRule(p.Ask, rule)
	case Allow:
		p.Allow = appendRule(p.Allow, rule)
	}
}

func Merge(list ...Permissions) Permissions {
	var out Permissions
	for _, permissions := range list {
		if permissions.Default != "" {
			out.Default = permissions.Default
		}

		out.Deny = appendRules(out.Deny, permissions.Deny...)
		out.Ask = appendRules(out.Ask, permissions.Ask...)
		out.Allow = appendRules(out.Allow, permissions.Allow...)
	}

	return out
}

func appendRules(rules []Rule, next ...Rule) []Rule {
	for _, rule := range next {
		rules = appendRule(rules, rule)
	}

	return rules
}

func appendRule(rules []Rule, next Rule) []Rule {
	if slices.Contains(rules, next) {
		return rules
	}

	return append(rules, next)
}
