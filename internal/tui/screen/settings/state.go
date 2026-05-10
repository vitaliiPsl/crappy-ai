package settings

type state int

const (
	stateIdle state = iota
	stateDirty
	stateEditing
	stateSaving
	stateSaved
	stateFailed
)

func isDirtyState(s state) bool {
	return s == stateDirty || s == stateEditing || s == stateSaving || s == stateFailed
}
