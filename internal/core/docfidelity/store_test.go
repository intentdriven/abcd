package docfidelity

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/surface"
)

var at = time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=T", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// armedRepo is a repository that ships the binary the gate judges: a surface
// snapshot, the brief's surfaces directory and one agent.
func armedRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	git(t, root, "init", "-q")
	write(t, root, SnapshotPath, "{}\n")
	write(t, root, ChaptersDir+"/06-capture.md", "### `abcd capture`\n")
	write(t, root, AgentsDir+"/scribe.md", "---\nname: scribe\n---\n")
	write(t, root, ChaptersDir+"/32-scribe.md", "The `scribe` agent.\n")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "c0")
	return root
}

var tree = []surface.Command{{Path: "abcd"}, {Path: "abcd capture"}}

func TestUnarmedRepositoryIsNotJudged(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q")
	if Armed(root) {
		t.Fatal("a repository with no surface snapshot is armed")
	}
	v, armed, err := Gate(root, tree, []string{"itd-1"}, false)
	if err != nil || armed || v.Refuse {
		t.Fatalf("unarmed gate: armed=%v refuse=%v err=%v", armed, v.Refuse, err)
	}
}

func TestReadInputsReadsChaptersAgentsAndBaseline(t *testing.T) {
	root := armedRepo(t)
	write(t, root, BaselinePath, `{"schema_version": 1, "reason": "pre-gate backlog", "surfaces": ["abcd rules"]}`)
	in, err := ReadInputs(root, tree, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(in.Chapters) != 2 || len(in.Agents) != 1 || in.Agents[0] != "scribe" || len(in.Baseline) != 1 {
		t.Fatalf("inputs: %d chapters, agents %v, baseline %v", len(in.Chapters), in.Agents, in.Baseline)
	}
	write(t, root, BaselinePath, `{"schema_version": 1, "surfaces": ["abcd rules"], "extra": 1}`)
	if _, err := ReadInputs(root, tree, nil); err == nil {
		t.Fatal("a baseline with an unknown field was accepted")
	}
}

func TestSavedReviewFoundAutomatically(t *testing.T) {
	root := armedRepo(t)
	// none
	v, _, err := Gate(root, tree, []string{"itd-1"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Refuse || v.Review == nil || v.Review.Status != ReviewNone || !strings.Contains(strings.Join(v.Reasons, "\n"), RunReviewFirst) {
		t.Fatalf("no receipt: %+v", v)
	}
	// match
	if _, _, err := Record(root, []byte(`{"verificationResult": "PROMOTE", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": []}`), at); err != nil {
		t.Fatal(err)
	}
	v, _, err = Gate(root, tree, []string{"itd-1"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if v.Refuse || v.Review.Status != ReviewMatch {
		t.Fatalf("matching receipt refused: %+v %+v", v.Reasons, v.Review)
	}
	// stale: the code moved on
	write(t, root, "main.go", "package main\n")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "c1")
	v, _, err = Gate(root, tree, []string{"itd-1"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Refuse || v.Review.Status != ReviewStale || !strings.Contains(strings.Join(v.Reasons, "\n"), RunReviewFirst) {
		t.Fatalf("stale receipt: %+v %+v", v.Reasons, v.Review)
	}
}

func TestSavedReviewRefusals(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		status  ReviewStatus
		want    string
	}{
		{"confirmed false sentence", `{"verificationResult": "HOLD", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": [
			{"doc": "brief", "chapter": "06-capture.md", "sentence": "capture prints YAML.", "evidence": "cli.go:1 prints JSON", "disposition": "confirmed"}]}`,
			ReviewHold, `"capture prints YAML."`},
		{"inconclusive", `{"verificationResult": "INCONCLUSIVE", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": []}`,
			ReviewInconclusive, RunReviewFirst},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := armedRepo(t)
			if _, _, err := Record(root, []byte(c.payload), at); err != nil {
				t.Fatal(err)
			}
			v, _, err := Gate(root, tree, []string{"itd-1"}, false)
			if err != nil {
				t.Fatal(err)
			}
			if !v.Refuse || v.Review.Status != c.status || !strings.Contains(strings.Join(v.Reasons, "\n"), c.want) {
				t.Fatalf("%+v %+v", v.Reasons, v.Review)
			}
		})
	}
	t.Run("malformed receipt on disk", func(t *testing.T) {
		root := armedRepo(t)
		head := git(t, root, "rev-parse", "HEAD")
		write(t, root, ReceiptsDir+"/"+head+"/"+GateName+".json", "{not json")
		v, _, err := Gate(root, tree, []string{"itd-1"}, false)
		if err != nil {
			t.Fatal(err)
		}
		if !v.Refuse || v.Review.Status != ReviewInvalid || !strings.Contains(strings.Join(v.Reasons, "\n"), RunReviewFirst) {
			t.Fatalf("%+v %+v", v.Reasons, v.Review)
		}
	})
}

func TestRecordRefusesAnUnusablePayload(t *testing.T) {
	for name, payload := range map[string]string{
		"unknown verdict":      `{"verificationResult": "MAYBE", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": []}`,
		"no judge":             `{"verificationResult": "PROMOTE", "judgeModel": "", "tier": "full", "failing": []}`,
		"HOLD naming nothing":  `{"verificationResult": "HOLD", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": []}`,
		"sentence missing":     `{"verificationResult": "HOLD", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": [{"doc": "brief", "chapter": "a.md", "sentence": "", "evidence": "e", "disposition": "confirmed"}]}`,
		"self-labelled commit": `{"verificationResult": "PROMOTE", "judgeModel": "claude-opus-5-5", "tier": "full", "failing": [], "subject": {}}`,
	} {
		t.Run(name, func(t *testing.T) {
			root := armedRepo(t)
			if _, _, err := Record(root, []byte(payload), at); err == nil {
				t.Fatal("recorded")
			}
			if _, err := os.Stat(filepath.Join(root, ReceiptsDir)); !os.IsNotExist(err) {
				t.Fatalf("a refused payload left a receipt directory: %v", err)
			}
		})
	}
}

// The boundary: the judgement and its reads write nothing; Record and Apply
// are the only paths that touch the tree.
func TestGateWritesNothing(t *testing.T) {
	root := armedRepo(t)
	before := git(t, root, "status", "--porcelain", "--untracked-files=all", "--ignored")
	for _, report := range []bool{false, true} {
		if _, _, err := Gate(root, tree, []string{"itd-1"}, report); err != nil {
			t.Fatal(err)
		}
	}
	if after := git(t, root, "status", "--porcelain", "--untracked-files=all", "--ignored"); after != before {
		t.Fatalf("the gate wrote to the tree:\nbefore %q\nafter  %q", before, after)
	}
}
