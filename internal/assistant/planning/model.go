package planning

import "fmt"

const (
	ArtifactName = "plan"

	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"

	maxItems = 12
)

type Status string

type Plan struct {
	Explanation string `json:"explanation,omitempty"`
	Items       []Item `json:"items"`
}

type Item struct {
	Step   string `json:"step"`
	Status Status `json:"status"`
}

func (p Plan) validate() error {
	if len(p.Items) == 0 {
		return fmt.Errorf("plan must include at least one item")
	}

	if len(p.Items) > maxItems {
		return fmt.Errorf("plan has %d items, max is %d", len(p.Items), maxItems)
	}

	for i, item := range p.Items {
		if item.Step == "" {
			return fmt.Errorf("plan item %d has empty step", i)
		}

		switch item.Status {
		case StatusPending, StatusInProgress, StatusCompleted:
		default:
			return fmt.Errorf("plan item %d has invalid status %q", i, item.Status)
		}
	}

	return nil
}
