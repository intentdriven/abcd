package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/rules"
)

// failingWriter refuses every write, as a host's closed pipe does.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("closed") }

// TestSubagentStartKeepsNoToldFactForAnAnswerNotDelivered: when the answer
// cannot be written, the sub-agent was told nothing, so no told fact stands
// for it, and its stop reads it as never told rather than told and unread.
func TestSubagentStartKeepsNoToldFactForAnAnswerNotDelivered(t *testing.T) {
	repo, _ := subagentRepo(t)
	cmd := newSubagentStartCommand()
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	var se bytes.Buffer
	cmd.SetOut(failingWriter{})
	cmd.SetErr(&se)
	cmd.SetIn(strings.NewReader(startHookPayload(t, "s1", repo.Root(), "a1", "t")))
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("an answer that could not be written succeeded; stderr %q", se.String())
	}
	if _, found, err := rules.ReadSubagentFact("s1", "a1", rules.SubagentTold); err != nil || found {
		t.Errorf("a told fact stands for an answer never delivered (found %v, err %v)", found, err)
	}
}

// TestSubagentStartWorkKeepsNoFact: the work beside the bound keeps no told
// fact itself, since the hook may give up on it at the bound, and a fact kept
// by work given up on says the sub-agent was told when nothing reached it. It
// hands the fact back, and only the hook keeps it, before it writes the answer.
func TestSubagentStartWorkKeepsNoFact(t *testing.T) {
	repo, _ := subagentRepo(t)
	in := hookInput{SessionID: "s1", AgentID: "a1", AgentType: "t", Cwd: repo.Root()}
	var notes bytes.Buffer
	doc, told, err := subagentStartMessage(context.Background(), in, &notes)
	if err != nil || len(doc) == 0 || told == nil {
		t.Fatalf("the work: doc %d bytes, told %v, err %v; want an answer and a fact to keep", len(doc), told, err)
	}
	if _, found, err := rules.ReadSubagentFact("s1", "a1", rules.SubagentTold); err != nil || found {
		t.Errorf("the work kept a told fact itself (found %v, err %v)", found, err)
	}
}
