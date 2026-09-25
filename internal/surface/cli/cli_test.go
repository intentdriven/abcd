package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestVersionJSON proves the CLI -> core -> JSON round-trip the Phase 0 exit
// criterion requires.
func TestVersionJSON(t *testing.T) {
	out := runCLI(t, "version", "--json")

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if got["name"] != "abcd" {
		t.Fatalf("name = %v, want abcd", got["name"])
	}
	if got["version"] == "" || got["version"] == nil {
		t.Fatalf("version missing: %v", got)
	}
}

func TestVersionText(t *testing.T) {
	out := runCLI(t, "version")
	if !strings.HasPrefix(string(out), "abcd ") {
		t.Fatalf("text output = %q, want it to start with \"abcd \"", out)
	}
}

func TestBareStatusJSON(t *testing.T) {
	out := runCLI(t, "--json")
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("bare status output is not JSON: %v\n%s", err, out)
	}
	if _, ok := got["dir"]; !ok {
		t.Fatalf("status JSON missing dir: %v", got)
	}
}

func TestRulesBareText(t *testing.T) {
	out := string(runCLI(t, "rules"))
	for _, want := range []string{"COMMITTING", "PII"} {
		if !strings.Contains(out, want) {
			t.Fatalf("bare `rules` missing %q:\n%s", want, out)
		}
	}
}

func TestRulesBareJSON(t *testing.T) {
	out := runCLI(t, "rules", "--json")
	var got struct {
		Disabled bool             `json:"disabled"`
		Domains  []map[string]any `json:"domains"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("rules --json not JSON: %v\n%s", err, out)
	}
	if len(got.Domains) == 0 {
		t.Fatalf("rules --json returned no domains: %s", out)
	}
	found := false
	for _, d := range got.Domains {
		if d["name"] == "COMMITTING" {
			found = true
		}
	}
	if !found {
		t.Fatalf("rules --json missing COMMITTING: %s", out)
	}
}

func TestRulesScopedUppercasesArg(t *testing.T) {
	out := string(runCLI(t, "rules", "committing"))
	if !strings.Contains(out, "COMMITTING") {
		t.Fatalf("scoped `rules committing` missing COMMITTING:\n%s", out)
	}
	if strings.Contains(out, "## PII") {
		t.Fatalf("scoped render leaked another domain:\n%s", out)
	}
}

func TestRulesUnknownDomainErrors(t *testing.T) {
	if _, err := runCLIErr(t, "rules", "nosuch"); err == nil {
		t.Fatal("unknown domain must exit non-zero")
	}
}

// runHook executes a hook entrypoint with separated streams: stdout is the
// model-facing context (must be empty on a no-match), stderr is the out-of-band
// diagnostic. Hooks exit 0, so an error is a test failure.
func runHook(t *testing.T, stdin string, args ...string) (stdout, stderr string) {
	t.Helper()
	cmd := NewRootCommand()
	var so, se bytes.Buffer
	cmd.SetOut(&so)
	cmd.SetErr(&se)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook %v exited non-zero: %v\n%s", args, err, se.String())
	}
	return so.String(), se.String()
}

func hookInputJSON(t *testing.T, session, cwd, prompt string) string {
	t.Helper()
	b, err := json.Marshal(map[string]string{
		"session_id": session, "cwd": cwd, "prompt": prompt,
		"hook_event_name": "UserPromptSubmit",
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestHookPromptRouterInjects(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	cwd := t.TempDir() // no rules.json -> bundled defaults
	out, errlog := runHook(t, hookInputJSON(t, "s1", cwd, "commit and push"), "hook", "prompt-router")
	if !strings.Contains(out, "COMMITTING") {
		t.Fatalf("prompt-router did not inject COMMITTING into context:\n%s", out)
	}
	if !strings.Contains(errlog, "injected 1 domain") {
		t.Fatalf("missing out-of-band diagnostic:\n%s", errlog)
	}
}

func TestHookPromptRouterDedups(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	cwd := t.TempDir()
	in := hookInputJSON(t, "s2", cwd, "commit and push")
	_, _ = runHook(t, in, "hook", "prompt-router")
	out2, _ := runHook(t, in, "hook", "prompt-router")
	if out2 != "" {
		t.Fatalf("second turn re-injected context (dedup failed):\n%s", out2)
	}
}

func TestHookResetReinjects(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	cwd := t.TempDir()
	in := hookInputJSON(t, "s3", cwd, "commit and push")
	_, _ = runHook(t, in, "hook", "prompt-router")
	_, _ = runHook(t, `{"session_id":"s3","hook_event_name":"SessionStart","source":"compact"}`, "hook", "prompt-router-reset")
	out, _ := runHook(t, in, "hook", "prompt-router")
	if !strings.Contains(out, "COMMITTING") {
		t.Fatalf("post-reset turn did not re-inject:\n%s", out)
	}
}

func TestHookNoMatchZeroStdout(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	cwd := t.TempDir()
	out, _ := runHook(t, hookInputJSON(t, "s4", cwd, "paint a landscape"), "hook", "prompt-router")
	if out != "" {
		t.Fatalf("no-match must produce zero model-facing stdout, got %q", out)
	}
}

func TestHookMalformedStdinExitsZero(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	// A malformed hook payload must not error the process (non-blocking) and must
	// inject nothing.
	out, err := runCLIErr(t, "hook", "prompt-router")
	if err != nil {
		t.Fatalf("malformed/empty stdin should exit 0, got %v", err)
	}
	_ = out
}

// TestHookPromptRouterLoadErrorHasSinglePrefix pins iss-2608261550491547: a
// rules-load failure must read once, not twice. rules.Load already prefixes its
// errors with "rules:", so the hook wrapper must not add a second "rules:" of its
// own — the user must never see "abcd rules: rules: …".
func TestHookPromptRouterLoadErrorHasSinglePrefix(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	cwd := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cwd, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Malformed JSON drives rules.Load down its fail-closed error path.
	if err := os.WriteFile(filepath.Join(cwd, ".abcd", "rules.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, errlog := runHook(t, hookInputJSON(t, "s-prefix", cwd, "commit and push"), "hook", "prompt-router")
	if strings.Contains(errlog, "rules: rules:") {
		t.Fatalf("doubled prefix: the load error must read once, got:\n%s", errlog)
	}
	if !strings.Contains(errlog, "injecting nothing") {
		t.Fatalf("a load error must still say it injected nothing:\n%s", errlog)
	}
}

// validHooksJSON is a structurally-sound plugin hook manifest for the hermetic
// plugin root, so the install path's hook-manifest verification passes.
const validHooksJSON = `{
  "hooks": {
    "UserPromptSubmit": [{"hooks": [{"type": "command", "command": "\"$CLAUDE_PLUGIN_ROOT/abcd\" hook prompt-router"}]}],
    "SessionStart":     [{"hooks": [{"type": "command", "command": "\"$CLAUDE_PLUGIN_ROOT/abcd\" hook prompt-router-reset"}]}],
    "PreCompact":       [{"hooks": [{"type": "command", "command": "\"$CLAUDE_PLUGIN_ROOT/abcd\" hook prompt-router-reset"}]}]
  }
}`

// hermeticRepo redirects HOME, the plugin root and the PATH symlink target to
// temp locations, chdirs into a fresh adoptable repo, and returns its path.
func hermeticRepo(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	pluginRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(pluginRoot, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "hooks", "hooks.json"), []byte(validHooksJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "abcd"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", pluginRoot)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	// The harness's settings resolve under HOME once this is empty, so a
	// machine that names its own configuration directory never offers its real
	// status line to a hermetic install (spc-70).
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("ABCD_BIN_TARGET", filepath.Join(t.TempDir(), "bin", "abcd"))

	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	return repo
}

// TestAhoyInstallWiredAndIdempotent proves `abcd ahoy install` reaches the core
// engine from the CLI front door (the Phase 1 install milestone) and that a
// re-run is an exact no-op.
func TestAhoyInstallWiredAndIdempotent(t *testing.T) {
	repo := hermeticRepo(t)

	out := runCLI(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")
	var res struct {
		Status string   `json:"status"`
		Writes []string `json:"writes"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("install output not JSON: %v\n%s", err, out)
	}
	if res.Status != "clean" {
		t.Fatalf("install status = %q, want clean\n%s", res.Status, out)
	}
	// The marker block reached disk via the CLI path.
	body, err := os.ReadFile(filepath.Join(repo, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("CLAUDE.md not written: %v", err)
	}
	if !strings.Contains(string(body), "<!-- BEGIN ABCD -->") {
		t.Fatalf("CLAUDE.md has no marker block:\n%s", body)
	}

	// Second run is an exact no-op.
	out2 := runCLI(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")
	var res2 struct {
		Status string   `json:"status"`
		Writes []string `json:"writes"`
	}
	if err := json.Unmarshal(out2, &res2); err != nil {
		t.Fatalf("re-install output not JSON: %v\n%s", err, out2)
	}
	if res2.Status != "already_up_to_date" {
		t.Fatalf("re-install status = %q, want already_up_to_date", res2.Status)
	}
	if len(res2.Writes) != 0 {
		t.Fatalf("re-install wrote files: %v", res2.Writes)
	}
}

// TestAhoyInstallExplicitOverrideAppliesAsUpdate proves the iss-107 fix end to
// end through the CLI: an explicit --visibility on an already-configured repo is
// applied-as-update (not silently no-op'd by the already_up_to_date short
// circuit), the change is echoed, and it reaches config.json.
func TestAhoyInstallExplicitOverrideAppliesAsUpdate(t *testing.T) {
	repo := hermeticRepo(t)

	// First install pins visibility=private and reaches a clean state.
	runCLI(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")

	// Re-install with an explicit --visibility public: must NOT no-op.
	out := runCLI(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "public", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--json")
	var res struct {
		Status  string   `json:"status"`
		Changes []string `json:"changes"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("re-install output not JSON: %v\n%s", err, out)
	}
	if res.Status == "already_up_to_date" {
		t.Fatalf("explicit override silently no-op'd: status=%q\n%s", res.Status, out)
	}
	if len(res.Changes) == 0 {
		t.Fatalf("no change echoed for an explicit override:\n%s", out)
	}
	body, err := os.ReadFile(filepath.Join(repo, ".abcd", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"visibility": "public"`) {
		t.Fatalf("visibility override not persisted to config.json:\n%s", body)
	}
}

// TestAhoyInstallDocsTargetNarrowingRetractsOrphan proves the apply-as-update
// leaves no orphan: narrowing docs-target from both to claude_md via an explicit
// override removes the now-de-selected AGENTS.md marker block (iss-107).
func TestAhoyInstallDocsTargetNarrowingRetractsOrphan(t *testing.T) {
	repo := hermeticRepo(t)

	runCLI(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")
	agents := filepath.Join(repo, "AGENTS.md")
	if body, err := os.ReadFile(agents); err != nil || !strings.Contains(string(body), "<!-- BEGIN ABCD -->") {
		t.Fatalf("precondition: AGENTS.md must have a marker block after both-target install (err=%v)", err)
	}

	runCLI(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "claude_md",
		"--oracle-backend", "host-delegated", "--json")

	body, err := os.ReadFile(agents)
	if err != nil {
		t.Fatalf("AGENTS.md read: %v", err)
	}
	if strings.Contains(string(body), "<!-- BEGIN ABCD -->") {
		t.Fatalf("orphaned AGENTS.md marker block not retracted after narrowing docs-target:\n%s", body)
	}
	if claude, err := os.ReadFile(filepath.Join(repo, "CLAUDE.md")); err != nil || !strings.Contains(string(claude), "<!-- BEGIN ABCD -->") {
		t.Fatalf("CLAUDE.md marker block must survive the narrowing (err=%v)", err)
	}
}

func runCLI(t *testing.T, args ...string) []byte {
	t.Helper()
	return runCLIStdin(t, "", args...)
}

// runCLIStdin runs the CLI with stdin bound to `stdin`, so commands that read a
// payload from "-" (e.g. `history capture -`) can be exercised end-to-end.
func runCLIStdin(t *testing.T, stdin string, args ...string) []byte {
	t.Helper()
	out, err := runCLIStdinErr(t, stdin, args...)
	if err != nil {
		t.Fatalf("execute %v: %v\n%s", args, err, out)
	}
	return out
}

// gitCommit is gitCmd with a commit identity pinned.
//
// gittest.Env deliberately pins NO identity — its doc says a caller that needs
// one supplies it — so a bare `git commit` resolves whatever the machine
// happens to carry. That is an ambient dependency: it passes on any developer
// machine with a system or global user.name, and fails with "Author identity
// unknown" on a CI runner that has neither. The sibling fixture in
// internal/core/reading pins one; this surface did not, and eleven tests failed
// the first time they ever ran in CI (iss-2609010759382400).
func gitCommit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	ident := []string{"-c", "user.name=abcd test", "-c", "user.email=test@example.invalid"}
	return gitCmd(t, repo, append(ident, args...)...)
}

func gitCmd(t *testing.T, repo string, args ...string) string {
	t.Helper()
	full := append([]string{"-C", repo}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = gittest.Env(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// TestHistoryListEmptyStoreIsJSONArray pins that `history list --json` on a repo
// with nothing stored emits an empty array, not bare `null` — the command doc
// promises "an empty list", and a consumer that iterates the value must get [].
func TestHistoryListEmptyStoreIsJSONArray(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := t.TempDir()
	gitCmd(t, repo, "init")
	gitCmd(t, repo, "config", "user.email", "test@example.com")
	gitCmd(t, repo, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repo, "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, repo, "add", ".")
	gitCommit(t, repo, "commit", "-m", "init")
	t.Chdir(repo)

	out := strings.TrimSpace(string(runCLI(t, "history", "list", "--json")))
	if out != "[]" {
		t.Fatalf("history list --json on an empty store = %q, want %q", out, "[]")
	}
}

// TestHistoryCaptureWiredAndRedacts proves the Finding-B wiring: `abcd history
// capture` reaches history.Capture from the CLI front door, redacts a planted
// secret, and stores the record on disk. Before this change history.Capture was
// dead scaffolding — no CLI subverb reached it.
func TestHistoryCaptureWiredAndRedacts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := t.TempDir()
	gitCmd(t, repo, "init")
	gitCmd(t, repo, "config", "user.email", "test@example.com")
	gitCmd(t, repo, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repo, "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, repo, "add", ".")
	gitCommit(t, repo, "commit", "-m", "init")
	t.Chdir(repo)

	// Create the store dir exactly as `abcd ahoy install` would (Capture never
	// bootstraps it).
	rootSHA := gitCmd(t, repo, "rev-list", "--max-parents=0", "HEAD")
	tdir := filepath.Join(home, ".abcd", "history", rootSHA, "transcripts")
	if err := os.MkdirAll(tdir, 0o755); err != nil {
		t.Fatal(err)
	}

	pat := "ghp_" + strings.Repeat("b", 40)
	transcript := "user: deploy with token " + pat + "\nassistant: done\n"

	out := runCLIStdin(t, transcript, "history", "capture", "--session", "sess-wired", "--json")

	var res struct {
		Record struct {
			Path      string `json:"path"`
			SessionID string `json:"session_id"`
			Secrets   int    `json:"redacted_secrets"`
		} `json:"record"`
		Wrote bool `json:"wrote"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("capture output not JSON: %v\n%s", err, out)
	}
	if !res.Wrote {
		t.Fatalf("expected Wrote=true on first capture\n%s", out)
	}
	if res.Record.SessionID != "sess-wired" {
		t.Errorf("session id = %q, want sess-wired", res.Record.SessionID)
	}
	if res.Record.Secrets < 1 {
		t.Errorf("expected >=1 secret redaction counted, got %d", res.Record.Secrets)
	}
	// The success envelope's path is home-redacted: it must not carry the
	// absolute home root (a developer-identity leak in machine output), and it
	// renders with the ~ shorthand instead.
	if home, herr := os.UserHomeDir(); herr == nil && home != "" {
		if strings.Contains(res.Record.Path, home) {
			t.Errorf("stored path leaked the absolute home root: %q", res.Record.Path)
		}
	}
	if !strings.HasPrefix(res.Record.Path, "~") {
		t.Errorf("stored path = %q, want a ~-rooted (home-redacted) path", res.Record.Path)
	}
	// Expand the ~ shorthand back to an absolute path to read the file — the
	// shell does this for a user; a programmatic consumer reads by session id.
	readPath := res.Record.Path
	if home, herr := os.UserHomeDir(); herr == nil && strings.HasPrefix(readPath, "~/") {
		readPath = filepath.Join(home, readPath[2:])
	}
	body, err := os.ReadFile(readPath)
	if err != nil {
		t.Fatalf("stored record unreadable: %v", err)
	}
	if bytes.Contains(body, []byte(pat)) {
		t.Errorf("planted secret leaked into the stored record:\n%s", body)
	}
}

// TestHistoryShowSanitisesTranscriptBody proves `abcd history show` neutralises
// terminal-control sequences carried in a stored transcript. The body is
// untrusted input (it may have ingested hostile fetched pages or target-repo
// files), and capture redacts only secrets/home paths — nothing masks control
// bytes — so the human render must pass the body through termsafe before it
// reaches the terminal. Line structure must survive (the transcript is the
// artefact). The C1 fixture uses the two-byte-encoded U+009B, not a raw 0x9b
// byte (which would decode to U+FFFD and make the test vacuous).
func TestHistoryShowSanitisesTranscriptBody(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := t.TempDir()
	gitCmd(t, repo, "init")
	gitCmd(t, repo, "config", "user.email", "test@example.com")
	gitCmd(t, repo, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repo, "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, repo, "add", ".")
	gitCommit(t, repo, "commit", "-m", "init")
	t.Chdir(repo)

	rootSHA := gitCmd(t, repo, "rev-list", "--max-parents=0", "HEAD")
	tdir := filepath.Join(home, ".abcd", "history", rootSHA, "transcripts")
	if err := os.MkdirAll(tdir, 0o755); err != nil {
		t.Fatal(err)
	}

	// A benign three-line transcript laced with an ESC/CSI colour code, a
	// two-byte C1 CSI (U+009B), a bidi override (U+202E), and a bare CR.
	transcript := "user: line one [31mred[0m\n" +
		"assistant: 2Kcleared and ‮overridden\n" +
		"end\rof line\n"
	runCLIStdin(t, transcript, "history", "capture", "--session", "sess-esc", "--json")

	out := string(runCLI(t, "history", "show", "sess-esc"))

	for _, bad := range []string{"", "", "‮"} {
		if strings.Contains(out, bad) {
			t.Errorf("history show leaked control rune %q into terminal output:\n%q", bad, out)
		}
	}
	// A bare CR (not part of a CRLF) is an overprint vector and must be masked.
	if strings.Contains(out, "end\rof line") {
		t.Errorf("history show left a bare CR unmasked:\n%q", out)
	}
	// The body's three lines must survive sanitisation.
	body := out[strings.Index(out, "---\n")+len("---\n"):]
	if got := strings.Count(body, "\n"); got != 3 {
		t.Errorf("body line count = %d, want 3 (line structure not preserved):\n%q", got, body)
	}
}

// TestCaptureBlockedByWiredAndAnnotated proves the --blocked-by flag reaches
// capture.Capture from the CLI (writing the dependency edge), that an invalid
// token is rejected at the boundary, and that the derived-priority view renders
// unblocked-first with a [blocked-by …] annotation on the blocked row.
func TestCaptureBlockedByWiredAndAnnotated(t *testing.T) {
	_ = captureLedgerRepo(t)

	// iss-1: the blocker target (minor, unblocked).
	out := runCLI(t, "capture", "root cause", "--slug", "root", "--json")
	var r1 struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(out, &r1); err != nil {
		t.Fatalf("capture output not JSON: %v\n%s", err, out)
	}
	if r1.ID == "" {
		t.Fatalf("first capture minted no id:\n%s", out)
	}

	// The second issue: critical but blocked by the still-open first one.
	out2 := runCLI(t, "capture", "dependent thing", "--slug", "dep",
		"--severity", "critical", "--blocked-by", r1.ID, "--json")
	var r2 struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(out2, &r2); err != nil {
		t.Fatalf("blocked capture output not JSON: %v\n%s", err, out2)
	}
	if r2.ID == "" || r2.ID == r1.ID {
		t.Fatalf("second id = %q (first %q), want a distinct id", r2.ID, r1.ID)
	}
	// The edge reached disk.
	body, err := os.ReadFile(r2.Path)
	if err != nil {
		t.Fatalf("%s unreadable: %v", r2.ID, err)
	}
	if !strings.Contains(string(body), "blocked_by: ["+r1.ID+"]") {
		t.Fatalf("blocked_by not written to %s:\n%s", r2.ID, body)
	}

	// Derived view: the unblocked blocker ahead of the blocked, annotated row.
	list := string(runCLI(t, "capture", "list", "--open"))
	i1 := strings.Index(list, r1.ID)
	i2 := strings.Index(list, r2.ID)
	if i1 < 0 || i2 < 0 || i1 > i2 {
		t.Fatalf("expected %s before %s (unblocked-first):\n%s", r1.ID, r2.ID, list)
	}
	if !strings.Contains(list, "[blocked-by "+r1.ID+"]") {
		t.Fatalf("expected [blocked-by %s] annotation:\n%s", r1.ID, list)
	}

	// An invalid --blocked-by token is rejected at the boundary.
	if _, err := runCLIErr(t, "capture", "bad edge", "--blocked-by", "bogus"); err == nil {
		t.Fatalf("expected error for invalid --blocked-by token")
	}
}

// TestCaptureLinkWiredEndToEnd (iss-2609200951237670) proves `capture link`
// reaches capture.Link from the CLI: --blocked-by appends the edge after
// capture, --unblock removes it, the JSON carries id/path/blocked_by, the plain
// render is one line, and the refusal for an absent target names where the
// field is documented — on link and on the capture flag alike, whose help says
// so too.
func TestCaptureLinkWiredEndToEnd(t *testing.T) {
	_ = captureLedgerRepo(t)
	var r1, r2 struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(runCLI(t, "capture", "the blocker", "--slug", "blocker", "--json"), &r1); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(runCLI(t, "capture", "the dependent", "--slug", "dep", "--json"), &r2); err != nil {
		t.Fatal(err)
	}

	out := runCLI(t, "capture", "link", r2.ID, "--blocked-by", r1.ID, "--json")
	var res struct {
		ID        string   `json:"id"`
		Path      string   `json:"path"`
		BlockedBy []string `json:"blocked_by"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("link output not JSON: %v\n%s", err, out)
	}
	if res.ID != r2.ID || res.Path != r2.Path || strings.Join(res.BlockedBy, ",") != r1.ID {
		t.Fatalf("link result = %+v, want id %s path %s blocked_by [%s]", res, r2.ID, r2.Path, r1.ID)
	}
	body, err := os.ReadFile(r2.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "blocked_by: ["+r1.ID+"]") {
		t.Fatalf("edge not written to %s:\n%s", r2.ID, body)
	}
	if list := string(runCLI(t, "capture", "list", "--open")); !strings.Contains(list, "[blocked-by "+r1.ID+"]") {
		t.Fatalf("derived view did not pick the linked edge up:\n%s", list)
	}

	// Plain render: one line naming the id and the list after the write.
	// The harness merges stderr, where every capture verb names the ledger it
	// addressed (iss-2609202053570475); the render under test is stdout's.
	plain := withoutLedgerLine(string(runCLI(t, "capture", "link", r2.ID, "--unblock", r1.ID)))
	if n := strings.Count(strings.TrimRight(plain, "\n"), "\n"); n != 0 {
		t.Fatalf("plain render is not one line:\n%s", plain)
	}
	if !strings.Contains(plain, r2.ID) || !strings.Contains(plain, "blocked_by") {
		t.Fatalf("plain render names neither the id nor the field:\n%s", plain)
	}
	if body, _ = os.ReadFile(r2.Path); strings.Contains(string(body), "blocked_by") {
		t.Fatalf("unblock left the key behind:\n%s", body)
	}

	// The refusals, on both verbs, name where the field is documented.
	for _, args := range [][]string{
		{"capture", "link", r2.ID, "--blocked-by", "iss-999999"},
		{"capture", "another thing", "--blocked-by", "iss-999999"},
	} {
		out, err := runCLIErr(t, args...)
		if err == nil {
			t.Fatalf("%v: expected a refusal", args)
		}
		msg := err.Error() + string(out)
		for _, w := range []string{"iss-999999", ".abcd/work/issues/README.md", "commands/capture.md"} {
			if !strings.Contains(msg, w) {
				t.Errorf("%v: refusal does not carry %q: %s", args, w, msg)
			}
		}
	}
	if _, err := runCLIErr(t, "capture", "link", r2.ID); err == nil {
		t.Fatal("link with neither flag must be refused")
	}

	// The flag help on both verbs points at the same documentation.
	for _, args := range [][]string{{"capture", "--help"}, {"capture", "link", "--help"}} {
		help := string(runCLI(t, args...))
		if !strings.Contains(help, ".abcd/work/issues/README.md") || !strings.Contains(help, "commands/capture.md") {
			t.Errorf("%v: --blocked-by help does not name the docs:\n%s", args, help)
		}
	}
	if help := string(runCLI(t, "capture", "link", "--help")); !strings.Contains(help, "unblock") || !strings.Contains(strings.ToLower(help), "then") {
		t.Errorf("link help must say both flags are applied unblock-then-block:\n%s", help)
	}
}

// runCLIErr executes the command tree and returns its stdout/stderr plus the
// error, so a gate's non-zero exit can be asserted rather than fataled on.
func runCLIErr(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()
	return runCLIStdinErr(t, "", args...)
}

// runCLIStdinErr is the one harness the other three runners delegate to: stdin
// bound, output captured, error returned rather than fataled. A gate whose
// payload arrives on "-" needs both halves at once.
func runCLIStdinErr(t *testing.T, stdin string, args ...string) ([]byte, error) {
	t.Helper()
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.Bytes(), err
}

const docsLintConfig = `{
  "roots": ["docs"],
  "banned_tokens": [
    {"id":"present_tense/previously","pattern":"(?i)\\bpreviously\\b","severity":"blocker","message":"change-narration","successor":"present-tense phrasing","allow_context":["(?i)<!--\\s*docs-lint:\\s*allow\\b"]}
  ],
  "rules": {
    "links_resolve": {"enabled": true, "severity": "blocker"},
    "stray_root_docs": {"enabled": true, "severity": "blocker",
      "allowlist": ["README","AGENTS","CHANGELOG","CONTRIBUTING","SECURITY","LICENSE"]}
  }
}`

// TestDocsLintRootFlagFlagsDrift proves `docs lint --root/--config` runs layer 1
// over an arbitrary tree: a change-narration token, a broken cross-link, and a
// stray top-level markdown each surface as a blocker and drive a non-zero exit.
func TestDocsLintRootFlagFlagsDrift(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "docs-lint.json")
	if err := os.WriteFile(cfg, []byte(docsLintConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "bad.md"),
		[]byte("# Bad\n\nThis was previously X, now Y.\n\nSee [gone](./missing.md).\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "FOO.md"), []byte("# Stray\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCLIErr(t, "docs", "lint", "--json", "--config", cfg, "--root", root)
	if err == nil {
		t.Fatalf("expected non-zero exit on blockers, got nil\n%s", out)
	}
	var res struct {
		Findings []struct {
			RuleID   string `json:"RuleID"`
			Severity string `json:"Severity"`
		} `json:"findings"`
		Blockers int `json:"blockers"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, out)
	}
	if res.Blockers < 3 {
		t.Fatalf("blockers = %d, want >= 3\n%s", res.Blockers, out)
	}
	rules := map[string]bool{}
	for _, f := range res.Findings {
		rules[f.RuleID] = true
	}
	for _, want := range []string{"present_tense/previously", "links_resolve", "stray_root_docs"} {
		if !rules[want] {
			t.Fatalf("expected a %s blocker, got findings %v", want, rules)
		}
	}
}

// TestDocsLintCleanTreePasses proves a present-tense, well-linked tree with no
// stray root docs exits zero under the same layer-1 config.
func TestDocsLintCleanTreePasses(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "docs-lint.json")
	if err := os.WriteFile(cfg, []byte(docsLintConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "peer.md"), []byte("# Peer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "good.md"),
		[]byte("# Good\n\nThe pass grades docs against reality.\n\nSee [peer](./peer.md).\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runCLIErr(t, "docs", "lint", "--json", "--config", cfg, "--root", root)
	if err != nil {
		t.Fatalf("clean tree should pass, got error: %v\n%s", err, out)
	}
	var res struct {
		Blockers int `json:"blockers"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, out)
	}
	if res.Blockers != 0 {
		t.Fatalf("clean tree blockers = %d, want 0\n%s", res.Blockers, out)
	}
}

// hermeticEnv redirects HOME, the plugin root and the PATH symlink target to
// temp locations without chdir'ing anywhere, so a caller can classify an
// arbitrary folder shape. It never touches the real machine.
func hermeticEnv(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	pluginRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(pluginRoot, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "hooks", "hooks.json"), []byte(validHooksJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "abcd"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", pluginRoot)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("ABCD_BIN_TARGET", filepath.Join(t.TempDir(), "bin", "abcd"))
	// Scrub any real abcd off PATH (iss-249, and the same guard setupHermetic in the
	// ahoy package applies). Without it, effectiveBinTarget finds the developer's own
	// installed abcd — exactly the machines that dogfood the installer — and `ahoy
	// install` in a test ADOPTS it, rewriting a real PATH entry to point into the
	// test's temp dir: a destructive escape from the hermetic sandbox. Lstat, not
	// Stat, so a DANGLING abcd symlink (a link whose target a prior killed test
	// removed) is dropped too rather than followed to an error and kept.
	var kept []string
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if fi, err := os.Lstat(filepath.Join(dir, "abcd")); err == nil && !fi.IsDir() {
			continue
		}
		kept = append(kept, dir)
	}
	t.Setenv("PATH", strings.Join(kept, string(os.PathListSeparator)))
}

// TestAhoyBareUnmanagedRepoNamesAdoptPath proves itd-40 AC2: bare `abcd ahoy`
// in a git repo with no abcd markers reports unmanaged-repo AND names
// `/abcd:ahoy install` as the way to adopt it — without adopting it.
func TestAhoyBareUnmanagedRepoNamesAdoptPath(t *testing.T) {
	hermeticEnv(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)

	out := string(runCLI(t, "ahoy"))
	if !strings.Contains(out, "unmanaged-repo") {
		t.Fatalf("bare ahoy did not report unmanaged-repo:\n%s", out)
	}
	if !strings.Contains(out, "/abcd:ahoy install") {
		t.Fatalf("bare ahoy on an unmanaged repo did not name the adopt path `/abcd:ahoy install`:\n%s", out)
	}
	// Read-only: classification must not adopt (no .abcd/, no marker written).
	if _, err := os.Stat(filepath.Join(repo, ".abcd")); !os.IsNotExist(err) {
		t.Fatalf("bare ahoy mutated the repo (.abcd/ appeared): %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Fatalf("bare ahoy mutated the repo (CLAUDE.md appeared): %v", err)
	}
}

// TestAhoyBareUnmanagedFolderReportsNothingToActOn proves itd-40 AC3: bare
// `abcd ahoy` in a non-git folder reports unmanaged-folder and that there is
// nothing to act on, mutating nothing.
func TestAhoyBareUnmanagedFolderReportsNothingToActOn(t *testing.T) {
	hermeticEnv(t)
	folder := t.TempDir()
	t.Chdir(folder)

	out := string(runCLI(t, "ahoy"))
	if !strings.Contains(out, "unmanaged-folder") {
		t.Fatalf("bare ahoy did not report unmanaged-folder:\n%s", out)
	}
	if !strings.Contains(out, "nothing to act on") {
		t.Fatalf("bare ahoy on a plain folder did not report there is nothing to act on:\n%s", out)
	}
}

// hermeticGitRepo sets the hermetic env (as hermeticEnv) and chdirs into a
// fresh real git repo carrying one root commit, returning the repo path and its
// root-commit SHA. Unlike hermeticRepo (a bare .git mkdir with no commits) this
// yields a non-empty root SHA, so the history registry keys on a real identity.
func hermeticGitRepo(t *testing.T) (repo, rootSHA string) {
	t.Helper()
	hermeticEnv(t)
	repo = t.TempDir()
	gitCmd(t, repo, "init", "-q")
	gitCmd(t, repo, "config", "user.email", "dev@example.com")
	gitCmd(t, repo, "config", "user.name", "Dev")
	gitCommit(t, repo, "commit", "-q", "--allow-empty", "-m", "root")
	rootSHA = gitCmd(t, repo, "rev-list", "--max-parents=0", "HEAD")
	t.Chdir(repo)
	return repo, rootSHA
}

// TestAhoyInstallBootstrapsAndRegistersByRootSHA proves itd-40 AC4: with no
// ~/.abcd/history/ store present, the first `ahoy install` bootstraps the store
// (dir + index.json) and registers the repo in it keyed on the root-commit SHA.
func TestAhoyInstallBootstrapsAndRegistersByRootSHA(t *testing.T) {
	repo, rootSHA := hermeticGitRepo(t)
	_ = repo
	home := os.Getenv("HOME")
	indexPath := filepath.Join(home, ".abcd", "history", "index.json")
	if _, err := os.Stat(indexPath); !os.IsNotExist(err) {
		t.Fatalf("history store existed before install: %v", err)
	}

	runCLI(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("history store not bootstrapped: %v", err)
	}
	var idx struct {
		Repos []struct {
			RootCommit string `json:"root_commit"`
			Path       string `json:"path"`
		} `json:"repos"`
	}
	if err := json.Unmarshal(data, &idx); err != nil {
		t.Fatalf("index.json not JSON: %v\n%s", err, data)
	}
	found := false
	for _, r := range idx.Repos {
		if r.RootCommit == rootSHA {
			found = true
		}
	}
	if !found {
		t.Fatalf("repo not registered by root-commit SHA %q in index.json:\n%s", rootSHA, data)
	}
}

// TestAhoyDoctorResolvesCentralLocationFromIndex proves itd-40 AC5: the
// central/host location is resolved by reading index.json — not a hardcoded path
// or a directory walk. Rewriting the registered path makes doctor report a stale
// path whose detail quotes the value it read from index.json.
func TestAhoyDoctorResolvesCentralLocationFromIndex(t *testing.T) {
	_, _ = hermeticGitRepo(t)
	runCLI(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")

	indexPath := filepath.Join(os.Getenv("HOME"), ".abcd", "history", "index.json")
	// A freshly-registered repo reconciles cleanly: zero audit gaps.
	out := runCLI(t, "ahoy", "doctor", "--json")
	var clean struct {
		AuditGaps []struct {
			ID string `json:"id"`
		} `json:"audit_gaps"`
	}
	if err := json.Unmarshal(out, &clean); err != nil {
		t.Fatalf("doctor output not JSON: %v\n%s", err, out)
	}
	if len(clean.AuditGaps) != 0 {
		t.Fatalf("clean repo produced audit gaps: %+v", clean.AuditGaps)
	}

	// Corrupt only the registered path in index.json.
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	repos := raw["repos"].([]any)
	repos[0].(map[string]any)["path"] = "/somewhere/relocated"
	patched, _ := json.MarshalIndent(raw, "", "  ")
	if err := os.WriteFile(indexPath, patched, 0o644); err != nil {
		t.Fatal(err)
	}

	out2 := runCLI(t, "ahoy", "doctor", "--json")
	var stale struct {
		AuditGaps []struct {
			ID     string `json:"id"`
			Detail string `json:"detail"`
		} `json:"audit_gaps"`
	}
	if err := json.Unmarshal(out2, &stale); err != nil {
		t.Fatalf("doctor output not JSON: %v\n%s", err, out2)
	}
	staleFound := false
	for _, g := range stale.AuditGaps {
		if g.ID == "history.path_stale" {
			staleFound = true
			if !strings.Contains(g.Detail, "/somewhere/relocated") {
				t.Fatalf("path_stale detail did not quote the location read from index.json: %q", g.Detail)
			}
		}
	}
	if !staleFound {
		t.Fatalf("doctor did not resolve the central location from index.json (no history.path_stale): %+v", stale.AuditGaps)
	}
}

// TestAhoyDoctorJSONCarriesNoHomePrefix is the surface pin for
// GHSA-m8pg-chhv-hxvq: a stale registered path under the home directory must
// reach doctor's JSON in tilde form, never as the raw absolute path.
func TestAhoyDoctorJSONCarriesNoHomePrefix(t *testing.T) {
	_, _ = hermeticGitRepo(t)
	runCLI(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")

	home := os.Getenv("HOME")
	indexPath := filepath.Join(home, ".abcd", "history", "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	repos := raw["repos"].([]any)
	repos[0].(map[string]any)["path"] = filepath.Join(home, "elsewhere", "repo")
	patched, _ := json.MarshalIndent(raw, "", "  ")
	if err := os.WriteFile(indexPath, patched, 0o644); err != nil {
		t.Fatal(err)
	}

	out := runCLI(t, "ahoy", "doctor", "--json")
	var report struct {
		AuditGaps []struct {
			ID     string `json:"id"`
			Detail string `json:"detail"`
		} `json:"audit_gaps"`
	}
	if err := json.Unmarshal(out, &report); err != nil {
		t.Fatalf("doctor output not JSON: %v\n%s", err, out)
	}
	found := false
	for _, g := range report.AuditGaps {
		if g.ID != "history.path_stale" {
			continue
		}
		found = true
		if strings.Contains(g.Detail, home) {
			t.Fatalf("path_stale detail leaks the home directory: %q", g.Detail)
		}
		if !strings.Contains(g.Detail, "~/elsewhere/repo") {
			t.Fatalf("path_stale detail = %q, want the registered path in tilde form", g.Detail)
		}
	}
	if !found {
		t.Fatalf("doctor reported no history.path_stale: %+v", report.AuditGaps)
	}
	if strings.Contains(string(out), home) {
		t.Fatalf("doctor --json carries the home directory somewhere in its output:\n%s", out)
	}
}

// TestAhoyDoctorNamesTheDiagnosticsNoInstallCanFix: doctor's text render is the
// read-only surface an operator reaches for when something is wrong, and it
// printed two integers. A required gap that is deliberately NOT resolvable —
// config.malformed is the case that matters, since abcd will never touch the
// file again until a human repairs it — is exactly the one no later `install`
// will clear, so a count that cannot be acted on is the wrong report. Name each
// such diagnostic and its detail.
func TestAhoyDoctorNamesTheDiagnosticsNoInstallCanFix(t *testing.T) {
	repo, _ := hermeticGitRepo(t)
	runCLI(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")

	cfg := filepath.Join(repo, ".abcd", "config.json")
	if err := os.WriteFile(cfg, []byte("{\"repo\":{\"visibility\":\"private\"}\n<<<<<<<\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := string(runCLI(t, "ahoy", "doctor"))
	if !strings.Contains(out, "config.malformed") {
		t.Fatalf("doctor did not name the diagnostic no install can fix:\n%s", out)
	}
	if !strings.Contains(out, "could not be parsed") {
		t.Fatalf("doctor named the gap but not what is wrong with it:\n%s", out)
	}
}
