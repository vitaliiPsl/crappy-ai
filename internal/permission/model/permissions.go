package model

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
		p.Deny = append(p.Deny, rule)
	case Ask:
		p.Ask = append(p.Ask, rule)
	case Allow:
		p.Allow = append(p.Allow, rule)
	}
}

func Merge(list ...Permissions) Permissions {
	var out Permissions
	for _, permissions := range list {
		if permissions.Default != "" {
			out.Default = permissions.Default
		}

		out.Deny = append(out.Deny, permissions.Deny...)
		out.Ask = append(out.Ask, permissions.Ask...)
		out.Allow = append(out.Allow, permissions.Allow...)
	}

	return out
}
