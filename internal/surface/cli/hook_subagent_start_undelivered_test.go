package cli

import (
	"bytes"
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
