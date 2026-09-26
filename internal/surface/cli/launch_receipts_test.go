package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/release"
	"github.com/intentdriven/abcd/internal/gittest"
)

// committedReleaseWorkflow is this repository's own release.yml — the workflow
// whose receipt gate `launch receipts` must agree with.
func committedReleaseWorkflow(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRootForTest(t), filepath.FromSlash(lint.ReleaseWorkflowPath)))
	if err != nil {
		t.Fatalf("read the committed release workflow: %v", err)
	}
	return string(data)
}

// repoRootForTest is the module root, two directories above this package.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
}

// receiptsRepo is a release branch whose tip is the CHANGELOG roll, in a
// repository carrying abcd's own release workflow and a record-lint config.
func receiptsRepo(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(lint.ReleaseWorkflowPath, committedReleaseWorkflow(t))
	r.Write(".abcd/record-lint.json",
		`{"rules":{"receipt_gate":{"enabled":false,"severity":"blocker","receipts_dir":".abcd/work/reviews","required_gates":[]}}}`+"\n")
	r.Write("CHANGELOG.md", "## [Unreleased]\n")
	r.Commit("base")
	r.Git("switch", "-c", "release")
	r.Write("CHANGELOG.md", "## [Unreleased]\n\n## [1.0.0] - 2026-01-01\n")
	r.Commit("roll (content)")
	return r, r.Git("rev-parse", "HEAD")
}

func receiptJSON(gate, commit, verdict, detector string) string {
	return `{"subject":{"digest":{"gitCommit":"` + commit + `"}},"verificationResult":"` + verdict + `",` +
		`"policy":{"detector":"` + detector + `"},"judgeModel":"claude-opus-4-8"}` + "\n"
}

// TestLaunchReceiptsNamesEachMissingReceiptAndTheCommit is itd-93 AC7's
// first half, wired: on a release branch whose content commit has no receipts,
// the verb names every required gate's missing receipt and the commit it must
// name, and exits 1; once the second commit carries valid receipts it exits 0.
func TestLaunchReceiptsNamesEachMissingReceiptAndTheCommit(t *testing.T) {
	r, roll := receiptsRepo(t)

	out, err := shipIn(t, r, "launch", "receipts")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("exit = %d, want 1 (the gate refuses)\n%s", code, out)
	}
	for _, want := range []string{"REFUSED", roll, "docs-currency-reviewer", "iss35-brief-surface-crosscheck"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("render does not name %q:\n%s", want, out)
		}
	}

	dir := ".abcd/work/reviews/" + roll + "/"
	r.Write(dir+"docs-currency-reviewer.json", receiptJSON("docs-currency-reviewer", roll, "PROMOTE", "docs-currency-reviewer"))
	r.Write(dir+"iss35-brief-surface-crosscheck.json",
		receiptJSON("iss35-brief-surface-crosscheck", roll, "PROMOTE", "iss35-brief-surface-crosscheck"))
	r.Commit("receipts")

	out, err = shipIn(t, r, "launch", "receipts", "--json")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0 on a valid two-commit branch\n%s", code, out)
	}
	var check lint.ReceiptCheck
	if err := json.Unmarshal(out, &check); err != nil {
		t.Fatalf("--json: %v\n%s", err, out)
	}
	if !check.Pass || check.Commit != roll || !check.Derived {
		t.Errorf("want a derived pass keyed to the roll %s, got %+v", roll, check)
	}
}

// TestLaunchReceiptsFailsIdenticallyToTheReleaseJobsGate is itd-93 AC7's
// second half. It runs the release job's receipt step itself — the run script
// lifted verbatim out of this repository's committed release.yml, against a
// record-lint built from this tree — and `abcd launch receipts` over the same
// repository states, and holds them to one verdict and one set of reasons. The
// two share a reader by construction (lint.CheckReleaseReceipts calls the
// derivation, the arming and the check record-lint runs); this is the proof
// that no difference in how either front door reaches it has crept in.
func TestLaunchReceiptsFailsIdenticallyToTheReleaseJobsGate(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is required to run the release job's step")
	}
	recordLint := filepath.Join(t.TempDir(), "record-lint")
	build := exec.Command("go", "build", "-o", recordLint, "./cmd/record-lint")
	build.Dir = repoRootForTest(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build record-lint: %v\n%s", err, out)
	}
	script := strings.ReplaceAll(receiptStepScript(t, committedReleaseWorkflow(t)), "go run ./cmd/record-lint", recordLint)

	type state func(r *gittest.Repo, roll string)
	receipts := func(docs, cross string) state {
		return func(r *gittest.Repo, roll string) {
			dir := ".abcd/work/reviews/" + roll + "/"
			if docs != "" {
				r.Write(dir+"docs-currency-reviewer.json", docs)
			}
			if cross != "" {
				r.Write(dir+"iss35-brief-surface-crosscheck.json", cross)
			}
			r.Commit("receipts")
		}
	}
	cases := []struct {
		name  string
		state func(roll string) state
		pass  bool
	}{
		{"no receipts at all", func(string) state { return func(*gittest.Repo, string) {} }, false},
		{"one receipt missing", func(roll string) state {
			return receipts(receiptJSON("", roll, "PROMOTE", "docs-currency-reviewer"), "")
		}, false},
		{"one receipt HOLD", func(roll string) state {
			return receipts(receiptJSON("", roll, "PROMOTE", "docs-currency-reviewer"),
				receiptJSON("", roll, "HOLD", "iss35-brief-surface-crosscheck"))
		}, false},
		{"a receipt bound to the wrong detector", func(roll string) state {
			return receipts(receiptJSON("", roll, "PROMOTE", "docs-currency-reviewer"),
				receiptJSON("", roll, "PROMOTE", "docs-currency-reviewer"))
		}, false},
		{"every receipt PROMOTE", func(roll string) state {
			return receipts(receiptJSON("", roll, "PROMOTE", "docs-currency-reviewer"),
				receiptJSON("", roll, "PROMOTE", "iss35-brief-surface-crosscheck"))
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, roll := receiptsRepo(t)
			tc.state(roll)(r, roll)

			job := exec.Command("bash", "-c", script)
			job.Dir = r.Root()
			job.Env = append(r.Env(), "GITHUB_OUTPUT="+filepath.Join(t.TempDir(), "output"))
			jobOut, jobErr := job.CombinedOutput()
			jobPass := jobErr == nil

			out, err := shipIn(t, r, "launch", "receipts", "--json")
			code := exitCodeOf(err)
			if code != 0 && code != 1 {
				t.Fatalf("launch receipts exit = %d\n%s", code, out)
			}
			var check lint.ReceiptCheck
			if err := json.Unmarshal(out, &check); err != nil {
				t.Fatalf("--json: %v\n%s", err, out)
			}

			if jobPass != tc.pass || check.Pass != tc.pass || (code == 0) != tc.pass {
				t.Fatalf("verdicts disagree: release job pass=%v, launch receipts pass=%v (exit %d), want %v\nrelease job:\n%s\nlaunch receipts:\n%s",
					jobPass, check.Pass, code, tc.pass, jobOut, out)
			}
			// Same reasons: every refusal the local check names is one the
			// release job printed, and the job printed no receipt_gate finding
			// the local check left out.
			// When the derivation itself refuses, the release job stops there
			// and prints only that; the local check says the same thing first
			// and then names the receipts the roll still needs, which is what
			// the operator acts on.
			if !check.Derived {
				if check.DeriveError == "" || !strings.Contains(string(jobOut), check.DeriveError) {
					t.Errorf("the derivation refusal %q is not the one the release job printed:\n%s", check.DeriveError, jobOut)
				}
				return
			}
			for _, p := range check.Problems {
				if !strings.Contains(string(jobOut), p.Message) {
					t.Errorf("launch receipts names %q, which the release job did not print:\n%s", p.Message, jobOut)
				}
			}
			if n := strings.Count(string(jobOut), "receipt_gate]"); n != len(check.Problems) {
				t.Errorf("release job printed %d receipt_gate finding(s), launch receipts named %d\n%s\n%+v",
					n, len(check.Problems), jobOut, check.Problems)
			}
		})
	}
}

// receiptStepScript lifts the run script of release.yml's receipt step out of
// the workflow, dedented, exactly as the runner hands it to bash.
func receiptStepScript(t *testing.T, workflow string) string {
	t.Helper()
	lines := strings.Split(workflow, "\n")
	start := -1
	for i, l := range lines {
		if strings.Contains(l, "- name: Semantic-gate receipts") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatal("release.yml carries no Semantic-gate receipts step")
	}
	for i := start + 1; i < len(lines); i++ {
		l := lines[i]
		if strings.TrimSpace(l) != "run: |" {
			continue
		}
		indent := len(l) - len(strings.TrimLeft(l, " "))
		var body []string
		bodyIndent := -1
		for _, b := range lines[i+1:] {
			if strings.TrimSpace(b) == "" {
				body = append(body, "")
				continue
			}
			bi := len(b) - len(strings.TrimLeft(b, " "))
			if bi <= indent {
				break
			}
			if bodyIndent < 0 {
				bodyIndent = bi
			}
			body = append(body, b[bodyIndent:])
		}
		return strings.Join(body, "\n") + "\n"
	}
	t.Fatal("the Semantic-gate receipts step has no run block")
	return ""
}

// TestLaunchShipEmitEndsWithTheReceiptsProtocol is itd-93 AC8, wired: the
// emit step's render ends with the receipts protocol as a numbered checklist,
// and --json carries the same steps.
func TestLaunchShipEmitEndsWithTheReceiptsProtocol(t *testing.T) {
	r := shipReadyRepo(t)
	r.Write(lint.ReleaseWorkflowPath, committedReleaseWorkflow(t))
	r.Commit("the release workflow")

	out, err := shipIn(t, r, "launch", "ship")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	text := strings.TrimRight(string(out), "\n")
	head := strings.Index(text, "receipts protocol")
	if head < 0 {
		t.Fatalf("the emit render carries no receipts protocol:\n%s", text)
	}
	tail := text[head:]
	proto, err := release.ReceiptsProtocolFor(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	for i, step := range proto.Steps {
		if !strings.Contains(tail, "\n  "+strconv.Itoa(i+1)+". "+step) {
			t.Errorf("checklist step %d is not rendered numbered:\n%s", i+1, tail)
		}
	}
	lastLine := text[strings.LastIndex(text, "\n")+1:]
	if want := "  " + strconv.Itoa(len(proto.Steps)) + ". " + proto.Steps[len(proto.Steps)-1]; lastLine != want {
		t.Errorf("the render must END with the protocol's last step;\n got %q\nwant %q", lastLine, want)
	}
	for _, want := range []string{"docs-currency-reviewer", "exactly two commits", "abcd launch receipts"} {
		if !strings.Contains(tail, want) {
			t.Errorf("protocol render lacks %q:\n%s", want, tail)
		}
	}

	out, err = shipIn(t, r, "launch", "ship", "--json")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("--json exit = %d\n%s", code, out)
	}
	var got struct {
		Ready    bool                     `json:"ready"`
		Protocol release.ReceiptsProtocol `json:"receipts_protocol"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("--json: %v\n%s", err, out)
	}
	if !got.Ready || len(got.Protocol.Steps) != len(proto.Steps) {
		t.Errorf("--json must keep the cut's fields and carry the protocol; got %+v", got)
	}
}
