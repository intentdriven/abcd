package findings

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

const issues = ".abcd/work/issues/"

// TestTheDefaultBranchDecidesWhatARecordSays is criterion 12's resolver: the
// default branch answers for every record it holds, and the checkout and the
// session's start copy answer only for a record it lacks.
func TestTheDefaultBranchDecidesWhatARecordSays(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(issues+"open/iss-2611250000000001-open-on-main.md", "x\n")
	repo.Write(issues+"resolved/iss-2611250000000002-resolved-on-main.md", "x\n")
	repo.Write(issues+"wontfix/iss-2611250000000003-wontfix-on-main.md", "x\n")
	repo.Write(".abcd/development/decisions/adrs/2611250000000009-an-adr.md", "x\n")
	repo.Write(".abcd/work/decisions/dcn-2611250000000010-a-decision.md", "x\n")
	repo.Write(".abcd/development/intents/planned/itd-2611250000000011-an-intent.md", "x\n")
	repo.Commit("records on main")
	repo.Git("checkout", "-q", "-b", "lane")
	// The lane's own tree: a record main resolved shown open again, and three
	// filed in this change.
	repo.Write(issues+"open/iss-2611250000000002-resolved-on-main.md", "x\n")
	repo.Write(issues+"open/iss-2611250000000004-open-here.md", "x\n")
	repo.Write(issues+"resolved/iss-2611250000000005-resolved-here.md", "x\n")
	repo.Write(issues+"wontfix/iss-2611250000000006-wontfix-here.md", "x\n")
	// A third worktree of the same repository, which is never asked.
	third := filepath.Join(t.TempDir(), "third")
	repo.Git("worktree", "add", "-q", "-b", "third", third, "main")
	if err := os.MkdirAll(filepath.Join(third, issues, "open"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(third, issues, "open", "iss-2611250000000007-third.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	start := t.TempDir()

	r := NewRecords(context.Background(), repo.Root(), start)
	for _, c := range []struct {
		act   Act
		id    string
		holds bool
		state string
	}{
		{ActCaptured, "iss-2611250000000001", true, "open"},
		{ActCaptured, "iss-2611250000000002", false, "resolved"},
		{ActCaptured, "iss-2611250000000003", false, "wontfix"},
		{ActCaptured, "iss-2611250000000004", true, "open"},
		{ActCaptured, "iss-2611250000000005", true, "resolved"},
		{ActCaptured, "iss-2611250000000006", false, "wontfix"},
		{ActCaptured, "iss-2611250000000007", false, "absent"},
		{ActCaptured, "adr-2611250000000009", false, "an ADR"},
		{ActSettled, "iss-2611250000000001", true, "open"},
		{ActSettled, "iss-2611250000000002", false, "resolved"},
		{ActSettled, "iss-2611250000000003", true, "wontfix"},
		{ActSettled, "iss-2611250000000004", true, "open"},
		{ActSettled, "iss-2611250000000005", false, "resolved"},
		{ActSettled, "iss-2611250000000007", false, "absent"},
		{ActSettled, "adr-2611250000000009", true, "an ADR"},
		{ActSettled, "dcn-2611250000000010", true, "a decision"},
		{ActSettled, "itd-2611250000000011", false, "an intent"},
	} {
		got, err := r.Answer(c.act, c.id)
		if err != nil {
			t.Errorf("%s %s: %v", c.act, c.id, err)
			continue
		}
		if got.Holds != c.holds || got.State != c.state {
			t.Errorf("%s %s = %+v; want holds %v, state %q", c.act, c.id, got, c.holds, c.state)
		}
	}
	if got, _ := r.Answer(ActCaptured, "iss-2611250000000002"); got.Where != "the default branch" {
		t.Errorf("a record main resolved is answered from %q; want the default branch", got.Where)
	}

	// The session's start copy answers too, for a record the default branch lacks.
	if err := os.MkdirAll(filepath.Join(start, issues, "open"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(start, issues, "open", "iss-2611250000000008-in-the-start-copy.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := NewRecords(context.Background(), repo.Root(), start).Answer(ActCaptured, "iss-2611250000000008"); err != nil || !got.Holds {
		t.Errorf("a record in the session's start copy alone = %+v, %v; want it to hold", got, err)
	}
}

func TestWithNoDefaultBranchTheCheckoutAnswersAlone(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(issues+"resolved/iss-2611250000000002-resolved-here.md", "x\n")
	repo.Commit("one record")
	repo.Git("branch", "-m", "main", "work")
	repo.Write(issues+"open/iss-2611250000000001-open-here.md", "x\n")
	r := NewRecords(context.Background(), repo.Root())
	for id, state := range map[string]string{"iss-2611250000000001": "open", "iss-2611250000000002": "resolved"} {
		got, err := r.Answer(ActCaptured, id)
		if err != nil || !got.Holds || got.State != state || got.Where != "the checkout" {
			t.Errorf("captured %s with no default branch = %+v, %v; want it held from the checkout as %s", id, got, err, state)
		}
	}
}

func TestARecordQuestionGitCannotAnswerIsAnError(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Commit("empty")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := NewRecords(ctx, repo.Root()).Answer(ActCaptured, "iss-2611250000000001"); err == nil {
		t.Errorf("a question asked past its deadline = %+v; want an error", got)
	}
	notARepo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(notARepo, issues, "open"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(notARepo, issues, "open", "iss-2611250000000001-x.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := NewRecords(context.Background(), notARepo).Answer(ActCaptured, "iss-2611250000000001"); err == nil {
		t.Errorf("a checkout git cannot read = %+v; want an error, not no default branch", got)
	}
}
