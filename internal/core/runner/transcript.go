package runner

// transcript.go is where a runner's transcript lands: abcd's own transcript
// store (internal/core/history), whatever the harness keeps of its own. The
// store redacts on write and refuses a transcript whose redaction leaves a
// blocking span, so what a harness echoed never reaches the record unredacted.

import "github.com/intentdriven/abcd/internal/core/history"

// TranscriptStore keeps one runner's transcript of one role's run.
type TranscriptStore interface {
	Store(runner string, req Request, ans Answer, raw []byte) error
}

// HistoryStore is the production store: the repository's lane of the
// user-level transcript store, keyed on its root commit.
type HistoryStore struct {
	RepoRoot string
	RootSHA  string
}

// Store captures raw as a native transcript produced by the runner, recorded
// under the harness's own session id when it reported one and the request's
// otherwise, with the role as its agent type.
func (h HistoryStore) Store(runner string, req Request, ans Answer, raw []byte) error {
	sid := ans.SessionID
	if !sessionRe.MatchString(sid) {
		sid = req.SessionID
	}
	_, err := history.Capture(h.RepoRoot, h.RootSHA, raw, history.CaptureMeta{
		SessionID: sid,
		Kind:      history.RouteNative,
		Tool:      runner,
		AgentType: req.Role,
	})
	return err
}
