package ahoy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestInstallScaffoldsTheBanlistArtefacts is spc-20 AC5 at the install seam: a repo
// abcd configures inherits the whole two-layer arrangement — the public family in
// the docs-lint config, the guard hook in the repo's committed hooks, and the
// gitignored private stub — rather than a maintainer hand-wiring three files.
func TestInstallScaffoldsTheBanlistArtefacts(t *testing.T) {
	setupHermetic(t)
	// A REAL repository, not the bare `.git` directory the sibling install tests use:
	// the stub write is gated on git's own ignore verdict, and a directory git cannot
	// answer for is a state this test must not be silently exercising.
	repo := gittest.NewRepo(t).Root()
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("install status = %q (remaining=%v), want clean", res.Status, res.Remaining)
	}

	hook := filepath.Join(repo, filepath.FromSlash(GuardHookRelPath))
	fi, err := os.Stat(hook)
	if err != nil {
		t.Fatalf("guard hook not scaffolded: %v", err)
	}
	if fi.Mode().Perm()&0o111 == 0 {
		t.Errorf("guard hook is not executable: mode %v", fi.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(banlist.PublicConfigRelPath))); err != nil {
		t.Errorf("public docs-lint config not scaffolded: %v", err)
	}
	pub, err := banlist.ListPublic(repo)
	if err != nil {
		t.Fatalf("scaffolded public config does not load: %v", err)
	}
	if !pub.Present {
		t.Errorf("public banned-names family absent after scaffolding")
	}
	priv, err := banlist.ListPrivate(repo)
	if err != nil {
		t.Fatalf("scaffolded private stub does not load: %v", err)
	}
	if !priv.Present {
		t.Errorf("private banlist stub absent after scaffolding")
	}
	if !priv.Keyed {
		t.Errorf("private stub does not declare the keyed format, so a later `add` is refused as legacy")
	}
	if len(priv.Entries) != 0 || len(priv.Malformed) != 0 {
		t.Errorf("stub seeded live entries (%v) or unusable lines (%v); its examples must be commented out",
			priv.Entries, priv.Malformed)
	}
}

// TestBanlistStubSeedsOnlyReservedIdentifiers is the AC5 seeding clause: the stub is
// scaffolded into every managed repo, so an example value that looked like a real
// host or address would teach the shape of a leak. Every illustrative identifier
// comes from a reserved documentation range (examples-use-reserved-identifiers), and
// the repo's own network-identifier detector is the judge — one primitive decides
// what "reserved" means, here and in the privacy-hygiene rule.
func TestBanlistStubSeedsOnlyReservedIdentifiers(t *testing.T) {
	// The stub's examples are regular expressions, so their dots are backslash-
	// escaped. Scanning them as written would prove nothing — no detector matches
	// `192\.0\.2\.17` — so the escaping is removed first and the plain identifiers
	// are what gets judged.
	plain := strings.ReplaceAll(privateStubContent(), `\`, "")
	scan := func(text string) []scanner.Finding {
		return scanner.ScanText(text, scanner.Identity{}, scanner.NetworkPatterns(), nil, "private-names.txt")
	}
	// The detector must be armed, or "no findings" is a test that cannot fail: a
	// control value from RFC 1918 has to be flagged before the stub's silence means
	// anything (fix-the-detector).
	control := "\n#   lab-ipv4      10.11.12.13\n" // abcd-audit:allow — the control the detector must flag
	if len(scan(plain+control)) == 0 {
		t.Fatal("the network-identifier detector reported nothing for a control RFC 1918 address; the assertion below is vacuous")
	}
	for _, f := range scan(plain) {
		t.Errorf("stub seeds a non-reserved identifier: kind=%s line=%d", f.Kind, f.Line)
	}
	// The positive half: the seeded examples actually exercise the reserved ranges a
	// user needs to see, so the stub teaches the convention rather than merely
	// avoiding a violation.
	for _, want := range []string{"192.0.2.", "2001:db8:", "example.com", "alice-laptop", "00:00:5e:00:53:"} {
		if !strings.Contains(plain, want) {
			t.Errorf("stub omits the reserved example %q", want)
		}
	}
}

// TestInstallNeverClobbersAHandWrittenGuardHook: a repo's pre-commit hook is the
// maintainer's, and it may carry gates abcd knows nothing about. Scaffolding creates
// the hook it is missing and leaves an existing one exactly as it found it.
//
// A bare `.git` DIRECTORY, not a real repository: this asserts a refusal, and a
// refusal holds in every git state. Anything that asserts a WRITE — the stub in
// particular — must use gittest.NewRepo, because the write is gated on git's own
// ignore verdict and a directory git cannot answer for deliberately fails closed
// (TestInstallScaffoldsTheBanlistArtefacts, TestInstalledStubIsIgnoredByRealGit and
// TestStorePathFailsClosedWhenGitCannotAnswer are the pins for that).
func TestInstallNeverClobbersAHandWrittenGuardHook(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	hook := filepath.Join(repo, filepath.FromSlash(GuardHookRelPath))
	if err := os.MkdirAll(filepath.Dir(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	const handWritten = "#!/usr/bin/env bash\nexit 0\n"
	if err := os.WriteFile(hook, []byte(handWritten), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(hook)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != handWritten {
		t.Errorf("install overwrote a hand-written pre-commit hook")
	}
}

// TestInstallNeverClobbersAnExistingDocsLintConfig: the public config gates CI, so a
// repo that already carries one keeps it byte for byte.
func TestInstallNeverClobbersAnExistingDocsLintConfig(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(repo, filepath.FromSlash(banlist.PublicConfigRelPath))
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	const existing = "{\n  \"roots\": [\"docs\"],\n  \"banned_tokens\": []\n}\n"
	if err := os.WriteFile(cfg, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != existing {
		t.Errorf("install rewrote an existing docs-lint config:\n%s", got)
	}
}

// TestInstallNeverOverwritesAPopulatedPrivateStore is the one that matters most: the
// store holds the patterns whose literal text must not leave the machine, and a
// scaffold that re-seeded it would delete them.
func TestInstallNeverOverwritesAPopulatedPrivateStore(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(repo, filepath.FromSlash(banlist.PrivateRelPath))
	if err := os.MkdirAll(filepath.Dir(store), 0o700); err != nil {
		t.Fatal(err)
	}
	existing := "# abcd-banlist: keyed\nlab-host alice-laptop\\.example\\.com\n"
	if err := os.WriteFile(store, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(store)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != existing {
		t.Errorf("install overwrote a populated private store")
	}
}

// TestScaffoldedGuardHookHoldsItsContract executes the artefact ahoy installs rather
// than reading it: the guard's value is entirely in what it does at commit time, and
// a scaffolded hook that parsed but refused nothing would look identical on disk
// (guards-prove-themselves). It pins the three behaviours the surface promises —
// refuse by key with the pattern withheld, refuse to stage the store itself, and
// warn loudly rather than block when the layer is absent.
func TestScaffoldedGuardHookHoldsItsContract(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	setupHermetic(t)
	repo := t.TempDir()
	env := gittest.Env(t)
	git := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	if out, err := git("init"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if out, err := git("config", "user.name", "Alice Example"); err != nil {
		t.Fatalf("git config: %v\n%s", err, out)
	}
	if out, err := git("config", "user.email", "alice@example.com"); err != nil {
		t.Fatalf("git config: %v\n%s", err, out)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	// git runs .git/hooks/pre-commit; the scaffolded artefact is the committed
	// .githooks/pre-commit, so the test wires the two the way a clone's hooks path
	// does.
	src, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(GuardHookRelPath)))
	if err != nil {
		t.Fatal(err)
	}
	hooksDir := filepath.Join(repo, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), src, 0o755); err != nil {
		t.Fatal(err)
	}

	// The scaffolded stub has no live entry, so the guard warns loudly and lets the
	// commit through: silence must never impersonate protection.
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := git("add", "README.md"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	out, err := git("commit", "-m", "seed")
	if err != nil {
		t.Fatalf("the scaffolded guard blocked a clean commit: %v\n%s", err, out)
	}
	if !strings.Contains(out, "NO ENTRIES") {
		t.Errorf("an entry-less store must warn loudly; got:\n%s", out)
	}

	// One real entry: a match refuses the commit and names the key alone.
	store := filepath.Join(repo, filepath.FromSlash(banlist.PrivateRelPath))
	if err := os.WriteFile(store, []byte("# abcd-banlist: keyed\nlab-host carol-server\\.example\\.net\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "notes.md"), []byte("ssh carol-server.example.net\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := git("add", "notes.md"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	out, err = git("commit", "-m", "leak")
	if err == nil {
		t.Fatalf("the scaffolded guard let a banned name through:\n%s", out)
	}
	if !strings.Contains(out, "lab-host") {
		t.Errorf("the refusal does not name the key:\n%s", out)
	}
	if strings.Contains(out, "carol-server") {
		t.Errorf("the refusal echoes the pattern or the matched text:\n%s", out)
	}

	// Staging the store itself is refused: the whole layer rests on it being
	// untracked, and the guard cannot catch its own source.
	if out, err := git("reset"); err != nil {
		t.Fatalf("git reset: %v\n%s", err, out)
	}
	if out, err := git("add", "-f", filepath.FromSlash(banlist.PrivateRelPath)); err != nil {
		t.Fatalf("git add -f: %v\n%s", err, out)
	}
	if out, err := git("commit", "-m", "stage the store"); err == nil {
		t.Fatalf("the scaffolded guard allowed the private store to be committed:\n%s", out)
	}
}

// --- fence truth (the shared blocker) ---------------------------------------

// TestStubIsNotWrittenWhereGitWouldTrackIt is the blocker both reviewers found.
// Resolvability was decided from a TEXT comparison of the abcd-managed .gitignore
// block, which answers a different question from the one that matters: a repo can
// carry a byte-perfect block and still track the store — a negation after it, a
// tracked local tier, a declined config change that means the block was never
// written at all. Only git can answer "is this path ignored", so only git does.
//
// The shape here is the declined-config-change one: ConfigChange is not approved,
// so stepVisibility never runs, no fence exists, and the stub must NOT be created.
// The artefacts that carry no such hazard still land.
func TestStubIsNotWrittenWhereGitWouldTrackIt(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t).Root()

	opts := installOpts()
	opts.Yes = false
	opts.ValueOverrides = nil
	opts.ApprovedCategories = map[GapCategory]bool{SafeAutocreate: true}
	if _, err := Install(repo, opts, RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(repo, filepath.FromSlash(banlist.PrivateRelPath))
	if _, err := os.Stat(store); err == nil {
		t.Errorf("the stub was written into a repo where git does not ignore it")
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(GuardHookRelPath))); err != nil {
		t.Errorf("the hook carries no such hazard and must still be scaffolded: %v", err)
	}

	// And the gap that remains has to say what would fix it, in git's terms.
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	gap := findGap(det.Gaps, "banlist.private_stub_missing")
	if gap == nil {
		t.Fatal("no private-stub gap after a run that refused to write it")
	}
	if gap.Resolvable {
		t.Errorf("the gap claims apply can close it, but apply refuses while git would track the store")
	}
	if !strings.Contains(gap.FixHint, "ignore") {
		t.Errorf("the fix hint does not name the ignore requirement: %q", gap.FixHint)
	}
}

// TestInstalledStubIsIgnoredByRealGit is the other half: after a full install in a
// real repository, git itself — not a string compare — reports the store as
// ignored, and the health report says so.
func TestInstalledStubIsIgnoredByRealGit(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t)
	if _, err := Install(repo.Root(), installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(banlist.PrivateRelPath))); err != nil {
		t.Fatalf("the stub was not written into a repo whose fence covers it: %v", err)
	}
	if !gitCheckIgnored(t, repo, banlist.PrivateRelPath) {
		t.Errorf("git does not ignore the scaffolded store")
	}
	det, err := Detect(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if !det.Banlist.PrivateStoreIgnored {
		t.Errorf("health reports the store as not ignored when git says it is")
	}
}

// --- containment (security M1) ----------------------------------------------

// TestScaffoldRefusesToWriteThroughASymlinkedAncestor: a repo can commit a symlink
// at .githooks or at the local tier, so a checkout materialises it before abcd ever
// runs. Writing through one lands a 0755 hook and a 0600 stub OUTSIDE the repo
// while every surface reports the in-repo path — the file abcd claims to have
// written is not the file that exists.
func TestScaffoldRefusesToWriteThroughASymlinkedAncestor(t *testing.T) {
	for _, tc := range []struct{ name, dir string }{
		{"hooks dir", ".githooks"},
		{"local tier", ".abcd/.work.local"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupHermetic(t)
			repo := gittest.NewRepo(t).Root()
			outside := t.TempDir()
			link := filepath.Join(repo, filepath.FromSlash(tc.dir))
			if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, link); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
				t.Fatal(err)
			}
			entries, err := os.ReadDir(outside)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entries {
				t.Errorf("scaffolding wrote %q outside the repo through the symlinked %s", e.Name(), tc.dir)
			}
			// The two halves are protected by DIFFERENT mechanisms, and asserting only
			// the shared outcome would let one of them rot silently. The hooks path is
			// held by the containment root alone. The local tier never reaches a write
			// at all: increment 1's contained read refuses the escaping store, which
			// health reports as present-and-unreadable, so no stub gap is raised.
			det, derr := Detect(repo)
			if derr != nil {
				t.Fatal(derr)
			}
			if tc.dir == ".abcd/.work.local" && !det.Banlist.PrivateUnreadable {
				t.Error("a store behind an escaping symlink must read as unreadable, not as absent")
			}
		})
	}
}

// findGap returns the gap with the given id, or nil.
func findGap(gaps []Gap, id string) *Gap {
	for i := range gaps {
		if gaps[i].ID == id {
			return &gaps[i]
		}
	}
	return nil
}

// TestForeignHookIsNeverReportedAsInstalled is correctness M2. A repo's own
// pre-commit hook is legitimate and abcd must not replace it — but presence is not
// identity, and reporting it as the abcd guard is the worst of both states: nothing
// checks the banlist and the status board says something does.
func TestForeignHookIsNeverReportedAsInstalled(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t).Root()
	hook := filepath.Join(repo, filepath.FromSlash(GuardHookRelPath))
	if err := os.MkdirAll(filepath.Dir(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	const handWritten = "#!/usr/bin/env bash\nexit 0\n"
	if err := os.WriteFile(hook, []byte(handWritten), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if det.Banlist.Hook != HookForeign {
		t.Errorf("hook state = %q; want %q", det.Banlist.Hook, HookForeign)
	}
	gap := findGap(det.Gaps, "banlist.hook_foreign")
	if gap == nil {
		t.Fatal("a foreign hook raises no gap at all, so nothing tells the maintainer the banlist is unchecked")
	}
	if gap.Resolvable {
		t.Error("the foreign-hook gap claims apply can close it; abcd must never replace a maintainer's hook")
	}
	if got, err := os.ReadFile(hook); err != nil || string(got) != handWritten {
		t.Errorf("install overwrote a hand-written pre-commit hook")
	}
	// The merge half is NOT abcd's to write here. The shim delegates to the
	// pre-commit hook, so writing it beside a foreign one does two wrong things at
	// once: abcd's marker lands in the shim and the board reports the merge half as
	// installed while merges stay unchecked, and the maintainer's hook silently
	// starts running on merge commits, which git never did before — apply taking
	// over wiring the foreign-hook gap explicitly disclaims owning.
	if det.Banlist.MergeHook == HookInstalled {
		t.Errorf("the merge shim was reported installed while a foreign hook occupies the guard it delegates to")
	}
	shim := filepath.Join(repo, filepath.FromSlash(GuardMergeHookRelPath))
	if _, err := os.Stat(shim); err == nil {
		t.Errorf("install wrote a merge shim that delegates to a hook abcd does not own")
	}
	if findGap(det.Gaps, "banlist.merge_hook_inert") == nil {
		t.Error("nothing reports that merge commits are unchecked")
	}
}

// TestPublicFamilyUnderPublicVisibility is correctness M4. Under
// `visibility: public` the abcd fence ignores the whole `.abcd/` namespace — which
// is where the public family lives — so a config written there reaches no CI run,
// exactly where public exposure is the risk. abcd does not move the file (that is a
// design question a maintainer settles, iss-176). It declines to write a config it
// would immediately have to call unenforceable, and it reports the state either way.
func TestPublicFamilyUnderPublicVisibility(t *testing.T) {
	t.Run("absent config is not offered", func(t *testing.T) {
		setupHermetic(t)
		repo := gittest.NewRepo(t).Root()
		opts := installOpts()
		opts.ValueOverrides["visibility"] = "public"
		if _, err := Install(repo, opts, RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(banlist.PublicConfigRelPath))); err == nil {
			t.Error("install wrote a docs-lint config into a path git ignores, delivering no enforcement")
		}
		det, err := Detect(repo)
		if err != nil {
			t.Fatal(err)
		}
		if !det.Banlist.PublicConfigIgnored {
			t.Fatal("health does not report that git ignores the public config's path")
		}
		gap := findGap(det.Gaps, "banlist.public_family_missing")
		if gap == nil {
			t.Fatal("no gap reports the absent public family")
		}
		if gap.Resolvable {
			t.Error("the gap offers a fix that would produce a config CI never sees")
		}
		if !strings.Contains(gap.FixHint, "iss-176") {
			t.Errorf("the fix hint does not point at the placement question: %q", gap.FixHint)
		}
	})

	t.Run("existing config is reported unenforceable", func(t *testing.T) {
		setupHermetic(t)
		repo := gittest.NewRepo(t).Root()
		cfg := filepath.Join(repo, filepath.FromSlash(banlist.PublicConfigRelPath))
		if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cfg, []byte("{\n  \"banned_tokens\": []\n}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		opts := installOpts()
		opts.ValueOverrides["visibility"] = "public"
		if _, err := Install(repo, opts, RefusingPrompter{}); err != nil {
			t.Fatal(err)
		}
		det, err := Detect(repo)
		if err != nil {
			t.Fatal(err)
		}
		if det.Banlist.PublicFamily != PublicFamilyIgnored {
			t.Fatalf("public family = %q; want %q — git ignores the config under public visibility",
				det.Banlist.PublicFamily, PublicFamilyIgnored)
		}
		if findGap(det.Gaps, "banlist.public_family_ignored") == nil {
			t.Error("no gap reports that the public family never reaches CI")
		}
	})
}

// TestStorePathFailsClosedWhenGitCannotAnswer is minor 1. gitutil.InRepo is false
// for three different worlds: a plain directory, a repo whose git is missing from
// PATH, and a corrupt .git. Only the first is safe to treat as "nothing can be
// committed here"; the other two can commit, and nothing can say whether this path
// would be — so they fail closed rather than borrow the plain directory's answer.
func TestStorePathFailsClosedWhenGitCannotAnswer(t *testing.T) {
	setupHermetic(t)
	plain := t.TempDir()
	if !storePathIsSafe(plain) {
		t.Error("a directory that is not a repository cannot commit anything; the check must be skipped there")
	}
	shaped := t.TempDir()
	if err := os.Mkdir(filepath.Join(shaped, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if storePathIsSafe(shaped) {
		t.Error("a repo-shaped directory git will not answer for must fail closed, not inherit the plain-folder answer")
	}
}

// TestScaffoldPinsTheHooksToLF is minor 8. abcd's own repo gained the attribute on
// this branch; a repo abcd manages needs it just as much, and for a worse reason —
// its maintainer did not choose the hook and will read "bad interpreter" as a
// broken tool rather than as an unprotected repo.
func TestScaffoldPinsTheHooksToLF(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t).Root()
	// A .gitattributes the maintainer already owns: the append must preserve it.
	existing := "*.png binary\n"
	if err := os.WriteFile(filepath.Join(repo, ".gitattributes"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(repo, ".gitattributes"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), guardEOLAttribute) {
		t.Errorf("the committed guard is not pinned to LF:\n%s", got)
	}
	if !strings.HasPrefix(string(got), existing) {
		t.Errorf("the maintainer's own attributes were not preserved:\n%s", got)
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !det.Banlist.HookEOLPinned {
		t.Error("health does not report the pin it just wrote")
	}
	if findGap(det.Gaps, "banlist.hook_eol_unpinned") != nil {
		t.Error("the gap survives the fix that closes it")
	}
	// Idempotent: a second install appends nothing.
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(filepath.Join(repo, ".gitattributes"))
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(got) {
		t.Errorf("a re-install rewrote .gitattributes:\n%s", again)
	}
}

// TestHooksPathArmedResolvesBothSides is minor 5. git accepts an ABSOLUTE
// core.hooksPath, and a literal comparison against ".githooks" reads a perfectly
// armed clone as unarmed — at which point the status line tells the user to replace
// a working absolute path with a relative one, which is advice to downgrade a
// config that already works.
func TestHooksPathArmedResolvesBothSides(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t)
	root := repo.Root()
	set := func(v string) {
		cmd := exec.Command("git", "-C", root, "config", "core.hooksPath", v)
		cmd.Env = repo.Env()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git config: %v\n%s", err, out)
		}
	}
	if hooksPathArmed(root) {
		t.Error("an unset hooks path is not armed")
	}
	set(".githooks")
	if !hooksPathArmed(root) {
		t.Error("a relative hooks path pointing at the committed directory is armed")
	}
	set(filepath.Join(root, ".githooks"))
	if !hooksPathArmed(root) {
		t.Error("an ABSOLUTE hooks path pointing at the same directory is armed just the same")
	}
	set(filepath.Join(root, "other-hooks"))
	if hooksPathArmed(root) {
		t.Error("a hooks path pointing somewhere else is not armed")
	}
	// git expands ~/ itself, so a home-relative spelling of the committed
	// directory is armed too, never read as a path relative to the clone.
	t.Setenv("HOME", filepath.Dir(root))
	set("~/" + filepath.Base(root) + "/.githooks")
	if !hooksPathArmed(root) {
		t.Error("a ~/ hooks path git expands to the committed directory is armed just the same")
	}
}

// TestMarkerIsRecognisedOnlyAsAWholeLine is security MAJ-1. The marker was matched
// as a substring anywhere in the blob, so a foreign hook that merely MENTIONS it —
// a comment, a grep for it, a copied fragment — classified as abcd's own. The board
// then claimed coverage, the merge shim was written beside it, and the foreign hook
// started running on merge commits. Identity is a line, not a substring.
func TestMarkerIsRecognisedOnlyAsAWholeLine(t *testing.T) {
	for name, body := range map[string]HookState{
		"#!/bin/sh\n# see also: # abcd-name-guard: v1 in the abcd docs\nexit 0\n": HookForeign,
		"#!/bin/sh\ngrep -q '# abcd-name-guard:' \"$0\" || exit 1\n":              HookForeign,
		"#!/bin/sh\n# abcd-name-guard: v1\nexit 0\n":                              HookInstalled,
		"#!/bin/sh\n  # abcd-name-guard: v2  \nexit 0\n":                          HookInstalled,
		"#!/bin/sh\n# abcd-name-guard: v1-fork\nexit 0\n":                         HookForeign,
		// A CRLF checkout is the very failure the EOL pin exists for. Such a hook is
		// still OURS — install must be able to heal it — so a trailing CR may not turn
		// it into someone else's file for ever.
		"#!/bin/sh\r\n# abcd-name-guard: v1\r\nexit 0\r\n": HookInstalled,
	} {
		t.Run(shortName(name), func(t *testing.T) {
			setupHermetic(t)
			repo := t.TempDir()
			hook := filepath.Join(repo, filepath.FromSlash(GuardHookRelPath))
			if err := os.MkdirAll(filepath.Dir(hook), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(hook, []byte(name), 0o755); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(repo)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			if got := classifyGuardHook(root, GuardHookRelPath); got != body {
				t.Errorf("classify = %q; want %q", got, body)
			}
		})
	}
}

// TestEOLPinIsGitsAnswerNotAPatternMatch: whether the hooks are pinned to LF is a
// question about what git will DO on checkout, and only git can answer it. A pattern
// match on abcd's own line calls a file pinned when a later `* text eol=crlf`
// overrides it, and misses a live line that differs in whitespace — both of which
// leave a repo one checkout from a guard that silently stops running.
func TestEOLPinIsGitsAnswerNotAPatternMatch(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		want bool
	}{
		"commented out": {"# " + guardEOLAttribute + "\n", false},
		"no space":      {"#" + guardEOLAttribute + "\n", false},
		"live":          {guardEOLAttribute + "\n", true},
		"after another": {"*.png binary\n" + guardEOLAttribute + "\n", true},
		// git reads the LAST matching line, so a later catch-all overrides ours.
		"overridden later":  {guardEOLAttribute + "\n* text eol=crlf\n", false},
		"whitespace and CR": {"  " + guardEOLAttribute + "  \r\n", true},
	} {
		t.Run(name, func(t *testing.T) {
			setupHermetic(t)
			repo := gittest.NewRepo(t).Root()
			if err := os.WriteFile(filepath.Join(repo, gitattributesRelPath), []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(repo)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			if got := gitattributesPinsHookEOL(repo, root); got != tc.want {
				t.Errorf("%q → pinned=%v; want %v", tc.body, got, tc.want)
			}
		})
	}
}

// TestStorePathFailsClosedInASubdirectory is minor 10. A repo-shaped check that only
// looks at cwd/.git answers "not a repository" for every SUBDIRECTORY of a repo, so
// with git unavailable the stub would be written into a tracked tree — the exact
// hazard the check exists to prevent.
func TestStorePathFailsClosedInASubdirectory(t *testing.T) {
	setupHermetic(t)
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "packages", "api")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if storePathIsSafe(sub) {
		t.Error("a subdirectory of a repo-shaped tree git will not answer for must fail closed")
	}
}

// TestScaffoldNeverWidensAnExistingDirectory is minor 8. The mode pin exists so a
// umask cannot strip a bit the guard depends on; applying it to a directory the
// maintainer already created inverts it into abcd loosening permissions nobody asked
// it to touch.
func TestScaffoldNeverWidensAnExistingDirectory(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t).Root()
	hooks := filepath.Join(repo, ".githooks")
	if err := os.Mkdir(hooks, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	// The hook must actually have landed, or the mode below is the mode of a
	// directory nothing was written into and the assertion proves nothing.
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(GuardHookRelPath))); err != nil {
		t.Fatalf("the guard hook was not scaffolded into the pre-existing directory: %v", err)
	}
	fi, err := os.Stat(hooks)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o700 {
		t.Errorf(".githooks mode = %v; want the 0700 the maintainer chose, untouched", fi.Mode().Perm())
	}
}

// shortName trims a fixture body down to a readable subtest name. It is not called
// min: that shadows the builtin, and a helper in a test file is not worth the
// confusion of reading `min` and meaning something else.
func shortName(s string) string {
	if len(s) > 40 {
		return s[:40]
	}
	return s
}

// TestScaffoldedGuardHookResistsEnvironmentSubversion is the regression pin for the
// environment-hardening pass (itd hardening after two adversarial reviews): the
// guard must not be talked into a silent — or a printed-but-unenforced — pass by
// hostile shell state it inherits. Each case injects that state through BASH_ENV,
// which bash sources at non-interactive startup BEFORE the hook body runs, exactly
// as a sourced repo helper or a poisoned rc would; the version-specific BASH_FUNC_*
// env encoding is avoided so the test holds on every bash the hook can run under.
//
// Every case FAILS against the stock v0.6.6 template (verified by reverting it) and
// PASSES against the hardened one.
func TestScaffoldedGuardHookResistsEnvironmentSubversion(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	setupHermetic(t)
	repo := t.TempDir()
	baseEnv := gittest.Env(t)
	git := func(extraEnv []string, args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(append([]string{}, baseEnv...), extraEnv...)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	mustGit := func(extraEnv []string, args ...string) string {
		t.Helper()
		out, err := git(extraEnv, args...)
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return out
	}
	mustGit(nil, "init")
	mustGit(nil, "config", "user.name", "Alice Example")
	mustGit(nil, "config", "user.email", "alice@example.com")
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	// Wire BOTH scaffolded hooks the way a clone's core.hooksPath does, so the merge
	// half's delegation is exercised too.
	hooksDir := filepath.Join(repo, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{GuardHookRelPath, GuardMergeHookRelPath} {
		src, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(hooksDir, filepath.Base(rel)), src, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A live store: key "lab-host", pattern matching a reserved-documentation host.
	store := filepath.Join(repo, filepath.FromSlash(banlist.PrivateRelPath))
	if err := os.WriteFile(store, []byte("# abcd-banlist: keyed\nlab-host carol-server\\.example\\.net\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Seed one clean commit so a later gitlink has a target sha and HEAD exists.
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(nil, "add", "README.md")
	mustGit(nil, "commit", "-m", "seed")
	headSHA := strings.TrimSpace(mustGit(nil, "rev-parse", "HEAD"))

	// bashEnv writes a sourced-at-startup script and returns the env entry that
	// points bash at it. The functions it defines are the hostile shell state.
	bashEnv := func(body string) []string {
		p := filepath.Join(t.TempDir(), "hostile.sh")
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return []string{"BASH_ENV=" + p}
	}
	const banned = "ssh carol-server.example.net\n"

	// Case A — read and echo shadowed. The stock hook's unset list omits them, so the
	// parse loop reads zero entries and every diagnostic goes quiet: a silent pass.
	// The hardened hook pins both, so it still announces its entry count and BLOCKS.
	t.Run("read_echo_shadowed", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte(banned), 0o644); err != nil {
			t.Fatal(err)
		}
		mustGit(nil, "add", "a.txt")
		out, err := git(bashEnv("read() { :; }\necho() { :; }\n"), "commit", "-m", "leakA")
		if err == nil {
			t.Fatalf("guard passed a banned name under a read/echo shadow:\n%s", out)
		}
		if !strings.Contains(out, "1 entry") {
			t.Errorf("guard went quiet under attack (no entry count):\n%s", out)
		}
		if !strings.Contains(out, "lab-host") {
			t.Errorf("refusal did not name the key:\n%s", out)
		}
		if strings.Contains(out, "carol-server") {
			t.Errorf("refusal echoed the pattern or matched text:\n%s", out)
		}
		mustGit(nil, "reset")
	})

	// Case B — declare and exit shadowed. A fake declare neuters the function sweep,
	// so a surviving exit function turns every `exit 1` into a no-op: the stock hook
	// PRINTS a refusal and commits anyway. The hardened hook pins exit on the fixed,
	// expansion-free list that runs before declare is ever consulted, so exit still
	// exits and the commit does not land.
	t.Run("declare_exit_shadowed", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(repo, "b.txt"), []byte(banned), 0o644); err != nil {
			t.Fatal(err)
		}
		mustGit(nil, "add", "b.txt")
		out, err := git(bashEnv("declare() { return 0; }\nexit() { return 0; }\n"), "commit", "-m", "leakB")
		if err == nil {
			t.Fatalf("guard let a banned name through under a declare/exit shadow:\n%s", out)
		}
		if tree := mustGit(nil, "ls-tree", "-r", "--name-only", "HEAD"); strings.Contains(tree, "b.txt") {
			t.Fatalf("the banned commit landed in history despite the printed refusal:\n%s", tree)
		}
		mustGit(nil, "reset")
	})

	// Case C — a gitlink whose PATH is a banned name. The stock hook appended the
	// staged path AFTER the gitlink `continue`, so a submodule path skipped the scan
	// entirely. The hardened hook appends it before, so the path is scanned like
	// content and refuses by key.
	t.Run("gitlink_path", func(t *testing.T) {
		if out, err := git(nil, "update-index", "--add", "--cacheinfo",
			"160000,"+headSHA+",carol-server.example.net"); err != nil {
			t.Fatalf("stage gitlink: %v\n%s", err, out)
		}
		out, err := git(nil, "commit", "-m", "leakC")
		if err == nil {
			t.Fatalf("guard passed a gitlink named with a banned path:\n%s", out)
		}
		if !strings.Contains(out, "lab-host") {
			t.Errorf("gitlink refusal did not name the key:\n%s", out)
		}
		mustGit(nil, "read-tree", "HEAD")
	})

	// Case D — a refused path carrying control bytes. The stock hook echoed the raw
	// staged path, so an embedded escape could redraw or forge the refusal text. The
	// hardened hook scrubs control bytes before printing, so no raw ESC reaches the
	// output while the refusal still stands.
	t.Run("control_bytes_scrubbed", func(t *testing.T) {
		tierDir := filepath.Join(repo, ".abcd", ".work.local")
		if err := os.MkdirAll(tierDir, 0o755); err != nil {
			t.Fatal(err)
		}
		evil := ".abcd/.work.local/\x1b[2Kevil.txt"
		if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(evil)), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := git(nil, "add", "-f", filepath.FromSlash(evil))
		if err != nil {
			t.Fatalf("git add tier path: %v\n%s", err, out)
		}
		out, err = git(nil, "commit", "-m", "leakD")
		if err == nil {
			t.Fatalf("guard allowed a file inside the gitignored local tier to be committed:\n%s", out)
		}
		if strings.ContainsRune(out, '\x1b') {
			t.Errorf("a raw ESC byte from the staged path reached the terminal output:\n%q", out)
		}
		mustGit(nil, "reset")
	})
}

// TestScaffoldedGuardHookInheritsThePrimaryStoreInAWorktree pins itd-150 on the
// SCAFFOLDED template rather than on this repo's own dogfood copy. The private
// store lives in the gitignored per-worktree local tier, so every commit made from
// a `git worktree add` checkout ran with the banlist absent — the isolated-agent
// pattern systematically bypassing a protection the main checkout has (iss-370).
// The guard resolves the primary checkout's store through the common git dir and
// enforces it there, naming the key alone as ever.
func TestScaffoldedGuardHookInheritsThePrimaryStoreInAWorktree(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	setupHermetic(t)
	repo := t.TempDir()
	env := gittest.Env(t)
	gitIn := func(dir string, args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	git := func(args ...string) (string, error) { return gitIn(repo, args...) }
	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "Alice Example"},
		{"config", "user.email", "alice@example.com"},
	} {
		if out, err := git(args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	// git runs the hook the COMMON git dir carries, which a linked worktree shares
	// with its primary — the same wiring `core.hooksPath .githooks` gives a clone.
	src, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(GuardHookRelPath)))
	if err != nil {
		t.Fatal(err)
	}
	hooksDir := filepath.Join(repo, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), src, 0o755); err != nil {
		t.Fatal(err)
	}
	// One real entry in the PRIMARY checkout's store, and nothing in the worktree's.
	store := filepath.Join(repo, filepath.FromSlash(banlist.PrivateRelPath))
	if err := os.WriteFile(store, []byte("# abcd-banlist: keyed\nlab-host carol-server\\.example\\.net\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := git("add", "-A"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	if out, err := git("-c", "core.hooksPath=/dev/null", "commit", "-m", "seed"); err != nil {
		t.Fatalf("seed commit: %v\n%s", err, out)
	}

	linked := filepath.Join(t.TempDir(), "linked")
	if out, err := git("worktree", "add", "-b", "linked", linked); err != nil {
		t.Skipf("git worktree add unavailable: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(linked, ".abcd", ".work.local", "private-names.txt")); err == nil {
		t.Fatal("the linked worktree already carries a store; the fixture proves nothing")
	}
	if err := os.WriteFile(filepath.Join(linked, "notes.md"), []byte("ssh carol-server.example.net\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := gitIn(linked, "add", "notes.md"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	out, err := gitIn(linked, "commit", "-m", "leak")
	if err == nil {
		t.Fatalf("a linked worktree committed a name the primary checkout's store bans:\n%s", out)
	}
	if !strings.Contains(out, "lab-host") {
		t.Errorf("the refusal does not name the key:\n%s", out)
	}
	if strings.Contains(out, "carol-server") {
		t.Errorf("the refusal echoes the pattern or the matched text:\n%s", out)
	}
	if !strings.Contains(out, "primary checkout") {
		t.Errorf("the refusal does not say the entry was inherited from the primary checkout:\n%s", out)
	}
}

// TestScaffoldedGuardHookAnnouncesOncePerCommit is iss-2609181122202952 on the
// SCAFFOLDED template, which is the copy the report came from: a managed repo's
// commit from a linked worktree printed three name-guard notice lines (the
// inheritance, then the format and the count of the one store it read), which reads
// as the hook running three times. One commit, one notice line.
func TestScaffoldedGuardHookAnnouncesOncePerCommit(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	setupHermetic(t)
	repo := t.TempDir()
	env := gittest.Env(t)
	gitIn := func(dir string, args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "Alice Example"},
		{"config", "user.email", "alice@example.com"},
	} {
		if out, err := gitIn(repo, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(GuardHookRelPath)))
	if err != nil {
		t.Fatal(err)
	}
	hooksDir := filepath.Join(repo, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), src, 0o755); err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(repo, filepath.FromSlash(banlist.PrivateRelPath))
	if err := os.WriteFile(store, []byte("# abcd-banlist: keyed\nlab-host carol-server\\.example\\.net\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := gitIn(repo, "add", "-A"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	if out, err := gitIn(repo, "-c", "core.hooksPath=/dev/null", "commit", "-m", "seed"); err != nil {
		t.Fatalf("seed commit: %v\n%s", err, out)
	}
	linked := filepath.Join(t.TempDir(), "linked")
	if out, err := gitIn(repo, "worktree", "add", "-b", "linked", linked); err != nil {
		t.Skipf("git worktree add unavailable: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(linked, "notes.md"), []byte("nothing sensitive here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := gitIn(linked, "add", "notes.md"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	out, err := gitIn(linked, "commit", "-m", "clean")
	if err != nil {
		t.Fatalf("clean content was refused:\n%s", out)
	}
	var notices []string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "abcd name-guard:") {
			notices = append(notices, l)
		}
	}
	if len(notices) != 1 {
		t.Fatalf("a clean commit from a linked worktree printed %d name-guard notice lines, want 1:\n%s", len(notices), out)
	}
	if !strings.Contains(notices[0], "primary checkout") || !strings.Contains(notices[0], "keyed store") {
		t.Errorf("the one notice does not say which store was inherited and in what format:\n%s", notices[0])
	}
}

// TestScaffoldedGuardHookRefusesAMirrorInsideAnotherCheckout is the worktree
// resolution's confinement, pinned on the SCAFFOLDED template rather than on this
// repo's dogfood copy — the two halves must stay in lockstep, and a fix applied to
// one alone leaves every repo abcd configures carrying the hole.
//
// The layout: a VICTIM checkout that is a real working tree with a private store of
// its own, and a linked worktree of an UNRELATED repository whose git dir was
// placed inside that checkout (by a bare clone, or by --separate-git-dir). The
// common dir's parent is then the victim, and the victim carries a `.git` — so a
// guard that resolves its primary checkout by path arithmetic plus a `.git`
// existence test enforces a stranger's private list on this repository's commits.
func TestScaffoldedGuardHookRefusesAMirrorInsideAnotherCheckout(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	for _, shape := range []string{"bare", "separate-git-dir"} {
		t.Run(shape, func(t *testing.T) {
			setupHermetic(t)
			env := gittest.Env(t)
			git := func(dir string, args ...string) (string, error) {
				cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
				cmd.Env = env
				out, err := cmd.CombinedOutput()
				return string(out), err
			}
			seed := func(dir string) {
				t.Helper()
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				for _, args := range [][]string{
					{"init"},
					{"config", "user.name", "Alice Example"},
					{"config", "user.email", "alice@example.com"},
					{"-c", "core.hooksPath=/dev/null", "commit", "--allow-empty", "-m", "seed"},
				} {
					if out, err := git(dir, args...); err != nil {
						t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
					}
				}
			}

			root := t.TempDir()
			victim := filepath.Join(root, "victim")
			seed(victim)
			if err := os.MkdirAll(filepath.Join(victim, filepath.FromSlash(banlist.PrivateDirRelPath)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(victim, filepath.FromSlash(banlist.PrivateRelPath)),
				[]byte("# abcd-banlist: keyed\nlab-host carol-server\\.example\\.net\n"), 0o600); err != nil {
				t.Fatal(err)
			}

			var commonDir string
			switch shape {
			case "bare":
				src := filepath.Join(root, "src")
				seed(src)
				commonDir = filepath.Join(victim, "mirror.git")
				if out, err := git(root, "clone", "--bare", src, commonDir); err != nil {
					t.Skipf("bare clone unavailable: %v\n%s", err, out)
				}
			default:
				work := filepath.Join(root, "work")
				if err := os.MkdirAll(work, 0o755); err != nil {
					t.Fatal(err)
				}
				commonDir = filepath.Join(victim, "gitdir")
				if out, err := git(work, "init", "--separate-git-dir="+commonDir); err != nil {
					t.Skipf("git init --separate-git-dir unavailable: %v\n%s", err, out)
				}
				for _, args := range [][]string{
					{"config", "user.name", "Alice Example"},
					{"config", "user.email", "alice@example.com"},
					{"-c", "core.hooksPath=/dev/null", "commit", "--allow-empty", "-m", "seed"},
				} {
					if out, err := git(work, args...); err != nil {
						t.Fatalf("git %v: %v\n%s", args, err, out)
					}
				}
			}

			linked := filepath.Join(t.TempDir(), "linked")
			if out, err := git(commonDir, "worktree", "add", linked, "HEAD"); err != nil {
				t.Skipf("git worktree add unavailable: %v\n%s", err, out)
			}
			// The SCAFFOLDED template is what is under test, installed where a linked
			// worktree resolves its hooks: the common git dir.
			hooksDir := filepath.Join(commonDir, "hooks")
			if err := os.MkdirAll(hooksDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), guardHookTemplate, 0o755); err != nil {
				t.Fatal(err)
			}

			if out, err := git(linked, "config", "user.name", "Alice Example"); err != nil {
				t.Fatalf("git config: %v\n%s", err, out)
			}
			if out, err := git(linked, "config", "user.email", "alice@example.com"); err != nil {
				t.Fatalf("git config: %v\n%s", err, out)
			}
			if err := os.WriteFile(filepath.Join(linked, "notes.md"), []byte("ssh carol-server.example.net\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if out, err := git(linked, "add", "notes.md"); err != nil {
				t.Fatalf("git add: %v\n%s", err, out)
			}
			out, err := git(linked, "commit", "-m", "t")
			if err != nil {
				t.Fatalf("the scaffolded guard enforced the private store of a checkout that merely holds this repository's git dir:\n%s", out)
			}
			if strings.Contains(out, "lab-host") {
				t.Errorf("the hook disclosed a key from an unrelated repository's private store:\n%s", out)
			}
			if strings.Contains(out, "primary checkout") {
				t.Errorf("the guard claimed to inherit a store from a directory that is not this repository's working tree:\n%s", out)
			}
		})
	}
}
