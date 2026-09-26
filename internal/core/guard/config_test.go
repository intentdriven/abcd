package guard

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// writeOverride lays down a repo root with .abcd/guard.json and returns the root.
func writeOverride(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "guard.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// writeCommittedOverride is writeOverride in a real repository with the file
// committed: an override that weakens a blocker or switches the guard off takes
// effect only once HEAD carries it (iss-147), so a test of what such an override
// DOES has to commit it first.
func writeCommittedOverride(t *testing.T, body string) string {
	t.Helper()
	repo := gittest.NewRepo(t)
	repo.Write(RepoRelPath, body)
	repo.Commit("guard override")
	return repo.Root()
}

func TestLoadWithoutOverrideReturnsDefaults(t *testing.T) {
	r, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("an absent %s must yield the defaults, got %v", RepoRelPath, err)
	}
	if len(r.Entries) != len(Defaults().Entries) {
		t.Fatalf("Entries = %d, want the %d bundled entries", len(r.Entries), len(Defaults().Entries))
	}
}

// TestLoadOverridesEntryPerField proves the committed override is the only
// escape hatch AND that it is per-field: retiering an entry keeps its pattern,
// successor, and why.
func TestLoadOverridesEntryPerField(t *testing.T) {
	root := writeOverride(t, `{"schema_version":1,"entries":{"git-reset-hard":{"tier":"blocker"}}}`)
	r, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	e := r.Entries["git-reset-hard"]
	if e.Tier != TierBlocker {
		t.Fatalf("tier = %q, want the override's %q", e.Tier, TierBlocker)
	}
	if e.Why != Defaults().Entries["git-reset-hard"].Why || e.Successor == "" {
		t.Fatal("an unset override field must inherit the bundled entry")
	}
	d, err := r.Check("git reset --hard origin/main")
	if err != nil {
		t.Fatal(err)
	}
	if d.Verdict != VerdictBlock {
		t.Fatalf("verdict = %q, want the retiered %q", d.Verdict, VerdictBlock)
	}
}

// TestLoadOverridesPatternPerField pins the pointer semantics of after_cd: an
// override can turn the cd-chain requirement OFF as well as on.
func TestLoadOverridesPatternPerField(t *testing.T) {
	root := writeCommittedOverride(t, `{"schema_version":1,"entries":{"rm-rf-after-cd-chain":{"pattern":{"after_cd":false}}}}`)
	r, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.Entries["rm-rf-after-cd-chain"].Pattern.Command; got != "rm" {
		t.Fatalf("pattern command = %q, want the inherited %q", got, "rm")
	}
	d, err := r.Check("rm -rf ./build")
	if err != nil {
		t.Fatal(err)
	}
	if d.Verdict != VerdictBlock {
		t.Fatalf("verdict = %q, want %q once the cd-chain requirement is lifted", d.Verdict, VerdictBlock)
	}
}

func TestLoadAddsRepoEntry(t *testing.T) {
	root := writeOverride(t, `{"schema_version":1,"entries":{"drop-database":{
		"tier":"blocker",
		"pattern":{"command":"dropdb"},
		"why":"It deletes the whole database.",
		"successor":"Take a dump first."}}}`)
	r, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	d, err := r.Check("dropdb production")
	if err != nil {
		t.Fatal(err)
	}
	if d.Verdict != VerdictBlock || d.EntryID != "drop-database" {
		t.Fatalf("a repo-added entry must fire, got %+v", d)
	}
	if len(r.Entries) != len(Defaults().Entries)+1 {
		t.Fatal("adding an entry must not drop the bundled ones")
	}
}

// TestLoadAcceptsADeclaredEntryID keeps the documented entry shape loadable: an
// override may spell out the id it is already keyed by. The map key stays
// authoritative — a mismatched id must not rename the entry.
func TestLoadAcceptsADeclaredEntryID(t *testing.T) {
	root := writeOverride(t, `{"schema_version":1,"entries":{"git-clean":{"id":"git-clean","tier":"blocker"}}}`)
	r, err := Load(root)
	if err != nil {
		t.Fatalf("an entry declaring its own id must load, got %v", err)
	}
	if r.Entries["git-clean"].ID != "git-clean" || r.Entries["git-clean"].Tier != TierBlocker {
		t.Fatalf("entry = %+v, want the keyed id and the override's tier", r.Entries["git-clean"])
	}
}

func TestLoadHonoursCommittedKillSwitch(t *testing.T) {
	root := writeCommittedOverride(t, `{"schema_version":1,"disabled":true}`)
	r, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	d, err := r.Check("cd scratch && rm -rf *")
	if err != nil {
		t.Fatal(err)
	}
	if d.Verdict != VerdictAllow {
		t.Fatalf("a committed kill switch must disable the guard, got %+v", d)
	}
}

func TestLoadRejectsBadConfig(t *testing.T) {
	tests := []struct {
		name string
		body string
		want error
	}{
		{
			name: "unknown tier",
			body: `{"schema_version":1,"entries":{"git-clean":{"tier":"fatal"}}}`,
			want: ErrUnknownTier,
		},
		{
			name: "unsupported schema version",
			body: `{"schema_version":2,"entries":{}}`,
			want: ErrSchemaVersion,
		},
		{
			name: "absent schema version",
			body: `{"entries":{}}`,
			want: ErrSchemaVersion,
		},
		{
			name: "malformed json",
			body: `{"schema_version":1,`,
			want: ErrMalformedConfig,
		},
		{
			name: "new entry without a successor",
			body: `{"schema_version":1,"entries":{"drop-database":{"tier":"blocker","pattern":{"command":"dropdb"},"why":"It deletes everything."}}}`,
			want: ErrInvalidEntry,
		},
		{
			name: "new entry without a pattern command",
			body: `{"schema_version":1,"entries":{"drop-database":{"tier":"blocker","why":"It deletes everything.","successor":"Take a dump first."}}}`,
			want: ErrInvalidEntry,
		},
		{
			// An empty flag group can never be satisfied, so it would silently
			// defang the entry it belongs to — the exact failure a guard must
			// never have (a guard that stops refusing looks identical to one
			// with nothing to refuse).
			name: "flag group that can never match",
			body: `{"schema_version":1,"entries":{"git-clean":{"pattern":{"flags":["-f","|"]}}}}`,
			want: ErrInvalidEntry,
		},
		{
			// The command is compared against a token's basename, so a path or a
			// phrase could never be satisfied — the same silent defang as an
			// empty flag group.
			name: "pattern command that could never match",
			body: `{"schema_version":1,"entries":{"drop-database":{"tier":"blocker","pattern":{"command":"/usr/bin/dropdb"},"why":"It deletes everything.","successor":"Take a dump first."}}}`,
			want: ErrInvalidEntry,
		},
		{
			name: "subcommand that could never match",
			body: `{"schema_version":1,"entries":{"drop-database":{"tier":"blocker","pattern":{"command":"dropdb","subcommand":"--all"},"why":"It deletes everything.","successor":"Take a dump first."}}}`,
			want: ErrInvalidEntry,
		},
		{
			// A misspelt top-level key would otherwise be dropped in silence, and
			// the repo's intended blocker would simply not exist.
			name: "unrecognised key",
			body: `{"schema_version":1,"entrys":{"git-clean":{"tier":"blocker"}}}`,
			want: ErrMalformedConfig,
		},
		{
			// An empty argument prefix is carried by EVERY operand, so the entry
			// would fire on any command that reached it — the over-blocking twin
			// of the empty flag group, and just as invisible in the file.
			name: "argument prefix that matches everything",
			body: `{"schema_version":1,"entries":{"git-push-force":{"pattern":{"arg_prefixes":["+",""]}}}}`,
			want: ErrInvalidEntry,
		},
		{
			// `operands` only ever returns tokens that do NOT start with a dash,
			// so a prefix that starts with one describes an argument nothing can
			// be — the same silent defang as an empty flag group, one field along.
			name: "argument prefix that could never match",
			body: `{"schema_version":1,"entries":{"git-push-force":{"pattern":{"arg_prefixes":["-f"]}}}}`,
			want: ErrInvalidEntry,
		},
		{
			// A root is compared against the FIRST segment of a path split on
			// "/", so a root carrying a slash could never be one.
			name: "path root that could never match",
			body: `{"schema_version":1,"entries":{"gh-api-repo-delete":{"pattern":{"arg_paths":[{"root":"repos/owner","segments":3}]}}}}`,
			want: ErrInvalidEntry,
		},
		{
			// A flag-value constraint with no accepted value can never be
			// satisfied: the entry would look armed and refuse nothing.
			name: "flag-value constraint with no value",
			body: `{"schema_version":1,"entries":{"gh-api-repo-delete":{"pattern":{"flag_values":[{"flag":"-X|--method","values":[]}]}}}}`,
			want: ErrInvalidEntry,
		},
		{
			name: "flag-value constraint with no flag",
			body: `{"schema_version":1,"entries":{"gh-api-repo-delete":{"pattern":{"flag_values":[{"flag":"","values":["DELETE"]}]}}}}`,
			want: ErrInvalidEntry,
		},
		{
			// Zero segments describe no path at all, and a rootless constraint
			// would depth-limit every operand that happened to be a path.
			name: "path constraint with no depth",
			body: `{"schema_version":1,"entries":{"gh-api-repo-delete":{"pattern":{"arg_paths":[{"root":"repos","segments":0}]}}}}`,
			want: ErrInvalidEntry,
		},
		{
			name: "path constraint with no root",
			body: `{"schema_version":1,"entries":{"gh-api-repo-delete":{"pattern":{"arg_paths":[{"root":"","segments":3}]}}}}`,
			want: ErrInvalidEntry,
		},
		{
			name: "entry id that could build a path",
			body: `{"schema_version":1,"entries":{"../escape":{"tier":"warn","pattern":{"command":"x"},"why":"w","successor":"s"}}}`,
			want: ErrInvalidEntry,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(writeOverride(t, tc.body))
			if !errors.Is(err, tc.want) {
				t.Fatalf("Load error = %v, want %v", err, tc.want)
			}
			if strings.Contains(err.Error(), os.TempDir()) {
				t.Fatalf("error text leaks an absolute local path: %v", err)
			}
		})
	}
}

// TestLoadFallsBackToBundledDefaultsOnBrokenRepoLayer pins iss-2608261551087492:
// a malformed/oversized repo guard.json must NOT drop the bundled hazards. When
// only the repo override layer is broken, Load falls back to the bundled defaults
// (fail-SAFE) — the built-in hazards stay armed and keep blocking — rather than
// returning an empty registry that would disable the whole guard (fail-open). The
// error is still returned so the broken repo layer can be announced.
func TestLoadFallsBackToBundledDefaultsOnBrokenRepoLayer(t *testing.T) {
	broken := []struct {
		name string
		body string
	}{
		{"malformed json", `{"schema_version":1,`},
		{"unknown tier", `{"schema_version":1,"entries":{"git-clean":{"tier":"fatal"}}}`},
		{"absent schema version", `{"entries":{}}`},
		{"unrecognised key", `{"schema_version":1,"entrys":{"git-clean":{"tier":"blocker"}}}`},
		{"oversize", `{"schema_version":1,"entries":{}}` + strings.Repeat(" ", maxGuardFileBytes)},
	}
	for _, tc := range broken {
		t.Run(tc.name, func(t *testing.T) {
			r, err := Load(writeOverride(t, tc.body))
			// The repo layer failing is still reported (so it can be announced),
			// but the returned registry must be the armed bundled defaults, never
			// an empty one.
			if err == nil {
				t.Fatal("a broken repo layer must still be reported as an error")
			}
			if len(r.Entries) != len(Defaults().Entries) {
				t.Fatalf("Entries = %d, want the %d bundled entries kept armed on a broken repo layer",
					len(r.Entries), len(Defaults().Entries))
			}
			if _, ok := r.Entries["git-commit-no-verify"]; !ok {
				t.Fatal("the bundled git-commit-no-verify hazard was dropped by a broken repo layer")
			}
			if r.Disabled {
				t.Fatal("a broken repo layer must not disable the guard")
			}
			d, cerr := r.Check(`git commit --no-verify -m "wip"`)
			if cerr != nil {
				t.Fatal(cerr)
			}
			if d.Verdict != VerdictBlock {
				t.Fatalf("verdict = %q, want the bundled hazard to still BLOCK when the repo config is broken", d.Verdict)
			}
		})
	}
}

// TestLoadAcceptsWellFormedOperandConstraints is the other half of the two
// operand checks above: the shapes they reject are the ones that could never
// match, and the ordinary shapes must still load. A validator that rejected
// `arg_prefixes: ["+"]` or a plain path root would take the fields out of use
// entirely.
func TestLoadAcceptsWellFormedOperandConstraints(t *testing.T) {
	for _, body := range []string{
		`{"schema_version":1,"entries":{"git-push-force":{"pattern":{"arg_prefixes":["+"]}}}}`,
		`{"schema_version":1,"entries":{"gh-api-repo-delete":{"pattern":{"arg_paths":[{"root":"repos","segments":3}]}}}}`,
	} {
		if _, err := Load(writeCommittedOverride(t, body)); err != nil {
			t.Fatalf("a well-formed operand constraint must load, got %v for %s", err, body)
		}
	}
}

// TestLoadRefusesSymlinkedGuardFile keeps the trust-boundary read honest: the
// override is repo-tree content, so a symlinked leaf is refused rather than
// followed to an arbitrary target.
func TestLoadRefusesSymlinkedGuardFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "real-guard.json")
	if err := os.WriteFile(target, []byte(`{"schema_version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, ".abcd", "guard.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); !errors.Is(err, ErrMalformedConfig) {
		t.Fatalf("a symlinked guard.json leaf must be refused, not followed; got %v", err)
	}
}

func TestLoadRefusesSymlinkedAbcdDir(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "elsewhere")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "guard.json"), []byte(`{"schema_version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(dir, ".abcd")); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); !errors.Is(err, ErrMalformedConfig) {
		t.Fatalf("a symlinked .abcd directory must be refused, not followed; got %v", err)
	}
}

func TestLoadRefusesOversizeConfig(t *testing.T) {
	root := writeOverride(t, `{"schema_version":1,"entries":{}}`+strings.Repeat(" ", maxGuardFileBytes))
	if _, err := Load(root); !errors.Is(err, ErrMalformedConfig) {
		t.Fatalf("Load error = %v, want ErrMalformedConfig for an oversize file", err)
	}
}

// TestLoadDoesNotBlockOnAFifo proves the read cannot wedge a hook: the guard is
// called on the execution path of every command, so a non-regular leaf must be
// refused promptly rather than blocking the open forever.
func TestLoadDoesNotBlockOnAFifo(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(dir, ".abcd", "guard.json"), 0o644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := Load(dir)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("a FIFO guard.json must be refused, not read")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Load hung on a FIFO guard.json leaf (must never block the guard hook)")
	}
}
