package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/gittest"
)

// sessionEndRepo builds an isolated git repo with one commit (so it has a
// root-commit SHA, the history store's key) and a hermetic ~/.abcd history store
// keyed on it. Returns the repo dir and its root SHA.
func sessionEndRepo(t *testing.T) (repo, rootSHA string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo = t.TempDir()
	env := append(gittest.Env(t),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e",
	)
	for _, args := range [][]string{
		{"init", "-q"},
		{"commit", "-q", "--allow-empty", "-m", "root"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	cmd := exec.Command("git", "rev-list", "--max-parents=0", "HEAD")
	cmd.Dir = repo
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-list: %v", err)
	}
	rootSHA = strings.TrimSpace(string(out))

	// Hermetic store: HOME drives ~/.abcd/transcripts/, and nothing is created
	// here. The store bootstraps itself on first use, so this harness is also
	// the "machine where `abcd ahoy install` never ran" case (iss-95): every
	// test built on it captures from a home holding nothing at all.
	t.Setenv("HOME", t.TempDir())
	return repo, rootSHA
}

// endPayload is the SessionEnd-hook JSON the harness writes to the verb's stdin.
func endPayload(t *testing.T, session, cwd, transcript string) string {
	t.Helper()
	b, err := json.Marshal(map[string]string{
		"session_id":      session,
		"cwd":             cwd,
		"transcript_path": transcript,
		"hook_event_name": "SessionEnd",
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// endThenStart runs the full capture: SessionEnd stages the transcript, the next
// SessionStart redacts and stores it. Capture is deliberately split across the
// two hooks (iss-2608230817034768) because redaction at exit loses the race with
// the host's shutdown cancellation, so no single hook proves the property any
// more — the pair does. Returns SessionStart's stderr, where a drain reports.
func endThenStart(t *testing.T, session, repo, transcript string) string {
	t.Helper()
	runHook(t, endPayload(t, session, repo, transcript), "hook", "session-end")
	// Not runHook: SessionStart exits non-zero on purpose when it has something
	// to say, because that is the only way the host renders a hook's stderr. A
	// drain failure is exactly such a notice, so the exit code is expected here
	// rather than a test failure.
	_, errlog, _ := runSessionStart(startPayload(session+"-next", repo), "hook", "session-start")
	return errlog
}

// TestHookSessionEndCapturesTranscript is the milestone: finishing a session
// leaves a record in the store. Without this the transcript corpus never
// accrues, and no later code can recover a session that was not captured while
// it ran. The work is split across SessionEnd (stage) and the next SessionStart
// (redact and store), so the test drives both.
func TestHookSessionEndCapturesTranscript(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	tp := filepath.Join(t.TempDir(), "sess.jsonl")
	if err := os.WriteFile(tp, []byte(`{"role":"user","text":"hello"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	errlog := endThenStart(t, "sess-1", repo, tp)

	recs, err := history.List(repo, rootSHA)
	if err != nil {
		t.Fatalf("history.List: %v (stderr: %s)", err, errlog)
	}
	if len(recs) != 1 {
		t.Fatalf("want 1 stored transcript, got %d (stderr: %s)", len(recs), errlog)
	}
	if recs[0].SessionID != "sess-1" {
		t.Errorf("session id = %q, want sess-1", recs[0].SessionID)
	}
}

// TestHookSessionEndIsIdempotent holds the re-capture property: the same
// transcript captured twice stores one record. A SessionEnd hook can fire more than
// once for a session, and the corpus must not grow a duplicate each time.
func TestHookSessionEndIsIdempotent(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	tp := filepath.Join(t.TempDir(), "sess.jsonl")
	if err := os.WriteFile(tp, []byte(`{"role":"user","text":"hello"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := endPayload(t, "sess-2", repo, tp)

	runHook(t, in, "hook", "session-end")
	runHook(t, in, "hook", "session-end")
	runHook(t, startPayload("sess-2-next", repo), "hook", "session-start")

	recs, err := history.List(repo, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("re-capture must be a no-op: want 1 record, got %d", len(recs))
	}
}

// TestHookSessionEndOneRecordPerSession is why this is wired to SessionEnd, not
// Stop. Stop fires once per assistant *turn*, and a live transcript grows
// between turns, so wiring Stop would store a fresh, larger superset every turn
// — Capture's sha256 dedup only collapses byte-identical re-captures. SessionEnd
// fires once at session termination. This test simulates a session that grew
// over several turns and asserts the store holds exactly one record for it.
func TestHookSessionEndOneRecordPerSession(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	tp := filepath.Join(t.TempDir(), "session.jsonl")

	// SessionEnd fires once, against the final, fully-grown transcript.
	body := ""
	for turn := 1; turn <= 5; turn++ {
		body += `{"role":"user","text":"turn ` + string(rune('0'+turn)) + `"}` + "\n"
	}
	if err := os.WriteFile(tp, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	endThenStart(t, "grown-session", repo, tp)

	recs, err := history.List(repo, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("one session must leave one record, got %d — is this wired to Stop (per-turn) instead of SessionEnd?", len(recs))
	}
}

// TestHookSessionEndNeverBlocksTheHost is the fail-closed contract. Every one of
// these is a payload the harness could plausibly hand us, and none may exit
// non-zero or write a record — a SessionEnd hook that errors or hangs wedges the
// user's session, which is a far worse outcome than a missed transcript.
func TestHookSessionEndNeverBlocksTheHost(t *testing.T) {
	cases := []struct {
		name  string
		stdin func(t *testing.T, repo string) string
	}{
		{"malformed json", func(*testing.T, string) string { return "{not json" }},
		{"empty payload", func(*testing.T, string) string { return "" }},
		{"no transcript_path", func(t *testing.T, repo string) string {
			return endPayload(t, "s", repo, "")
		}},
		{"transcript does not exist", func(t *testing.T, repo string) string {
			return endPayload(t, "s", repo, filepath.Join(t.TempDir(), "absent.jsonl"))
		}},
		{"transcript is a directory", func(t *testing.T, repo string) string {
			return endPayload(t, "s", repo, t.TempDir())
		}},
		{"hostile session id", func(t *testing.T, repo string) string {
			tp := filepath.Join(t.TempDir(), "s.jsonl")
			if err := os.WriteFile(tp, []byte("x\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			return endPayload(t, "../../escape", repo, tp)
		}},
		{"cwd is not a git repo", func(t *testing.T, _ string) string {
			tp := filepath.Join(t.TempDir(), "s.jsonl")
			if err := os.WriteFile(tp, []byte("x\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			return endPayload(t, "s", t.TempDir(), tp)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, rootSHA := sessionEndRepo(t)
			// runHook fails the test if the command exits non-zero.
			_, errlog := runHook(t, tc.stdin(t, repo), "hook", "session-end")

			if strings.TrimSpace(errlog) == "" {
				t.Error("a rejected payload must report its reason on stderr, got nothing")
			}
			recs, err := history.List(repo, rootSHA)
			if err != nil {
				t.Fatalf("history.List: %v", err)
			}
			if len(recs) != 0 {
				t.Errorf("a rejected payload must write nothing, got %d record(s)", len(recs))
			}
		})
	}
}

// TestHookSessionEndRedactsOnThisPath proves the redaction pass runs on the hook
// path itself, not just when `history capture` is called directly. Automatic
// capture makes this load-bearing: a secret or an absolute home path in a
// transcript must be masked before it lands in the store, and this asserts the
// hook — not some other caller — is what triggers that.
func TestHookSessionEndRedactsOnThisPath(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	home, err := os.UserHomeDir() // the hermetic HOME set by sessionEndRepo
	if err != nil {
		t.Fatal(err)
	}
	tp := filepath.Join(t.TempDir(), "sess.jsonl")
	// A ghp_ PAT-shaped token the bundled scanner matches (\bghp_[A-Za-z0-9]{36,}\b),
	// plus an absolute home path. The token is ASSEMBLED at runtime, never written
	// as a contiguous literal, so no committed source line carries a secret-shaped
	// string for the full-history secret scan to flag — the same discipline the
	// scanner's own fixtures use.
	token := "ghp_" + strings.Repeat("A", 40)
	body := `{"text":"token ` + token + ` and path ` + home + `/secret.env"}` + "\n"
	if err := os.WriteFile(tp, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	endThenStart(t, "redacts", repo, tp)

	recs, err := history.List(repo, rootSHA)
	if err != nil || len(recs) != 1 {
		t.Fatalf("want 1 record, got %d (err %v)", len(recs), err)
	}
	if recs[0].Secrets == 0 {
		t.Error("the GitHub token was not counted as redacted on the hook path")
	}
	if recs[0].HomePaths == 0 {
		t.Error("the absolute home path was not counted as redacted on the hook path")
	}
	_, stored, err := history.Read(repo, rootSHA, "redacts")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored), token) {
		t.Error("the raw GitHub token survived into the stored record")
	}
	if strings.Contains(string(stored), home+"/secret.env") {
		t.Error("the raw absolute home path survived into the stored record")
	}
}

// TestHistoryCaptureFromSubdirHonoursRepoPiiConfig proves capture resolves the
// git working-tree root, not the process cwd, before loading the per-repo
// redaction override. The scanner looks for .abcd/config/pii.json at the repo
// root only (no upward walk), so a capture run from a subdirectory that passed
// the subdirectory would silently redact with default patterns and let a
// custom-pattern secret land in the store in cleartext (B12).
func TestHistoryCaptureFromSubdirHonoursRepoPiiConfig(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)

	// A per-repo override adding a pattern the bundled defaults do NOT match.
	cfgDir := filepath.Join(repo, ".abcd", "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"patterns":{"acme_secret":{"regex":"ACME-WIDGET-[0-9]{6}","kind":"token","label":"acme secret","severity":"hard_fail"}}}`
	if err := os.WriteFile(filepath.Join(cfgDir, "pii.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run capture from a subdirectory of the repo.
	sub := filepath.Join(repo, "internal", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	token := "ACME-WIDGET-123456"
	tp := filepath.Join(t.TempDir(), "sess.jsonl")
	if err := os.WriteFile(tp, []byte(`{"text":"secret `+token+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	runCLI(t, "history", "capture", tp, "--session", "subdir-cfg")

	recs, err := history.List(repo, rootSHA)
	if err != nil || len(recs) != 1 {
		t.Fatalf("want 1 record, got %d (err %v)", len(recs), err)
	}
	if recs[0].Secrets == 0 {
		t.Error("the custom-pattern secret was not redacted — capture used the subdirectory, not the repo root, for pii.json (B12)")
	}
	_, stored, err := history.Read(repo, rootSHA, "subdir-cfg")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored), token) {
		t.Error("the raw custom-pattern secret survived into the stored record (B12)")
	}
}

// TestHookSessionEndWritesNothingToStdout keeps the SessionEnd hook silent on the
// model-facing stream. Diagnostics belong on stderr, out of band — the same rule
// the prompt-router follows.
func TestHookSessionEndWritesNothingToStdout(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	tp := filepath.Join(t.TempDir(), "sess.jsonl")
	if err := os.WriteFile(tp, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, _ := runHook(t, endPayload(t, "sess-3", repo, tp), "hook", "session-end")
	if stdout != "" {
		t.Errorf("session-end must produce zero stdout, got %q", stdout)
	}
}

// TestHookSessionEndDoesNotBlockOnIrregularFiles holds the no-hang contract for
// a transcript_path naming a FIFO or a device node. A plain O_RDONLY open of a
// FIFO with no writer blocks forever, and a hung SessionEnd hook wedges the
// session it is ending — readTranscript opens O_NONBLOCK precisely so this
// returns immediately and the non-regular-file check rejects it. The Execute
// runs in a goroutine so a regression hangs a 10-second timer, not the suite.
func TestHookSessionEndDoesNotBlockOnIrregularFiles(t *testing.T) {
	fifo := filepath.Join(t.TempDir(), "sess.fifo")
	if err := syscall.Mkfifo(fifo, 0o644); err != nil {
		t.Skipf("cannot create FIFO: %v", err)
	}
	for _, tc := range []struct {
		name, path string
	}{
		{"fifo with no writer", fifo},
		{"device node", os.DevNull},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, rootSHA := sessionEndRepo(t)
			in := endPayload(t, "s", repo, tc.path)

			type result struct {
				stdout, stderr string
				err            error
			}
			done := make(chan result, 1)
			go func() {
				cmd := NewRootCommand()
				var so, se bytes.Buffer
				cmd.SetOut(&so)
				cmd.SetErr(&se)
				cmd.SetIn(strings.NewReader(in))
				cmd.SetArgs([]string{"hook", "session-end"})
				err := cmd.Execute()
				done <- result{so.String(), se.String(), err}
			}()

			var res result
			select {
			case res = <-done:
			case <-time.After(10 * time.Second):
				t.Fatalf("session-end blocked opening %s — the O_NONBLOCK guard is gone", tc.name)
			}
			if res.err != nil {
				t.Fatalf("session-end exited non-zero on %s: %v\n%s", tc.name, res.err, res.stderr)
			}
			if strings.TrimSpace(res.stderr) == "" {
				t.Error("a rejected transcript must report its reason on stderr, got nothing")
			}
			recs, err := history.List(repo, rootSHA)
			if err != nil {
				t.Fatalf("history.List: %v", err)
			}
			if len(recs) != 0 {
				t.Errorf("a non-regular transcript must write nothing, got %d record(s)", len(recs))
			}
		})
	}
}

// TestHookSessionEndRefusesOverCapTranscript holds the 64 MiB cap: an over-cap
// transcript is refused whole (capturing a truncated prefix would break the
// sha256 idempotency key), the refusal is reported on stderr, and nothing is
// written. The file is sparse — the cap check reads Stat, not the bytes.
func TestHookSessionEndRefusesOverCapTranscript(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	tp := filepath.Join(t.TempDir(), "big.jsonl")
	if err := os.WriteFile(tp, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(tp, maxTranscriptBytes+1); err != nil {
		t.Fatal(err)
	}

	_, errlog := runHook(t, endPayload(t, "too-big", repo, tp), "hook", "session-end")

	// Match the distinctive cap phrasing, not the bare substring "cap": the
	// handler wraps every readTranscript error as "...; capturing nothing", and
	// "capturing" contains "cap", so a loose Contains(errlog, "cap") would pass
	// for ANY rejection reason — a false-confidence assertion.
	if !strings.Contains(errlog, "over the") || !strings.Contains(errlog, "cap") {
		t.Errorf("an over-cap transcript must report the size cap on stderr, got: %s", errlog)
	}
	recs, err := history.List(repo, rootSHA)
	if err != nil {
		t.Fatalf("history.List: %v", err)
	}
	if len(recs) != 0 {
		t.Errorf("an over-cap transcript must write nothing, got %d record(s)", len(recs))
	}
}

// TestHookSessionEndRefusesResidualHardFail holds the fail-closed tail of the
// two-stage redaction. If a hard_fail span SURVIVES stage-one masking, the
// stage-two re-scan refuses the write entirely — no record, reason reported,
// and neither hook exits non-zero. The custom pattern is built to survive on
// purpose: its regex matches both the raw token and the token's masked
// fingerprint (head 3 + starred middle + tail 2), so redaction cannot clear it.
//
// The refusal now lands on the DRAIN, not on SessionEnd, because SessionEnd no
// longer redacts. The guarantee is unchanged and so is its blast radius: what
// reaches transcripts/ is still redacted or absent. The staged raw copy is kept
// deliberately — it is the only copy abcd holds, and discarding it would turn a
// reported refusal into the silent permanent loss staging exists to end.
func TestHookSessionEndRefusesResidualHardFail(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	cfgDir := filepath.Join(repo, ".abcd", "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"patterns":{"sticky":{"regex":"ACM[A-Za-z0-9*]{13}Z9","kind":"token","label":"sticky token","severity":"hard_fail"}}}`
	if err := os.WriteFile(filepath.Join(cfgDir, "pii.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	// 18 runes: masked to "ACM" + 13 stars + "Z9", which the regex re-matches.
	token := "ACME" + strings.Repeat("Q", 12) + "Z9"
	tp := filepath.Join(t.TempDir(), "sess.jsonl")
	if err := os.WriteFile(tp, []byte(`{"text":"`+token+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	errlog := endThenStart(t, "residual", repo, tp)

	if !strings.Contains(errlog, "could not be stored") {
		t.Errorf("a surviving hard_fail span must be reported by the drain, got: %s", errlog)
	}
	recs, err := history.List(repo, rootSHA)
	if err != nil {
		t.Fatalf("history.List: %v", err)
	}
	if len(recs) != 0 {
		t.Errorf("a surviving hard_fail span must write NO file at all, got %d record(s)", len(recs))
	}
	// The raw copy survives a refusal, and the notice must name it so the
	// unredacted bytes are not left silently on disk.
	staged, err := history.ListStaged(repo, rootSHA)
	if err != nil {
		t.Fatalf("history.ListStaged: %v", err)
	}
	if len(staged) != 1 || staged[0].SessionID != "residual" {
		t.Fatalf("a refused capture must keep its staged copy, got %+v", staged)
	}
	if !strings.Contains(errlog, "unredacted") {
		t.Errorf("the notice must say the kept transcript is unredacted, got: %s", errlog)
	}
}

// TestHookSessionEndRefusesSymlinkedTranscript pins the O_NOFOLLOW half of the
// guarded read (iss-347): a symlink at transcript_path is refused, not
// followed, so a planted link cannot route a foreign file's bytes into the
// history store. Before the fsutil.ReadGuarded conversion the target was read
// and persisted.
func TestHookSessionEndRefusesSymlinkedTranscript(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "private.txt")
	if err := os.WriteFile(target, []byte("not a transcript\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "session.jsonl")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	_, errlog := runHook(t, endPayload(t, "sym", repo, link), "hook", "session-end")

	if !strings.Contains(errlog, "not a readable regular file") {
		t.Errorf("a symlinked transcript must be refused as non-regular, got: %s", errlog)
	}
	recs, err := history.List(repo, rootSHA)
	if err != nil {
		t.Fatalf("history.List: %v", err)
	}
	if len(recs) != 0 {
		t.Errorf("a symlinked transcript must write nothing, got %d record(s)", len(recs))
	}
}

// TestHookSessionStartDrainNoticeRedactsHomeInError holds the drain notice's
// error text to the same home redaction as its path: a gitleaks binary refusal
// quotes the configured path, and a $HOME-rooted one (a PATH-lookup result, a
// ~/.local/bin install) would otherwise print the developer's home directory.
func TestHookSessionStartDrainNoticeRedactsHomeInError(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	home := os.Getenv("HOME")
	// Present, outside the repo, correctly named, but NOT executable: refused,
	// with its $HOME-rooted spelling quoted in the error.
	bin := filepath.Join(home, ".local", "bin", "gitleaks")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("not executable"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(repo, ".abcd", "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"schema_version":1,"enabled":true,"path":"` + bin + `"}`
	if err := os.WriteFile(filepath.Join(cfgDir, "gitleaks.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	tp := filepath.Join(t.TempDir(), "sess.jsonl")
	if err := os.WriteFile(tp, []byte(`{"text":"hello"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	errlog := endThenStart(t, "homeleak", repo, tp)

	if !strings.Contains(errlog, "could not be stored") {
		t.Fatalf("the refused binary must be reported by the drain, got: %s", errlog)
	}
	if !strings.Contains(errlog, "gitleaks configured path refused") {
		t.Errorf("the notice must carry the refusal, got: %s", errlog)
	}
	if strings.Contains(errlog, home) {
		t.Errorf("the notice leaks the home directory %q:\n%s", home, errlog)
	}
	if !strings.Contains(errlog, "~/.local/bin/gitleaks") {
		t.Errorf("the notice must show the home-redacted path, got: %s", errlog)
	}
}

// TestHookSessionEndReStageKeepsNewerBytes is GHSA-xq36-hcgf-9wrj on the hook
// path: a second SessionEnd for the same session id carrying different bytes
// must replace the staged copy, so the next SessionStart stores the newer
// transcript, and the hook must say it re-staged rather than report a no-op.
// TestHookSessionEndIsIdempotent covers the identical-bytes case.
func TestHookSessionEndReStageKeepsNewerBytes(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	first := filepath.Join(t.TempDir(), "first.jsonl")
	second := filepath.Join(t.TempDir(), "second.jsonl")
	if err := os.WriteFile(first, []byte(`{"role":"user","text":"older snapshot"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte(`{"role":"user","text":"older snapshot"}`+"\n"+
		`{"role":"assistant","text":"NEWER-MARKER"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runHook(t, endPayload(t, "sess-restage", repo, first), "hook", "session-end")
	_, errlog := runHook(t, endPayload(t, "sess-restage", repo, second), "hook", "session-end")
	if strings.Contains(errlog, "no-op") || !strings.Contains(errlog, "re-staged") {
		t.Errorf("a re-stage with different bytes must say it replaced the staged copy, got: %q", errlog)
	}
	_, startLog, _ := runSessionStart(startPayload("sess-restage-next", repo), "hook", "session-start")

	_, stored, err := history.Read(repo, rootSHA, "sess-restage")
	if err != nil {
		t.Fatalf("history.Read: %v (session-start stderr: %s)", err, startLog)
	}
	if !strings.Contains(string(stored), "NEWER-MARKER") {
		t.Errorf("the store holds the older snapshot; the newer transcript is gone:\n%s", stored)
	}
}
