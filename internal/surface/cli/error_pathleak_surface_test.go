package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestJSONErrorEnvelopeNoAbsolutePathLeak is the iss-76 detector: cli.Run routes
// every command error through the --json envelope (and the stderr text line), so
// any verb whose error chain carries an *os.PathError / *os.LinkError would leak
// its absolute local path into machine output. The sanitisation belongs at the
// Run() boundary — iss-29 fixed only the docs-lint config-load branch. This is a
// per-verb table: each row drives a real verb into a filesystem error and asserts
// the envelope carries no absolute path but keeps the file's base name for context.
func TestJSONErrorEnvelopeNoAbsolutePathLeak(t *testing.T) {
	repo := t.TempDir()
	// A bare temporary directory is no longer a place a record store is
	// addressed: the memory front door resolves the checkout root first and
	// refuses outside one (iss-2609091729516940).
	gitInitAt(t, repo)
	t.Chdir(repo)

	// An absolute path guaranteed to fail os.ReadFile with a *PathError.
	absMissing := filepath.Join(repo, "no-such-dir", "page.json")

	cases := []struct {
		name string
		args []string
		base string // basename the sanitised message should retain
	}{
		{
			name: "memory ask --page-json missing file",
			args: []string{"memory", "ask", "q", "--file-back", "--page-json", absMissing, "--json"},
			base: "page.json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(tc.args, &stdout, &stderr)
			if code == 0 {
				t.Fatalf("expected a non-zero exit; stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			var env struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
				t.Fatalf("--json error not JSON-shaped: %v\nstdout: %q", err, stdout.String())
			}
			if strings.Contains(env.Error, repo) {
				t.Fatalf("envelope leaked the absolute path %q:\n%s", repo, env.Error)
			}
			if !strings.Contains(env.Error, tc.base) {
				t.Fatalf("envelope dropped all file context (want basename %q):\n%s", tc.base, env.Error)
			}
		})
	}
}

// TestJSONSuccessEnvelopeNoAbsolutePathLeak is the iss-81 detector: a path-echoing
// verb's SUCCESS envelope must not carry an absolute developer-identity path either.
// The iss-76 scrub only sanitises the ERROR surface, so a verb that renders a
// filesystem locator on the success path (capture's `path` field; resolve/wontfix's
// `path`; list's per-issue `path`) leaks the absolute repo path straight into machine
// output. The contract is repo-relative everywhere: the field stays a useful locator
// but is never absolute.
func TestJSONSuccessEnvelopeNoAbsolutePathLeak(t *testing.T) {
	// prepCapture mints a fixture issue and returns its id (the mint is
	// timestamp-numeric, so the id must be read from the envelope, never assumed).
	prepCapture := func(t *testing.T, text string) string {
		t.Helper()
		var so, se bytes.Buffer
		if code := Run([]string{"capture", text, "--slug", "detector-fixture", "--json"}, &so, &se); code != 0 {
			t.Fatalf("prep capture failed (code %d): %s", code, se.String())
		}
		var res struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(so.Bytes(), &res); err != nil || res.ID == "" {
			t.Fatalf("prep capture envelope unreadable: %v\n%s", err, so.String())
		}
		return res.ID
	}
	cases := []struct {
		name string
		// prep establishes state a verb needs and returns the args, so a verb
		// that targets an issue can name the freshly minted id.
		args func(t *testing.T) []string
	}{
		{
			name: "capture success path",
			args: func(t *testing.T) []string { return []string{"capture", "a defect", "--json"} },
		},
		{
			name: "resolve success path",
			args: func(t *testing.T) []string {
				id := prepCapture(t, "to resolve")
				return []string{"capture", "resolve", id, "handled", "--impact", "fix",
					"--grounds", "pursued: we expect the recorded reasoning to outlive the session", "--json"}
			},
		},
		{
			name: "wontfix success path",
			args: func(t *testing.T) []string {
				id := prepCapture(t, "to wontfix")
				return []string{"capture", "wontfix", id, "declined", "--json"}
			},
		},
		{
			name: "list issue path",
			args: func(t *testing.T) []string {
				prepCapture(t, "to list")
				return []string{"capture", "list", "--all", "--json"}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := captureLedgerRepo(t)
			args := tc.args(t)
			var stdout, stderr bytes.Buffer
			code := Run(args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("expected a zero exit; code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			// Every "path" string anywhere in the success envelope must be
			// repo-relative: never absolute, never containing the repo root.
			paths := collectPaths(t, stdout.Bytes())
			if len(paths) == 0 {
				t.Fatalf("no path field found in the success envelope:\n%s", stdout.String())
			}
			for _, p := range paths {
				if filepath.IsAbs(p) {
					t.Fatalf("success envelope leaked an absolute path %q:\n%s", p, stdout.String())
				}
				if strings.Contains(p, repo) {
					t.Fatalf("success envelope leaked the repo root inside %q:\n%s", p, stdout.String())
				}
			}
		})
	}
}

// collectPaths walks a JSON success envelope and returns every value of a field
// named "path" (case-insensitive), at any nesting depth, so the detector can
// assert none is absolute.
func collectPaths(t *testing.T, raw []byte) []string {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("success output not JSON-shaped: %v\noutput: %q", err, string(raw))
	}
	var out []string
	var walk func(any)
	walk = func(n any) {
		switch node := n.(type) {
		case map[string]any:
			for k, val := range node {
				if s, ok := val.(string); ok && strings.EqualFold(k, "path") && s != "" {
					out = append(out, s)
				}
				walk(val)
			}
		case []any:
			for _, e := range node {
				walk(e)
			}
		}
	}
	walk(v)
	return out
}

// TestMemoryIngestSuccessEnvelopeNoAbsolutePathLeak is the iss-81 detector for the
// STRONGER surface: a successful `memory ingest` builds its citation from the
// source's origin, which was the absolute EvalSymlinks-resolved path — so the
// success `--json` envelope emitted citation.origin as an absolute (and, for a
// ~/… source, home-rooted) developer path. Every string in the success envelope
// must be free of an absolute path and of the repo root.
func TestMemoryIngestSuccessEnvelopeNoAbsolutePathLeak(t *testing.T) {
	repo := t.TempDir()
	gitInitAt(t, repo)
	// Spelled the way the resolver spells it: the front door renders paths
	// against git's toplevel, which is symlink-resolved, so a fixture holding
	// the /var form would compare against a root spelled /private/var.
	repo = realPath(t, repo)
	t.Chdir(repo)
	src := filepath.Join(repo, "article.txt")
	if err := os.WriteFile(src, []byte("Rotate tokens every 24 hours.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pages := filepath.Join(repo, "pages.json")
	if err := os.WriteFile(pages, []byte(`[{"type":"topic","domain":"auth","slug":"tokens","body":"# Rotation\nRotate tokens every 24 hours."}]`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"memory", "ingest", src, "--pages-json", pages, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected a zero exit; code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	// The citation.origin must exist (so this assertion is not vacuous) and be
	// repo-relative like every other string in the envelope.
	var env struct {
		Citation map[string]any `json:"citation"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("success output not JSON-shaped: %v\n%s", err, stdout.String())
	}
	origin, _ := env.Citation["origin"].(string)
	if origin == "" {
		t.Fatalf("citation.origin missing — detector would be vacuous:\n%s", stdout.String())
	}
	for _, s := range collectStrings(t, stdout.Bytes()) {
		if filepath.IsAbs(s) {
			t.Fatalf("success envelope leaked an absolute path %q:\n%s", s, stdout.String())
		}
		if strings.Contains(s, repo) {
			t.Fatalf("success envelope leaked the repo root inside %q:\n%s", s, stdout.String())
		}
	}
}

// collectStrings walks a JSON document and returns every string value at any
// depth (map values and array elements), so a detector can assert none carries
// an absolute path.
func collectStrings(t *testing.T, raw []byte) []string {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("output not JSON-shaped: %v\noutput: %q", err, string(raw))
	}
	var out []string
	var walk func(any)
	walk = func(n any) {
		switch node := n.(type) {
		case string:
			out = append(out, node)
		case map[string]any:
			for _, val := range node {
				walk(val)
			}
		case []any:
			for _, e := range node {
				walk(e)
			}
		}
	}
	walk(v)
	return out
}

// TestMemoryIngestErrorNoAbsolutePathLeakOutsideRoots is the iss-81 detector for the
// ingest error path: materialFromLocal embeds an EvalSymlinks-resolved ABSOLUTE
// source path in its IngestError. When the source lies OUTSIDE cwd and home, the
// iss-76 scrub (which only redacts those two roots) cannot see it, so the absolute
// path reaches the --json error envelope. The path must be rendered repo-relative.
func TestMemoryIngestErrorNoAbsolutePathLeakOutsideRoots(t *testing.T) {
	repo := t.TempDir()
	gitInitAt(t, repo)
	repo = realPath(t, repo)
	t.Chdir(repo)
	// A source outside both cwd (repo) and home — the scrub's two roots.
	outside := realPath(t, t.TempDir())
	missing := filepath.Join(outside, "secret-notes.txt")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"memory", "ingest", missing, "--json"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected a non-zero exit for a missing source; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	var env struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("--json error not JSON-shaped: %v\nstdout: %q", err, stdout.String())
	}
	if strings.Contains(env.Error, outside) {
		t.Fatalf("ingest envelope leaked the absolute source path %q:\n%s", outside, env.Error)
	}
	if !strings.Contains(env.Error, "secret-notes.txt") {
		t.Fatalf("ingest envelope dropped the file basename for context:\n%s", env.Error)
	}
}

// TestCaptureSymlinkErrorNoPathLeak reproduces the security-review finding that
// the fix must also cover: capture's allocator embeds an ABSOLUTE ledger path via
// fmt.Errorf("%w … %s") — not an os.PathError — so a typed walk misses it. A
// symlinked issues/open dir trips the guard and its error reaches the --json
// envelope; it must not leak the developer's absolute repo path.
func TestCaptureSymlinkErrorNoPathLeak(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	issues := filepath.Join(repo, ".abcd", "work", "issues")
	if err := os.MkdirAll(issues, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(issues, "open")); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"capture", "a defect", "--json"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected a non-zero exit for a symlinked issues/open; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	var env struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("--json error not JSON-shaped: %v\nstdout: %q", err, stdout.String())
	}
	if strings.Contains(env.Error, repo) {
		t.Fatalf("capture envelope leaked the absolute repo path %q:\n%s", repo, env.Error)
	}
}

// homeRootedErr is a custom error type (like history.StorePathError) that renders
// a home-dir-rooted absolute path — the class a typed PathError walk cannot see.
type homeRootedErr struct{ path string }

func (e homeRootedErr) Error() string { return "history: store unreadable (" + e.path + ")" }

// TestScrubPaths is the unit-level detector for the Run()-boundary sanitiser. It
// removes local-identity paths reaching machine output three ways — os.PathError/
// os.LinkError (→ base name), and cwd- or home-rooted paths embedded by fmt or a
// custom error type (→ "." / "~") — while leaving relative and non-path errors
// untouched.
func TestScrubPaths(t *testing.T) {
	abs := filepath.Join(os.TempDir(), "secret", "config.json")
	rel := filepath.Join("rel", "config.json")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name        string
		err         error
		wantAbsent  string // must NOT appear in the scrubbed message
		wantPresent []string
	}{
		{
			name:        "fmt-embedded cwd path (capture class)",
			err:         fmt.Errorf("path unsafe: allocator lock path is a symlink: %s", filepath.Join(cwd, ".abcd", "work", "issues", ".iss-alloc.lock")),
			wantAbsent:  cwd,
			wantPresent: []string{".iss-alloc.lock"},
		},
		{
			name:        "custom-type home path (history class)",
			err:         homeRootedErr{path: filepath.Join(home, ".abcd", "history", "x")},
			wantAbsent:  home,
			wantPresent: []string{".abcd", "history: store unreadable"},
		},
		{
			// B39: a message naming exactly $HOME with no trailing separator must
			// still be redacted — its base segment is the username.
			name:        "fmt-embedded bare home path",
			err:         fmt.Errorf("cannot access %s (permission denied)", home),
			wantAbsent:  home,
			wantPresent: []string{"cannot access", "~"},
		},
		{
			// B39: a PathError whose Path IS exactly $HOME must not fall back to
			// filepath.Base(home) == the username.
			name:        "PathError equal to home",
			err:         &os.PathError{Op: "open", Path: home, Err: fs.ErrPermission},
			wantAbsent:  home,
			wantPresent: []string{"open", "~"},
		},
		{
			name:        "bare PathError",
			err:         &os.PathError{Op: "open", Path: abs, Err: fs.ErrNotExist},
			wantAbsent:  abs,
			wantPresent: []string{"open", "config.json", fs.ErrNotExist.Error()},
		},
		{
			name:        "wrapped PathError keeps context",
			err:         fmt.Errorf("cannot read --page-json: %w", &os.PathError{Op: "open", Path: abs, Err: fs.ErrPermission}),
			wantAbsent:  abs,
			wantPresent: []string{"cannot read --page-json", "config.json"},
		},
		{
			name:        "LinkError both paths",
			err:         &os.LinkError{Op: "rename", Old: abs, New: abs + ".tmp", Err: fs.ErrExist},
			wantAbsent:  abs,
			wantPresent: []string{"rename", "config.json"},
		},
		{
			name:        "joined errors both scrubbed",
			err:         errors.Join(&os.PathError{Op: "open", Path: abs, Err: fs.ErrNotExist}, fmt.Errorf("and: %w", &os.PathError{Op: "stat", Path: abs, Err: fs.ErrPermission})),
			wantAbsent:  abs,
			wantPresent: []string{"config.json"},
		},
		{
			name:        "relative PathError left intact",
			err:         &os.PathError{Op: "open", Path: rel, Err: fs.ErrNotExist},
			wantAbsent:  "\x00-never-", // sentinel: nothing to strip
			wantPresent: []string{rel},
		},
		{
			name:        "non-path error untouched",
			err:         errors.New("plain failure"),
			wantAbsent:  "\x00-never-",
			wantPresent: []string{"plain failure"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := scrubPaths(tc.err)
			if strings.Contains(got, tc.wantAbsent) {
				t.Fatalf("scrubPaths kept the absolute path %q: %s", tc.wantAbsent, got)
			}
			for _, want := range tc.wantPresent {
				if !strings.Contains(got, want) {
					t.Fatalf("scrubPaths dropped %q: %s", want, got)
				}
			}
		})
	}
}
