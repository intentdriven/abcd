package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/history"
)

// TestHistoryCaptureAllTakesTheWholeSession pins iss-2609202046145653: a run
// captures its own session in one call. `history capture --session <id> --all`
// finds, under the sources given, every transcript whose lines name that
// session — the main thread and each sub-agent — and stores them in this
// repository's store; a different session's transcript beside them is left
// alone.
func TestHistoryCaptureAllTakesTheWholeSession(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	t.Chdir(repo)
	src := t.TempDir()
	write := func(rel, session, agent string) {
		t.Helper()
		line := `{"type":"user","sessionId":"` + session + `","cwd":"` + repo + `"`
		if agent != "" {
			line += `,"agentId":"` + agent + `"`
		}
		line += `,"message":{"role":"user","content":"hello"}}` + "\n"
		p := filepath.Join(src, "proj", rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(line), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("sess-mine.jsonl", "sess-mine", "")
	write("sess-mine/subagents/agent-a1.jsonl", "sess-mine", "a1")
	write("sess-mine/subagents/agent-a2.jsonl", "sess-mine", "a2")
	write("sess-other.jsonl", "sess-other", "")

	out, errOut, err := runRecovery("", "history", "capture", "--session", "sess-mine", "--all", src)
	if err != nil {
		t.Fatalf("history capture --all: %v\n%s\n%s", err, out, errOut)
	}
	recs, err := history.List(repo, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, r := range recs {
		if r.SessionID != "sess-mine" {
			t.Errorf("a transcript of session %s was captured; --all names one session", r.SessionID)
		}
		got[r.AgentID] = true
	}
	for _, want := range []string{"", "a1", "a2"} {
		if !got[want] {
			t.Errorf("agent %q of the named session was not captured; records: %+v", want, recs)
		}
	}
}

// TestHistoryCaptureAllNeedsASession: --all is the whole of ONE named session,
// so without --session it is refused rather than read as every session.
func TestHistoryCaptureAllNeedsASession(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	t.Chdir(repo)
	_, errOut, err := runRecovery("", "history", "capture", "--all", t.TempDir())
	if err == nil {
		t.Fatal("history capture --all with no --session succeeded")
	}
	if !strings.Contains(err.Error()+errOut, "--session") {
		t.Errorf("the refusal must name --session, got %v / %s", err, errOut)
	}
}
