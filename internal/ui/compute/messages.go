package compute

// OpenLogsMsg is emitted when the user requests to view logs for a server.
// It carries the server ID to be used by the logs view.
type OpenLogsMsg struct {
	ServerID string
}

// GoBackMsg signals that the logs view should be closed and the UI should return to the previous view.
type GoBackMsg struct{}

// logChunkMsg carries a chunk of log content fetched from the server.
// If err is non-nil, the fetch failed.
type logChunkMsg struct {
	content string
	err     error
}

// logTickMsg is sent when the periodic ticker fires, indicating that logs should be refreshed.
type logTickMsg struct{}
