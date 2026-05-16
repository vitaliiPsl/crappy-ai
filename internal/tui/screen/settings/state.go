package settings

type state int

const (
	stateIdle state = iota
	stateDirty
	stateEditing
	statePickingModel
	stateSaving
	stateSaved
	stateFailed
)

func isDirtyState(s state) bool {
	return s == stateDirty || s == stateEditing || s == statePickingModel || s == stateSaving || s == stateFailed
}
