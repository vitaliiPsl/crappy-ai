package settings

type ClosedMsg struct{}

type savedMsg struct {
	err error
}

type oauthMsg struct {
	status string
	err    error
}
