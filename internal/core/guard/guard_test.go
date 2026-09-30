package guard

import (
	"errors"
	"strings"
	"testing"
)

// checkOK runs Check against the bundled defaults, failing the test on a
// tokenizer error (the cases that expect one assert it directly).
func checkOK(t *testing.T, command string) Decision {
	t.Helper()
	d, err := Defaults().Check(command)
	if err != nil {
		t.Fatalf("Check(%q): unexpected error: %v", command, err)
	}
	return d
}

func TestCheckDecisions(t *testing.T) {
	tests := []struct {
		name    string
		command string
		verdict Verdict
		entryID string // expected winning entry ("" when allowed)
	}{
		{
			name:    "cd chain then rm -rf is blocked",
			command: "cd scratch && rm -rf *",
			verdict: VerdictBlock,
			entryID: "rm-rf-after-cd-chain",
		},
		{
			name:    "hazard quoted inside an incident capture never fires",
			command: `./bin/abcd-darwin-arm64 capture "agent ran cd scratch && rm -rf * — one failed cd from disaster"`,
			verdict: VerdictAllow,
		},
		{
			name:    "rm -rf without a cd chain is allowed",
			command: "rm -rf ./build",
			verdict: VerdictAllow,
		},
		{
			name:    "a quoted flag still fires (quoting is not a bypass)",
			command: `git push '--force' origin main`,
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "a glued redirection does not hide the flag",
			command: "git push --force>/dev/null",
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "a glued redirection with a real target still fires",
			command: "git push --force>out.txt origin main",
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "a leading redirection does not displace the command",
			command: ">/dev/null git push --force origin main",
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "a glued redirection in a cd chain still fires",
			command: "cd scratch && rm -rf>/dev/null *",
			verdict: VerdictBlock,
			entryID: "rm-rf-after-cd-chain",
		},
		{
			name:    "an ordinary fd redirection is not a hazard",
			command: "go test ./... 2>&1",
			verdict: VerdictAllow,
		},
		{
			name:    "an ampersand redirection does not hide the flag",
			command: "git push &>/dev/null --force origin main",
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "a glued ampersand redirection does not hide the flag",
			command: "git push&>/dev/null --force origin main",
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "an ampersand-append redirection does not hide --no-verify",
			command: "git commit &>>log --no-verify -m x",
			verdict: VerdictBlock,
			entryID: "git-commit-no-verify",
		},
		{
			name:    "an ampersand redirection in a cd chain still fires",
			command: "cd scratch && rm &>/dev/null -rf *",
			verdict: VerdictBlock,
			entryID: "rm-rf-after-cd-chain",
		},
		{
			name:    "an ampersand redirection inside a bash payload still fires",
			command: `bash -c 'git push &>/dev/null --force origin main'`,
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "git global value flags do not hide the subcommand",
			command: "git -C /repo push --force origin main",
			verdict: VerdictBlock,
			entryID: "git-push-force",
		},
		{
			name:    "bundled short flags match (-fr satisfies -r and -f)",
			command: "cd scratch && rm -fr ./*",
			verdict: VerdictBlock,
			entryID: "rm-rf-after-cd-chain",
		},
		{
			name:    "a sudo wrapper does not hide the command",
			command: "cd scratch && sudo rm -rf *",
			verdict: VerdictBlock,
			entryID: "rm-rf-after-cd-chain",
		},
		{
			name:    "an environment assignment prefix does not hide the command",
			command: "cd scratch && FOO=bar rm -rf *",
			verdict: VerdictBlock,
			entryID: "rm-rf-after-cd-chain",
		},
		{
			name:    "push -n is dry-run, not --no-verify",
			command: "git push -n origin main",
			verdict: VerdictAllow,
		},
		{
			name:    "commit -n is --no-verify and is blocked",
			command: `git commit -n -m "wip"`,
			verdict: VerdictBlock,
			entryID: "git-commit-no-verify",
		},
		{
			name:    "reset --hard warns rather than blocks",
			command: "git reset --hard origin/main",
			verdict: VerdictWarn,
			entryID: "git-reset-hard",
		},
		{
			name:    "an unrelated command is allowed",
			command: "git status --porcelain",
			verdict: VerdictAllow,
		},
		{
			name:    "a cd chain wrapped over a newline still fires",
			command: "cd scratch &&\nrm -rf *",
			verdict: VerdictBlock,
			entryID: "rm-rf-after-cd-chain",
		},
		{
			name:    "a blank line does not break the cd chain",
			command: "cd scratch &&\n\nrm -rf *",
			verdict: VerdictBlock,
			entryID: "rm-rf-after-cd-chain",
		},
		{
			name:    "an arithmetic shift does not swallow the lines after it",
			command: "echo $((1<<20)) > size.txt\ncd scratch && rm -rf *",
			verdict: VerdictBlock,
			entryID: "rm-rf-after-cd-chain",
		},
		{
			name:    "a hazard inside a shell conditional is still in command position",
			command: "if cd scratch; then rm -rf *; fi",
			verdict: VerdictBlock,
			entryID: "rm-rf-after-cd-chain",
		},
		{
			name:    "a hazard inside a loop body is still in command position",
			command: "for d in a b; do cd $d && rm -rf *; done",
			verdict: VerdictBlock,
			entryID: "rm-rf-after-cd-chain",
		},
		{
			name:    "writing a document that names a hazard is not running one",
			command: "cat > CONTRIBUTING.md <<'EOF'\nNever run git push --force on main.\nEOF",
			verdict: VerdictAllow,
		},
		{
			name:    "a cd on the previous line does not chain into the next line",
			command: "cd scratch\nrm -rf ./build",
			verdict: VerdictAllow,
		},
		{
			name:    "deleting a repository is a human-only operation",
			command: "gh repo delete owner/repo",
			verdict: VerdictBlock,
			entryID: "gh-repo-delete",
		},
		{
			name:    "another two-level gh delete is not repository deletion",
			command: "gh release delete v1.0.0",
			verdict: VerdictAllow,
		},
		{
			name:    "a raw api DELETE on a repository is the same deletion",
			command: "gh api -X DELETE repos/owner/repo",
			verdict: VerdictBlock,
			entryID: "gh-api-repo-delete",
		},
		{
			name:    "the same call written with an attached short value",
			command: "gh api -XDELETE /repos/owner/repo",
			verdict: VerdictBlock,
			entryID: "gh-api-repo-delete",
		},
		{
			// gh passes a fully-qualified URL through to the API unchanged, so
			// this is the same repository deletion with the host spelled out.
			name:    "the same deletion written as a fully-qualified URL",
			command: "gh api -X DELETE https://api.github.com/repos/owner/repo",
			verdict: VerdictBlock,
			entryID: "gh-api-repo-delete",
		},
		{
			// Normalising the URL must not widen the entry: the depth limit is
			// what keeps ordinary work inside a repository allowed.
			name:    "a deeper URL path is still not repository deletion",
			command: "gh api -X DELETE https://api.github.com/repos/owner/repo/git/refs/heads/feature",
			verdict: VerdictAllow,
		},
		{
			name:    "a DELETE deeper under a repository is not repository deletion",
			command: "gh api -X DELETE repos/owner/repo/git/refs/heads/feature",
			verdict: VerdictAllow,
		},
		{
			name:    "a GET on the repository path is not a deletion",
			command: "gh api -X GET repos/owner/repo",
			verdict: VerdictAllow,
		},
		{
			name:    "a refspec with a leading plus is a force push in disguise",
			command: "git push origin +main:main",
			verdict: VerdictBlock,
			entryID: "git-push-force-refspec",
		},
		{
			name:    "an ordinary refspec is not a force push",
			command: "git push origin main:main",
			verdict: VerdictAllow,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := checkOK(t, tc.command)
			if d.Verdict != tc.verdict {
				t.Fatalf("Check(%q).Verdict = %q, want %q (matches: %v)", tc.command, d.Verdict, tc.verdict, d.Matches)
			}
			if d.EntryID != tc.entryID {
				t.Fatalf("Check(%q).EntryID = %q, want %q", tc.command, d.EntryID, tc.entryID)
			}
			if tc.verdict == VerdictAllow {
				return
			}
			if d.Successor == "" || d.Why == "" {
				t.Fatalf("Check(%q): a firing decision must carry a successor and a why, got %+v", tc.command, d)
			}
			if !strings.Contains(d.Message, d.Successor) {
				t.Fatalf("Check(%q).Message = %q, must cite the successor %q", tc.command, d.Message, d.Successor)
			}
		})
	}
}

// TestBlockerWinsOverWarn pins the severity ordering: a line that trips both
// tiers is refused, and both matches are reported.
func TestBlockerWinsOverWarn(t *testing.T) {
	d := checkOK(t, "cd repo && git reset --hard && rm -rf *")
	if d.Verdict != VerdictBlock || d.EntryID != "rm-rf-after-cd-chain" {
		t.Fatalf("blocker must win over warn, got %+v", d)
	}
	if !contains(d.Matches, "rm-rf-after-cd-chain") || !contains(d.Matches, "git-reset-hard") {
		t.Fatalf("Matches = %v, want both the blocker and the warn entry", d.Matches)
	}
}

// TestExecuteStringPayloadIsInspected closes what was the v1 gap (iss-200): a
// hazard carried as a string payload to sh -c is now expanded and matched. The
// payload's cd-chain fires the rm blocker just as the same command would inline.
func TestExecuteStringPayloadIsInspected(t *testing.T) {
	d := checkOK(t, `sh -c 'cd scratch && rm -rf *'`)
	if d.Verdict != VerdictBlock || d.EntryID != "rm-rf-after-cd-chain" {
		t.Fatalf("sh -c payload must be inspected and block; got %+v", d)
	}
}

// TestApiEnterpriseMountIsDocumentedV1Gap records what is left of the gh api
// entry's URL limit now that a `scheme://host/` prefix is stripped. A path
// constraint names a root SEGMENT, and a GitHub Enterprise Server install mounts
// the same API under `/api/v3/`, so the repository path no longer starts the
// path and the entry does not fire. Teaching the generic field about that mount
// would put one host's routing into a declarative shape every entry shares —
// and matching a `repos` root wherever it appeared would falsely refuse
// `DELETE /teams/{id}/repos/{owner}/{repo}`, which removes a repository from a
// team and destroys nothing. It is a stated gap, not a silent one; this test is
// what makes it visible if the behaviour ever changes.
func TestApiEnterpriseMountIsDocumentedV1Gap(t *testing.T) {
	d := checkOK(t, "gh api -X DELETE https://ghe.example.test/api/v3/repos/owner/repo")
	if d.Verdict != VerdictAllow {
		t.Fatalf("v1 reads a path from its root segment; got %+v — update the documented gap before changing this", d)
	}
}

// TestHelpIsNotAnExemption pins a deliberate absence: no entry in this registry
// special-cases `--help`, so a help invocation of a refused command is refused
// too. Fixing it would mean teaching every entry which of its flags mean "do
// nothing", and a guard that reasons about intent is one that can be argued out
// of refusing.
func TestHelpIsNotAnExemption(t *testing.T) {
	d := checkOK(t, "gh repo delete --help")
	if d.Verdict != VerdictBlock {
		t.Fatalf("Check(%q) = %+v, want the block; --help is not an exemption anywhere in this registry", "gh repo delete --help", d)
	}
}

func TestCheckRejectsUnparsableCommand(t *testing.T) {
	if _, err := Defaults().Check(`rm -rf "unterminated`); !errors.Is(err, ErrUnparsableCommand) {
		t.Fatalf("error = %v, want ErrUnparsableCommand", err)
	}
}

func TestDisabledRegistryAllowsEverything(t *testing.T) {
	r := Defaults()
	r.Disabled = true
	d, err := r.Check("cd scratch && rm -rf *")
	if err != nil {
		t.Fatal(err)
	}
	if d.Verdict != VerdictAllow {
		t.Fatalf("a disabled registry must allow everything, got %+v", d)
	}
}

// TestDefaultsAreACopy proves the bundled registry cannot be mutated through a
// caller's handle — the next Check must not inherit another caller's edit.
func TestDefaultsAreACopy(t *testing.T) {
	r := Defaults()
	e := r.Entries["git-reset-hard"]
	e.Tier = TierBlocker
	r.Entries["git-reset-hard"] = e
	delete(r.Entries, "git-push-force")

	fresh := Defaults()
	if fresh.Entries["git-reset-hard"].Tier != TierWarn {
		t.Fatal("mutating a Defaults() handle changed the bundled registry")
	}
	if _, ok := fresh.Entries["git-push-force"]; !ok {
		t.Fatal("deleting from a Defaults() handle removed a bundled entry")
	}
}

// TestDefaultsValidate proves the embedded registry passes the same validation a
// per-repo override must pass.
func TestDefaultsValidate(t *testing.T) {
	if err := Validate(Defaults()); err != nil {
		t.Fatalf("bundled defaults fail validation: %v", err)
	}
	if Defaults().SchemaVersion != SchemaVersion {
		t.Fatalf("bundled schema_version = %d, want %d", Defaults().SchemaVersion, SchemaVersion)
	}
}

// TestValidateBoundsWhatAnEntryTeaches: an entry's why and successor are
// injected into an agent's context word for word by the teaching plane, so
// each is capped at maxLessonFieldBytes. An entry over the bound is refused at
// validation, which refuses the whole file loudly, rather than taught at any
// size: a 240 KB why otherwise pushed every other domain out of the injection
// budget. The bundled registry sits well inside the bound, so the headroom is
// real and not a bound the defaults brush against.
func TestValidateBoundsWhatAnEntryTeaches(t *testing.T) {
	for _, field := range []string{"why", "successor"} {
		r := Defaults()
		e := r.Entries["git-clean"]
		huge := strings.Repeat("w", 240*1000)
		if field == "why" {
			e.Why = huge
		} else {
			e.Successor = huge
		}
		r.Entries["git-clean"] = e
		err := Validate(r)
		if !errors.Is(err, ErrInvalidEntry) {
			t.Fatalf("a 240 KB %s validated: err = %v, want ErrInvalidEntry", field, err)
		}
		if !strings.Contains(err.Error(), field) || !strings.Contains(err.Error(), "git-clean") {
			t.Errorf("the refusal does not name the entry and the field: %v", err)
		}

		e.Why, e.Successor = Defaults().Entries["git-clean"].Why, Defaults().Entries["git-clean"].Successor
		at := strings.Repeat("w", maxLessonFieldBytes)
		if field == "why" {
			e.Why = at
		} else {
			e.Successor = at
		}
		r.Entries["git-clean"] = e
		if err := Validate(r); err != nil {
			t.Errorf("a %s of exactly %d bytes is refused: %v", field, maxLessonFieldBytes, err)
		}
	}
	for id, e := range Defaults().Entries {
		if 3*len(e.Why) > maxLessonFieldBytes || 3*len(e.Successor) > maxLessonFieldBytes {
			t.Errorf("bundled entry %s (why %d bytes, successor %d bytes) leaves under 3x headroom below the %d-byte bound",
				id, len(e.Why), len(e.Successor), maxLessonFieldBytes)
		}
	}
}
