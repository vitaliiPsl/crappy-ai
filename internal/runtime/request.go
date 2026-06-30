package runtime

// Request is the input to a turn. For now it is plain text; skills and other
// input kinds return later as capabilities.
type Request struct {
	Text string
}
