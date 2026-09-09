package cli

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The three tests below are iss-29's acceptance corpus for the
// "unrecognized-input-never-writes" detector: a mistyped mutating subcommand
// must error without writing, --json errors must be JSON-shaped, and a missing
// config must surface a clean, path-safe error rather than raw Go text.

var reIssueFile = regexp.MustCompile(`^iss-\d+-.*\.md$`)

// ledgerIssueCount walks a repo tree and counts written ledger issue files.
func ledgerIssueCount(t *testing.T, root string) int {
	t.Helper()
	n := 0
	err := filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && reIssueFile.MatchString(d.Name()) {
			n++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return n
}

// TestCaptureTypoSubcommandNeverWrites is the headline: `capture resovle iss-1
// note` (a typo for `resolve`) must be refused with a did-you-mean, and must
// not file a new issue. Before the fix it was swallowed as free text and wrote.
func TestCaptureTypoSubcommandNeverWrites(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out, err := runCLIErr(t, "capture", "resovle", "iss-1", "clear the flake")
	if err == nil {
		t.Fatalf("expected an error for the mistyped subcommand, got success:\n%s", out)
	}
	if !strings.Contains(err.Error(), "resolve") {
		t.Fatalf("expected a did-you-mean pointing at %q, got: %v", "resolve", err)
	}
	if n := ledgerIssueCount(t, repo); n != 0 {
		t.Fatalf("a mistyped subcommand filed %d issue(s); it must write nothing", n)
	}
}

// TestCaptureFreeTextStillWrites guards the contract: a genuine free-text
// capture whose first word merely resembles a subcommand (but is followed by
// prose, not an iss-id) still files. The typo guard must be high-precision.
func TestCaptureFreeTextStillWrites(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out := runCLI(t, "capture", "resolved a flaky parser test by widening the timeout", "--json")
	var r struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("capture output not JSON: %v\n%s", err, out)
	}
	if !regexp.MustCompile(`^iss-[0-9]{16}$`).MatchString(r.ID) {
		t.Fatalf("free-text capture id = %q, want a native timestamp-numeric id", r.ID)
	}
	if n := ledgerIssueCount(t, repo); n != 1 {
		t.Fatalf("free-text capture wrote %d issue(s), want 1", n)
	}
}

// TestJSONErrorShapeIsJSON is the --json error-shape contract: when the caller
// asked for --json, a command error is emitted as a JSON envelope, not raw Go
// text. `capture list --json` (no state flag) is a stable erroring case.
func TestJSONErrorShapeIsJSON(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"capture", "list", "--json"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected a non-zero exit for `capture list` with no state flag")
	}
	var env struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &env); err != nil {
		t.Fatalf("--json error not JSON-shaped: %v\nstderr: %q", err, stderr.String())
	}
	if env.Error == "" {
		t.Fatalf("--json error envelope has an empty message:\n%s", stderr.String())
	}
}

// TestDocsLintMissingConfigCleanError proves the third instance: a missing
// docs-lint config yields a clean, repo-relative diagnostic — never a raw
// os.Open error leaking the absolute config path.
func TestDocsLintMissingConfigCleanError(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"docs", "lint"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected a non-zero exit for `docs lint` with no config")
	}
	msg := stderr.String()
	if strings.Contains(msg, repo) {
		t.Fatalf("docs lint error leaked the absolute repo path %q:\n%s", repo, msg)
	}
	if !strings.Contains(msg, filepath.Join(".abcd", "docs-lint.json")) {
		t.Fatalf("docs lint error should name the repo-relative config path:\n%s", msg)
	}

	// And under --json it is JSON-shaped, not raw text.
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"docs", "lint", "--json"}, &stdout, &stderr); code == 0 {
		t.Fatalf("expected a non-zero exit for `docs lint --json` with no config")
	}
	var env struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &env); err != nil {
		t.Fatalf("--json docs lint error not JSON-shaped: %v\nstderr: %q", err, stderr.String())
	}
}

// TestDocsLintEngineFaultExitsTwo pins that `docs lint` returns exit 2 — "could
// not be evaluated" — when the engine cannot run (here, a config whose root
// escapes the repository), the same tri-state code `abcd lint` and record-lint
// return. Exit 1 is reserved for a blocker finding, so a CI gate keying on >=2
// must not read a lint that never ran as an ordinary findings-pass.
func TestDocsLintEngineFaultExitsTwo(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	cfgDir := filepath.Join(repo, ".abcd")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A root that escapes the repository is a containment refusal the engine
	// raises before it can scan anything — an engine fault, not a finding.
	if err := os.WriteFile(filepath.Join(cfgDir, "docs-lint.json"),
		[]byte(`{"roots":["../outside"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"docs", "lint"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("docs lint engine fault exit = %d, want 2\nstderr: %s", code, stderr.String())
	}
	if strings.Contains(stderr.String(), repo) {
		t.Errorf("engine fault message leaked the absolute repo path:\n%s", stderr.String())
	}
}

// TestCaptureListOpenRendersIssueFields is itd-4 AC5's characterization gate:
// given five open captures, `capture list --open --json` must return all five,
// each carrying the id, slug, severity, and a one-line summary (the captured
// body). It exercises the exact binary path the acceptance criterion names, so
// a regression that dropped any of those fields from the list surface — or that
// failed to enumerate every open issue — would turn this red.
func TestCaptureListOpenRendersIssueFields(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	captures := []struct {
		text string
		sev  string
	}{
		{"issue one about the parser flake", "major"},
		{"issue two about a cache miss", "minor"},
		{"issue three nitpick on spacing", "nitpick"},
		{"issue four critical crash on boot", "critical"},
		{"issue five stray observation", "minor"},
	}
	for _, c := range captures {
		runCLI(t, "capture", c.text, "--severity", c.sev, "--json")
	}

	out := runCLI(t, "capture", "list", "--open", "--json")
	var res struct {
		Issues []struct {
			ID       string `json:"id"`
			Slug     string `json:"slug"`
			Severity string `json:"severity"`
			Body     string `json:"body"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("list --open --json not JSON: %v\n%s", err, out)
	}
	if len(res.Issues) != len(captures) {
		t.Fatalf("list --open returned %d issues, want %d", len(res.Issues), len(captures))
	}
	// Map by the one-line summary (body) so the assertion is order-independent
	// (list emits derived-priority order, not capture order). The stored body now
	// carries exactly one trailing newline (iss-175: the writer terminates every
	// record), which the raw body field reflects — so key on the trimmed summary.
	bySummary := make(map[string]struct{ id, slug, sev string })
	for _, iss := range res.Issues {
		if iss.ID == "" || iss.Slug == "" || iss.Severity == "" || iss.Body == "" {
			t.Fatalf("issue missing an AC5 field: id=%q slug=%q severity=%q body=%q",
				iss.ID, iss.Slug, iss.Severity, iss.Body)
		}
		bySummary[strings.TrimRight(iss.Body, "\n")] = struct{ id, slug, sev string }{iss.ID, iss.Slug, iss.Severity}
	}
	for _, c := range captures {
		got, ok := bySummary[c.text]
		if !ok {
			t.Fatalf("no listed issue carried the one-line summary %q", c.text)
		}
		if got.sev != c.sev {
			t.Errorf("summary %q: severity = %q, want %q", c.text, got.sev, c.sev)
		}
	}
}

// TestDocsLintUnreadableConfigNoPathLeak covers a non-not-exist load failure
// (the config path is a directory → EISDIR): a *PathError's Error() embeds the
// absolute path, so the branch must strip it. Guards the security-review BLOCK.
func TestDocsLintUnreadableConfigNoPathLeak(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	// Make .abcd/docs-lint.json a directory so os.ReadFile fails with EISDIR.
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", "docs-lint.json"), 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"docs", "lint"}, &stdout, &stderr); code == 0 {
		t.Fatalf("expected a non-zero exit for an unreadable docs-lint config")
	}
	if msg := stderr.String(); strings.Contains(msg, repo) {
		t.Fatalf("docs lint error leaked the absolute repo path %q:\n%s", repo, msg)
	}

	// Same guarantee under --json.
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"docs", "lint", "--json"}, &stdout, &stderr); code == 0 {
		t.Fatalf("expected a non-zero exit under --json for an unreadable config")
	}
	if msg := stderr.String(); strings.Contains(msg, repo) {
		t.Fatalf("--json docs lint error leaked the absolute repo path %q:\n%s", repo, msg)
	}
}

// TestCapturePromoteJSONContract is the spc-24 surface AC: `capture promote
// <iss-N> --json` reports the issue id, the minted intent id, and both
// repo-relative paths; the minted draft and the stamped issue exist on disk.
func TestCapturePromoteJSONContract(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	capOut := runCLI(t, "capture", "the parser eats trailing newlines", "--json")
	var minted struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(capOut, &minted); err != nil || minted.ID == "" {
		t.Fatalf("capture envelope unreadable: %v\n%s", err, capOut)
	}

	out := runCLI(t, "capture", "promote", minted.ID, "--grounds", cliGrounds, "--json")
	var r struct {
		IssueID    string `json:"issue_id"`
		IssuePath  string `json:"issue_path"`
		IntentID   string `json:"intent_id"`
		IntentPath string `json:"intent_path"`
		Linked     bool   `json:"linked"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("promote output not JSON: %v\n%s", err, out)
	}
	if r.IssueID != minted.ID || !cliNativeIntentIDRe.MatchString(r.IntentID) || r.Linked {
		t.Fatalf("unexpected promote result: %+v", r)
	}
	for _, p := range []string{r.IssuePath, r.IntentPath} {
		if p == "" || filepath.IsAbs(p) {
			t.Fatalf("promote paths must be repo-relative and non-empty: %+v", r)
		}
		if _, err := os.Stat(filepath.Join(repo, p)); err != nil {
			t.Fatalf("promote-reported path %s missing: %v", p, err)
		}
	}

	// Second promote refuses (exit non-zero) and names the existing intent.
	if _, err := runCLIErr(t, "capture", "promote", minted.ID, "--grounds", cliGrounds); err == nil || !strings.Contains(err.Error(), r.IntentID) {
		t.Fatalf("second promote must refuse naming %s, got: %v", r.IntentID, err)
	}
}

// TestCaptureResolveProvenanceJSON is the spc-25 surface AC: resolve with
// provenance flags reports the written resolved_by members in --json, and an
// unknown id refuses without writing.
func TestCaptureResolveProvenanceJSON(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	capOut := runCLI(t, "capture", "a provenance-carrying issue", "--json")
	var minted struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(capOut, &minted); err != nil || minted.ID == "" {
		t.Fatalf("capture envelope unreadable: %v\n%s", err, capOut)
	}
	intentRel := ".abcd/development/intents/shipped/itd-4-the-fixer.md"
	if err := os.MkdirAll(filepath.Join(repo, filepath.Dir(intentRel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, intentRel), []byte("---\nid: itd-4\n---\n\nx\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Unknown intent refuses; the issue stays open.
	if _, err := runCLIErr(t, "capture", "resolve", minted.ID, "nope", "--impact", "fix", "--grounds", cliGrounds, "--intent", "itd-99"); err == nil {
		t.Fatalf("resolve with an unknown --intent must refuse")
	}

	out := runCLI(t, "capture", "resolve", minted.ID, "fixed by the fixer", "--impact", "fix",
		"--grounds", cliGrounds, "--intent", "itd-4", "--commit", "abcdef0123", "--json")
	var r struct {
		ID         string `json:"id"`
		ToStatus   string `json:"to_status"`
		ResolvedBy *struct {
			Intent string `json:"intent"`
			Spec   string `json:"spec"`
			Commit string `json:"commit"`
		} `json:"resolved_by"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("resolve output not JSON: %v\n%s", err, out)
	}
	if r.ID != minted.ID || r.ToStatus != "resolved" {
		t.Fatalf("unexpected resolve result: %+v", r)
	}
	if r.ResolvedBy == nil || r.ResolvedBy.Intent != "itd-4" || r.ResolvedBy.Commit != "abcdef0123" || r.ResolvedBy.Spec != "" {
		t.Fatalf("resolved_by members wrong: %+v", r.ResolvedBy)
	}
}

// iss-2608221328552172: the same lone-token hole on the capture create path.
// `abcd capture nosuchthing` is not near any sub-verb, so the did-you-mean
// guard never fired and the token was filed as an issue at exit 0.

// TestCaptureLoneWordNeverWrites is the headline: a single bare word must be
// refused, and must file no issue.
func TestCaptureLoneWordNeverWrites(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out, err := runCLIErr(t, "capture", "nosuchthing")
	if err == nil {
		t.Fatalf("expected an error for the lone unknown token, got success:\n%s", out)
	}
	if n := ledgerIssueCount(t, repo); n != 0 {
		t.Fatalf("a lone unknown token filed %d issue(s); it must write nothing", n)
	}
}

// TestCaptureLoneRecordIDNeverWrites: `abcd capture iss-1` (a forgotten
// sub-verb) must not become an issue titled with a record id.
func TestCaptureLoneRecordIDNeverWrites(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out, err := runCLIErr(t, "capture", "iss-1")
	if err == nil {
		t.Fatalf("expected an error for the lone record id, got success:\n%s", out)
	}
	if n := ledgerIssueCount(t, repo); n != 0 {
		t.Fatalf("a lone record id filed %d issue(s); it must write nothing", n)
	}
}

// TestCaptureEmptyTextNeverWrites: `abcd capture ""` — a script whose variable
// expanded to nothing — is the degenerate lone token. Core already refused it
// ("slug normalises to empty", exit 1); the guard now names it for what it is,
// at the usage code. Either way nothing is written, which is the part that binds.
func TestCaptureEmptyTextNeverWrites(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out, err := runCLIErr(t, "capture", "")
	if err == nil {
		t.Fatalf("expected an error for empty capture text, got success:\n%s", out)
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected the error to name the text as empty, got: %v", err)
	}
	if n := ledgerIssueCount(t, repo); n != 0 {
		t.Fatalf("empty capture text filed %d issue(s); it must write nothing", n)
	}
}

// TestCaptureDerivedSlugNeverCarriesHomePathUsername is the regression for
// GH #485 (reported by @jogrun): `abcd capture <text>` derived the issue slug
// from the RAW text before redaction ran, so a /Users/<name>/… home path in the abcd-audit:allow
// capture text put the caller's username into the committed issue filename even
// though the body was redacted. deriveSlug kebab-cased the path first, so nothing
// left looked like a path and the ledger redactor never saw it — the body was
// clean but open/iss-N-users-alice-….md still carried the name. Redaction must
// run BEFORE the slug is derived, so a finding can never reach the filename.
func TestCaptureDerivedSlugNeverCarriesHomePathUsername(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	// A fake home path — deliberately NOT the caller's own HOME — so the
	// assertion turns on the ordering of redaction vs slug derivation, not on the
	// test host's identity. The generic /Users|/home matcher (home_path_other)
	// redacts it either way.
	const username = "alice"
	body := "the key at /Users/" + username + "/.ssh/id_rsa leaked into the log" // abcd-audit:allow

	out := runCLI(t, "capture", body, "--json")
	var r struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("capture output not JSON: %v\n%s", err, out)
	}
	if strings.Contains(r.Slug, username) {
		t.Fatalf("derived slug %q carries the home-path username %q", r.Slug, username)
	}
	if strings.Contains(r.Path, username) {
		t.Fatalf("reported path %q carries the home-path username %q", r.Path, username)
	}
	// And no committed issue filename on disk carries it either.
	if err := filepath.WalkDir(repo, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && reIssueFile.MatchString(d.Name()) && strings.Contains(d.Name(), username) {
			t.Fatalf("committed issue filename %q carries the home-path username %q", d.Name(), username)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", repo, err)
	}
}

// TestCaptureStatusBoardRendersSkipped (iss-2608261437041050): a ledger record
// the reader refuses is counted by none of the board's three totals, so a board
// that printed the totals alone would report a ledger smaller than the one on
// disk and say nothing about the record it dropped. `capture list` already
// renders the skipped roster; the bare status board must not undercount in
// silence.
func TestCaptureStatusBoardRendersSkipped(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	// One well-formed capture, so the ledger dirs exist and the board has a total.
	runCLI(t, "capture", "a well formed observation", "--slug", "fine", "--json")
	// A committed record missing a required property: the reader validates before
	// it reads, so this one is skipped rather than counted.
	stripped := filepath.Join(repo, ".abcd", "work", "issues", "open", "iss-900-stripped.md")
	if err := os.WriteFile(stripped, []byte(
		"---\nid: iss-900\nslug: stripped\nseverity: minor\ncategory: bug\nsource: user-observation\nfound_during: t\n---\n\nan issue\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	board := string(runCLI(t, "capture"))
	if !strings.Contains(board, "skipped") || !strings.Contains(board, "iss-900-stripped.md") {
		t.Fatalf("the status board must render the skipped roster:\n%s", board)
	}
}

// TestCaptureLapsedAtWritesTheGivenInstant pins the flag half of spc-60: the
// instant handed to --lapsed-at is the instant committed to the record. The
// record id is minted from the wall clock, so a surface that dropped, rounded or
// re-derived the value would leave a lapse entry stamped with its own write-up
// time and nothing to say so.
func TestCaptureLapsedAtWritesTheGivenInstant(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	const lapsedAt = "2026-08-28T09:15:00Z"
	out := runCLI(t, "capture", "the discipline gave way here",
		"--category", "lapse", "--lapsed-at", lapsedAt, "--json")
	var r struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("capture output not JSON: %v\n%s", err, out)
	}
	body, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(r.Path)))
	if err != nil {
		t.Fatalf("read the captured record: %v", err)
	}
	if want := "lapsed_at: \"" + lapsedAt + "\""; !strings.Contains(string(body), want) {
		t.Fatalf("the captured record does not carry %s:\n%s", want, body)
	}
}

// TestCaptureLapsedAtHasNoDefault is the flag's whole point, made checkable: a
// lapse capture with --lapsed-at omitted records NO instant. Every other
// provenance flag on capture falls back to a default; the only fallback
// available here is the wall clock at write-up, which is the one value itd-182's
// criterion rules out. The refusal that stood here is parked
// (iss-2609091009111294): the record is written, and it carries no lapsed_at.
func TestCaptureLapsedAtHasNoDefault(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	out := runCLI(t, "capture", "the discipline gave way here", "--category", "lapse", "--json")
	var minted struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(out, &minted); err != nil || minted.Path == "" {
		t.Fatalf("capture envelope unreadable: %v\n%s", err, out)
	}
	if n := ledgerIssueCount(t, repo); n != 1 {
		t.Fatalf("the lapse capture wrote %d record(s); want exactly 1", n)
	}
	body, err := os.ReadFile(filepath.Join(repo, minted.Path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "lapsed_at:") {
		t.Fatalf("a lapse capture with no --lapsed-at invented an instant:\n%s", body)
	}
}

// writeReadingFixture lays down one committed reading record by hand, so the
// disposition verb has an item to answer. Reading records are written by the
// ingest path, which is the cold-reading output contract's front door, not this
// surface's — a fixture is the honest stand-in here.
func writeReadingFixture(t *testing.T, repo, run, item string) {
	t.Helper()
	dir := filepath.Join(repo, ".abcd", "work", "issues", "readings", run)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	record := "---\n" +
		"schema_version: 1\n" +
		"id: \"" + item + "\"\n" +
		"run: \"" + run + "\"\n" +
		"manifest: \"sha256:beef\"\n" +
		"position: \"detection\"\n" +
		"regime: \"registrative\"\n" +
		"pattern: \"a stated constraint\"\n" +
		"tension: \"the two sides disagree\"\n" +
		"constraint_in_play: \"the stated invariant\"\n" +
		"why_a_tension: \"one of them must give\"\n" +
		"---\n\n"
	if err := os.WriteFile(filepath.Join(dir, item+".md"), []byte(record), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The disposition verb is the front door of every refusal spc-58 specifies, so
// it has to BE a front door: reachable from the CLI, refusing what the core
// refuses, and writing nothing when it refuses.
func TestCaptureDispositionRefusesEmptyGroundsAndWritesNothing(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeReadingFixture(t, repo, "rdg-2608300000000001", "rdi-2608300000000002")

	out, err := runCLIErr(t, "capture", "disposition", "rdi-2608300000000002", "--state", "accepted")
	if err == nil {
		t.Fatalf("a disposition with no grounds must be refused, got success:\n%s", out)
	}
	if !strings.Contains(err.Error(), "disposition_grounds") {
		t.Fatalf("the refusal must name the rule it enforces; got %v", err)
	}
	dispositions := filepath.Join(repo, ".abcd", "work", "issues", "dispositions")
	if entries, derr := os.ReadDir(filepath.Join(dispositions, "rdi-2608300000000002")); derr == nil && len(entries) > 0 {
		t.Fatalf("a refused disposition wrote %d file(s); it must write nothing", len(entries))
	}
}

// The happy path: one answer, one record, under a directory keyed by the item.
func TestCaptureDispositionWritesTheKeyedRecord(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeReadingFixture(t, repo, "rdg-2608300000000001", "rdi-2608300000000002")

	out := runCLI(t, "capture", "disposition", "rdi-2608300000000002",
		"--state", "accepted", "--grounds", "the tension is real and worth acting on", "--json")
	var r struct {
		ID       string `json:"id"`
		Item     string `json:"item"`
		State    string `json:"state"`
		Position string `json:"position"`
		Path     string `json:"path"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("disposition output not JSON: %v\n%s", err, out)
	}
	if !regexp.MustCompile(`^dsp-[0-9]{16}$`).MatchString(r.ID) {
		t.Fatalf("disposition id = %q, want a native timestamp-numeric dsp id", r.ID)
	}
	if r.Position != "detection" {
		t.Fatalf("position = %q, want the position read off the keyed reading record", r.Position)
	}
	want := filepath.ToSlash(filepath.Join(".abcd/work/issues/dispositions", r.Item, r.ID+".md"))
	if filepath.ToSlash(r.Path) != want {
		t.Fatalf("path = %q, want %q", r.Path, want)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(r.Path))); err != nil {
		t.Fatalf("the disposition record must exist on disk: %v", err)
	}
}

// The outstanding-readings roster rides the bare status board. This is the
// wiring assertion: the rule and the board call one function, so an item nobody
// has answered is visible where a person actually looks, not only in a lint run
// they have to remember to make.
func TestCaptureBoardCarriesTheOutstandingRoster(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	writeReadingFixture(t, repo, "rdg-2608300000000001", "rdi-2608300000000002")

	out := runCLI(t, "capture", "--json")
	var r struct {
		Outstanding struct {
			Undispositioned []struct {
				Item string `json:"item"`
			} `json:"undispositioned"`
			OpenHolds []struct {
				Item          string `json:"item"`
				ExitCondition string `json:"exit_condition"`
			} `json:"open_holds"`
		} `json:"reading_outstanding"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("capture board not JSON: %v\n%s", err, out)
	}
	if len(r.Outstanding.Undispositioned) != 1 || r.Outstanding.Undispositioned[0].Item != "rdi-2608300000000002" {
		t.Fatalf("the board must carry the undispositioned item; got %+v\n%s", r.Outstanding, out)
	}

	// A held item renders on the board WITH its exit condition, which is the only
	// thing that distinguishes a hold from a parking space.
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", "work", "issues", "dispositions", "rdi-2608300000000002"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(repo, ".abcd", "work", "issues", "dispositions", "rdi-2608300000000002", "dsp-2608300000000003.md"),
		[]byte("---\nschema_version: 1\nid: \"dsp-2608300000000003\"\nitem: \"rdi-2608300000000002\"\n"+
			"state: \"held\"\nexit_condition: \"the closing run returns it again\"\n---\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	text := string(runCLI(t, "capture"))
	if !strings.Contains(text, "exits when: the closing run returns it again") {
		t.Fatalf("the board must render an open hold with its exit condition:\n%s", text)
	}
}

// A widening proposal's answers are an admission or a decline, so the board
// gives it its own line: the disposition-only line would name the wrong remedy,
// and the grounds an admission was made on are the whole point of the record.
// This is the wiring assertion for spc-67's leg — the rule and the board call one
// function, and the person who looks at the board is the one who has to act.
func TestCaptureBoardNamesAnUnadmittedProposal(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	dir := filepath.Join(repo, ".abcd", "work", "issues", "readings", run)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, item+".md"), []byte("---\n"+
		"schema_version: 1\nid: \""+item+"\"\nrun: \""+run+"\"\nmanifest: \"sha256:beef\"\n"+
		"position: \"widening\"\nregime: \"generative\"\npattern: \"a stated constraint\"\n"+
		"configuration: \"a third arrangement the frame does not hold\"\n"+
		"what_admits_it: \"the constraint the record already states\"\n---\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Accepted, which at the widening position IS admission — and no admission
	// record, so the grounds it was admitted on were never written.
	dispDir := filepath.Join(repo, ".abcd", "work", "issues", "dispositions", item)
	if err := os.MkdirAll(dispDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dispDir, "dsp-2608300000000003.md"), []byte("---\n"+
		"schema_version: 1\nid: \"dsp-2608300000000003\"\nitem: \""+item+"\"\n"+
		"state: \"accepted\"\ndisposition_grounds: \"weighed and answered\"\n---\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runCLI(t, "capture", "--json")
	var r struct {
		Outstanding struct {
			Unadmitted []struct {
				Item string `json:"item"`
			} `json:"unadmitted"`
		} `json:"reading_outstanding"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("capture board not JSON: %v\n%s", err, out)
	}
	if len(r.Outstanding.Unadmitted) != 1 || r.Outstanding.Unadmitted[0].Item != item {
		t.Fatalf("the board must carry the unadmitted proposal; got %+v\n%s", r.Outstanding, out)
	}
	text := string(runCLI(t, "capture"))
	if !strings.Contains(text, "unadmitted "+item) {
		t.Fatalf("the board must name the unadmitted proposal:\n%s", text)
	}
}

// TestCaptureProductionModeFlag proves the closed-choice flag reaches the
// ledger: a declared mode is stamped, an undeclared one takes the repo's
// default, and a value outside the vocabulary is refused with nothing written.
// The flag is not free text — that is what keeps itd-178's "neither key was
// supplied as free text by the operator" true while a flag exists at all.
func TestCaptureProductionModeFlag(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	rec := captureRecordFields(t, repo, "a finding worth stamping", "--production-mode", "dictated-and-formatted")
	if rec["production_mode"] != "dictated-and-formatted" {
		t.Errorf("production_mode = %q, want dictated-and-formatted", rec["production_mode"])
	}
	if rec["origin"] != "researcher-authored" {
		t.Errorf("origin = %q, want researcher-authored", rec["origin"])
	}

	rec = captureRecordFields(t, repo, "a finding with no declared mode")
	if rec["production_mode"] != "hand-written" {
		t.Errorf("defaulted production_mode = %q, want hand-written", rec["production_mode"])
	}

	before := ledgerIssueCount(t, repo)
	out, err := runCLIErr(t, "capture", "a finding with a bogus mode", "--production-mode", "typed")
	if err == nil {
		t.Fatalf("an out-of-vocabulary production mode must be refused:\n%s", out)
	}
	if n := ledgerIssueCount(t, repo); n != before {
		t.Errorf("a refused capture wrote a record: %d -> %d", before, n)
	}
}

// TestCaptureProductionModeDefaultsToThePin proves the itd-91 attribution seam
// is the source of the default: a repo that declares one in its identity pin
// stamps it without the operator naming it on every command.
func TestCaptureProductionModeDefaultsToThePin(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	pin := `{"name":"A Maintainer","email":"maintainer@example.com","production_mode":"scribe-transcribed"}` + "\n"
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config", "identity.json"), []byte(pin), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := captureRecordFields(t, repo, "a finding taking the repo default")
	if rec["production_mode"] != "scribe-transcribed" {
		t.Errorf("production_mode = %q, want the pin's scribe-transcribed", rec["production_mode"])
	}
}

// captureRecordFields files an issue through the CLI and returns the written
// record's frontmatter as key/value pairs, read off the bytes that landed.
func captureRecordFields(t *testing.T, repo, text string, extra ...string) map[string]string {
	t.Helper()
	args := append([]string{"capture", text, "--json"}, extra...)
	out := runCLI(t, args...)
	var r struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("capture --json: %v\n%s", err, out)
	}
	data, err := os.ReadFile(filepath.Join(repo, r.Path))
	if err != nil {
		t.Fatalf("reading %s: %v", r.Path, err)
	}
	fields := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		if line == "---" && len(fields) > 0 {
			break
		}
		k, v, ok := strings.Cut(line, ": ")
		if ok {
			fields[k] = v
		}
	}
	return fields
}

// cliGrounds is a conjecture-shaped operand for the capture surface tests.
const cliGrounds = "pursued: we expect the recorded reasoning to outlive the session that had it"

// TestCapturePromoteMissingGroundsRecordsNone: itd-179 refused an absent
// --grounds at exit 2; that refusal is parked (iss-2609091009111294). Promote and
// resolve without the flag complete, and neither writes a `## Grounds` entry the
// caller did not give. A MALFORMED value is still a usage error at exit 2
// (TestCaptureMalformedGroundsExit2).
func TestCapturePromoteMissingGroundsRecordsNone(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	capOut := runCLI(t, "capture", "an observation that may turn out to be a capability", "--json")
	var minted struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(capOut, &minted); err != nil || minted.ID == "" {
		t.Fatalf("capture envelope unreadable: %v\n%s", err, capOut)
	}

	runCLI(t, "capture", "promote", minted.ID)
	if entries, _ := os.ReadDir(filepath.Join(repo, cliDrafts)); len(entries) != 1 {
		t.Fatalf("a promote without grounds minted %d draft(s), want 1", len(entries))
	}
	runCLI(t, "capture", "resolve", minted.ID, "fixed", "--impact", "fix")
	out := runCLI(t, "capture", "list", "--resolved", "--json")
	if !strings.Contains(string(out), minted.ID) {
		t.Fatalf("a resolve without grounds left the issue out of resolved/:\n%s", out)
	}
	matches, _ := filepath.Glob(filepath.Join(repo, ".abcd", "work", "issues", "resolved", minted.ID+"-*.md"))
	if len(matches) != 1 {
		t.Fatalf("resolved record for %s: %d match(es)", minted.ID, len(matches))
	}
	body, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "## Grounds") {
		t.Fatalf("a triage without grounds invented a grounds entry:\n%s", body)
	}
}

// TestCaptureGroundsReachTheRecord: the wired flag actually records, on all three
// triage routes, and wontfix derives its `declined:` grounds from the reason it
// already takes.
//
// It asserts the BULLET in the record body, which is where grounds live: a
// frontmatter scalar is set, and setting is what let one route overwrite
// another's conjecture (iss-2608301657354776). The end-to-end path is what this
// test is for — a writer that appended somewhere the record's own section reader
// cannot see would still satisfy the core tests.
func TestCaptureGroundsReachTheRecord(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	mint := func(text string) string {
		t.Helper()
		var m struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(runCLI(t, "capture", text, "--json"), &m); err != nil || m.ID == "" {
			t.Fatalf("capture envelope unreadable: %v", err)
		}
		return m.ID
	}
	readRecord := func(id string) string {
		t.Helper()
		var listed struct {
			Issues []struct {
				ID   string `json:"id"`
				Path string `json:"path"`
			} `json:"issues"`
		}
		if err := json.Unmarshal(runCLI(t, "capture", "list", "--all", "--json"), &listed); err != nil {
			t.Fatal(err)
		}
		for _, iss := range listed.Issues {
			if iss.ID == id {
				data, err := os.ReadFile(filepath.Join(repo, iss.Path))
				if err != nil {
					t.Fatal(err)
				}
				return string(data)
			}
		}
		t.Fatalf("%s not found in the ledger", id)
		return ""
	}

	promoted := mint("an observation worth graduating into an intent")
	runCLI(t, "capture", "promote", promoted, "--grounds", cliGrounds)
	if !strings.Contains(readRecord(promoted), "\n## Grounds\n\n- "+cliGrounds+"\n") {
		t.Fatalf("promote did not record the grounds as a body bullet:\n%s", readRecord(promoted))
	}

	resolved := mint("an observation that will be fixed outright")
	runCLI(t, "capture", "resolve", resolved, "fixed", "--impact", "fix", "--grounds", cliGrounds)
	if !strings.Contains(readRecord(resolved), "\n## Grounds\n\n- "+cliGrounds+"\n") {
		t.Fatalf("resolve did not record the grounds as a body bullet:\n%s", readRecord(resolved))
	}

	declined := mint("an observation that will not be acted on")
	runCLI(t, "capture", "wontfix", declined, "out of scope for this cycle")
	if !strings.Contains(readRecord(declined), "\n## Grounds\n\n- declined: out of scope for this cycle\n") {
		t.Fatalf("wontfix did not derive its declined grounds as a body bullet:\n%s", readRecord(declined))
	}
}

// TestCaptureMalformedGroundsExit2 closes the uneven half of
// iss-2608300930057882: a MISSING --grounds exited 2 while a MALFORMED one
// exited 1, so a caller distinguishing usage errors from real failures learned
// the wrong thing from the same flag. Every grounds refusal is a usage error.
func TestCaptureMalformedGroundsExit2(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)

	capOut := runCLI(t, "capture", "an observation that may turn out to be a capability", "--json")
	var minted struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(capOut, &minted); err != nil || minted.ID == "" {
		t.Fatalf("capture envelope unreadable: %v\n%s", err, capOut)
	}

	for _, bad := range []string{
		"planned: out of vocabulary entirely",
		"no token at all in this operand",
		"pursued: yes",
	} {
		for _, args := range [][]string{
			{"capture", "promote", minted.ID, "--grounds", bad},
			{"capture", "resolve", minted.ID, "fixed", "--impact", "fix", "--grounds", bad},
			{"capture", "wontfix", minted.ID, "no", "--grounds", bad},
		} {
			_, err := runCLIErr(t, args...)
			if exitCodeOf(err) != 2 {
				t.Fatalf("%v with %q: exit = %d (%v), want 2", args[:2], bad, exitCodeOf(err), err)
			}
		}
	}
}

// TestGroundsFlagUsageRendersAStringPlaceholder: cobra's UnquoteUsage takes the
// first backquoted word of a flag's usage string as the flag's value
// placeholder and strips it from the prose, so backticks in the wontfix
// `--grounds` help printed `--grounds declined` and lost the word
// (iss-2608301212428844). Every grounds flag names a string.
func TestGroundsFlagUsageRendersAStringPlaceholder(t *testing.T) {
	for _, verb := range []string{"promote", "resolve", "wontfix"} {
		out, err := runCLIErr(t, "capture", verb, "--help")
		if err != nil {
			t.Fatalf("capture %s --help: %v\n%s", verb, err, out)
		}
		if !strings.Contains(string(out), "--grounds string") {
			t.Fatalf("capture %s --help renders no `--grounds string` placeholder:\n%s", verb, out)
		}
	}
}

// TestCapturePromoteReadingItemNeedsNoGrounds pins the one route the mandatory
// --grounds flag does not govern. A reading item's reasoning is recorded in its
// DISPOSITION — a separate record promote already refuses to act without — and
// the reading promote path writes no grounds of its own, so demanding the flag
// here would collect a conjecture nothing stores. The issue route, asserted by
// TestCapturePromoteMissingGroundsExit2, is unaffected.
func TestCapturePromoteReadingItemNeedsNoGrounds(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	const run, item = "rdg-2608300000000001", "rdi-2608300000000002"
	writeReadingFixture(t, repo, run, item)
	runCLI(t, "capture", "disposition", item,
		"--state", "accepted", "--grounds", "the tension is real and worth acting on")

	out := runCLI(t, "capture", "promote", item, "--json")
	var r struct {
		IntentID string `json:"intent_id"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("promote output not JSON: %v\n%s", err, out)
	}
	if r.IntentID == "" {
		t.Fatalf("promoting a dispositioned reading item without --grounds must mint a draft:\n%s", out)
	}
}

// TestCapturePromoteReadingItemStampsTheOriginPair — framework 11.3 (linkage),
// through the front door: the JSON's `intent_path` names a file whose `origin`
// line carries the run and the item, so the CLI is shown executing the core
// change rather than assumed to. Framework 7.1 keeps the value out of every
// flag: it is derived from which command ran.
func TestCapturePromoteReadingItemStampsTheOriginPair(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	const run, item = "rdg-2608300000000001", "rdi-2608300000000002"
	writeReadingFixture(t, repo, run, item)
	runCLI(t, "capture", "disposition", item,
		"--state", "accepted", "--grounds", "the tension is real and worth acting on")

	out := runCLI(t, "capture", "promote", item, "--json")
	var r struct {
		IntentPath   string `json:"intent_path"`
		BackEdgeKept string `json:"back_edge_kept"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("promote output not JSON: %v\n%s", err, out)
	}
	if r.BackEdgeKept != "" {
		t.Errorf("a minted draft keeps no prior back-edge, got %q", r.BackEdgeKept)
	}
	data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(r.IntentPath)))
	if err != nil {
		t.Fatalf("reading the minted draft at %s: %v", r.IntentPath, err)
	}
	want := "\norigin: contributed-by-reading " + run + "/" + item + "\n"
	if !strings.Contains(string(data), want) {
		t.Fatalf("the minted draft carries no %q:\n%s", strings.TrimSpace(want), data)
	}
	if !strings.Contains(string(data), "\npromoted_from: "+item+"\n") {
		t.Fatalf("the minted draft carries no back-edge naming %s:\n%s", item, data)
	}
	// The seed names no item: the Press Release is projected to the entailment
	// reading, and no reading sees another's output (companion 8.3).
	press, _, _ := strings.Cut(after(t, string(data), "## Press Release"), "\n## ")
	if strings.Contains(press, "rdi-") {
		t.Fatalf("the promoted draft's Press Release names a reading item:\n%s", press)
	}
}

// TestCapturePromoteReadingItemLinkReportsAKeptBackEdge — the first scope
// condition, on the surface: a draft already promoted from another item keeps
// that edge, and BOTH renderings say which record stayed.
func TestCapturePromoteReadingItemLinkReportsAKeptBackEdge(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	const run, first, second = "rdg-2608300000000001", "rdi-2608300000000002", "rdi-2608300000000003"
	const third = "rdi-2608300000000004"
	writeReadingFixture(t, repo, run, first)
	writeReadingFixture(t, repo, run, second)
	writeReadingFixture(t, repo, run, third)
	for _, item := range []string{first, second, third} {
		runCLI(t, "capture", "disposition", item,
			"--state", "accepted", "--grounds", "the tension is real and worth acting on")
	}

	firstOut := runCLI(t, "capture", "promote", first, "--json")
	var minted struct {
		IntentID string `json:"intent_id"`
	}
	if err := json.Unmarshal(firstOut, &minted); err != nil {
		t.Fatalf("promote output not JSON: %v\n%s", err, firstOut)
	}

	out := runCLI(t, "capture", "promote", second, "--intent", minted.IntentID, "--json")
	var linked struct {
		BackEdgeKept string `json:"back_edge_kept"`
	}
	if err := json.Unmarshal(out, &linked); err != nil {
		t.Fatalf("link output not JSON: %v\n%s", err, out)
	}
	if linked.BackEdgeKept != first {
		t.Fatalf("the JSON rendering reports back_edge_kept %q, want %q", linked.BackEdgeKept, first)
	}
	// The item still points forward even though the draft's one back-edge stayed.
	itemPath := filepath.Join(repo, ".abcd", "work", "issues", "readings", run, second+".md")
	data, err := os.ReadFile(itemPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), minted.IntentID) {
		t.Fatalf("the second item was not stamped forward:\n%s", data)
	}

	// The human rendering carries the same report.
	line := string(runCLI(t, "capture", "promote", third, "--intent", minted.IntentID))
	if !strings.Contains(line, "back_edge: kept "+first) {
		t.Fatalf("the human rendering does not report the kept back-edge:\n%s", line)
	}
}

// after returns the text following heading in doc, for a section-level read.
func after(t *testing.T, doc, heading string) string {
	t.Helper()
	_, rest, ok := strings.Cut(doc, heading+"\n")
	if !ok {
		t.Fatalf("the record carries no %q section:\n%s", heading, doc)
	}
	return rest
}

// TestCapturePromoteReadingItemRefusesGrounds is the other half of the exemption
// above: the reading route does not merely stop REQUIRING the flag, it refuses a
// value handed to it. Nothing on this route writes grounds, so accepting one
// would report success over a conjecture that reached no record.
func TestCapturePromoteReadingItemRefusesGrounds(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	const run, item = "rdg-2608300000000001", "rdi-2608300000000002"
	writeReadingFixture(t, repo, run, item)
	runCLI(t, "capture", "disposition", item,
		"--state", "accepted", "--grounds", "the tension is real and worth acting on")

	_, err := runCLIErr(t, "capture", "promote", item, "--grounds", cliGrounds)
	if exitCodeOf(err) != 2 {
		t.Fatalf("exit = %d (%v), want 2", exitCodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "disposition") {
		t.Fatalf("the refusal must say where the conjecture belongs, got %q", err.Error())
	}
	if entries, _ := os.ReadDir(filepath.Join(repo, cliDrafts)); len(entries) != 0 {
		t.Fatalf("a refused promote minted %d draft(s), want 0", len(entries))
	}
}
