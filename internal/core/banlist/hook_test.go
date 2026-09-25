package banlist

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// locateHook finds the committed .githooks/pre-commit by walking up from the
// test's working directory. Skips when not run from a checkout (e.g. a build
// tarball) or when bash/git are unavailable.
func locateHook(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	topLevel := exec.Command("git", "rev-parse", "--show-toplevel")
	topLevel.Env = gittest.Env(t)
	out, err := topLevel.Output()
	if err != nil {
		t.Skip("not in a git checkout")
	}
	hook := filepath.Join(strings.TrimSpace(string(out)), ".githooks", "pre-commit")
	if _, err := os.Stat(hook); err != nil {
		t.Skipf("hook not found: %v", err)
	}
	return hook
}

// hookRepo is a throwaway repo with the committed hook installed, so a test can
// stage whatever shape it needs (a rename, a binary blob, a huge file, a
// .gitattributes that suppresses the textual diff) and then attempt a real commit.
type hookRepo struct {
	t    *testing.T
	dir  string
	env  []string
	name string
}

// newHookRepo initialises the repo, installs the committed hook, and writes the
// private banlist when body != "" (body == "" leaves it absent: the fresh-clone
// case).
func newHookRepo(t *testing.T, body string) *hookRepo {
	t.Helper()
	hook := locateHook(t)
	r := &hookRepo{t: t, dir: t.TempDir(), env: gittest.Env(t)}

	r.git("init")
	r.git("config", "user.name", "Alice Example")
	r.git("config", "user.email", "alice@example.com")

	if body != "" {
		r.writeBanlist(body)
	}

	hooksDir := filepath.Join(r.dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(hook)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), src, 0o755); err != nil {
		t.Fatal(err)
	}
	return r
}

// writeBanlist installs the private store's bytes, and gitignores the local tier
// exactly as a real abcd-managed repo does — without that, a test's `git add -A`
// stages the banlist itself and its patterns appear in the staged diff, which would
// let a shape test pass for the wrong reason.
func (r *hookRepo) writeBanlist(body string) {
	r.t.Helper()
	local := filepath.Join(r.dir, ".abcd", ".work.local")
	if err := os.MkdirAll(local, 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(local, "private-names.txt"), []byte(body), 0o600); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.dir, ".gitignore"), []byte(".abcd/.work.local/\n"), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

// git runs a git command that must succeed.
func (r *hookRepo) git(args ...string) string {
	r.t.Helper()
	out, err := r.tryGit(args...)
	if err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return out
}

// tryGit runs a git command and returns its combined output and error.
func (r *hookRepo) tryGit(args ...string) (string, error) {
	r.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", r.dir}, args...)...)
	cmd.Env = r.env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// write puts content at a repo-relative path, creating parents.
func (r *hookRepo) write(rel, content string) {
	r.t.Helper()
	p := filepath.Join(r.dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

// commit attempts a commit and reports whether the hook refused it, plus every
// byte the hook wrote.
func (r *hookRepo) commit() (blocked bool, output string) {
	r.t.Helper()
	out, err := r.tryGit("commit", "-m", "t")
	return err != nil, out
}

// hookRun is the common shape: stage `staged` as one file's content and attempt a
// commit against the given banlist body.
func hookRun(t *testing.T, banlist, staged string) (blocked bool, output string) {
	t.Helper()
	r := newHookRepo(t, banlist)
	r.write("note.md", staged)
	r.git("add", "note.md")
	return r.commit()
}

// corpus reads one of the shared fixture banlists — the files the Go parser reads
// too, so both readers are driven by identical bytes.
func corpus(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestPreCommitHook_AbsentBanlistWarnsLoudly pins AC4: a machine with no private
// banlist is UNPROTECTED, and the hook says so out loud. A silent pass looks
// exactly like a clean check, which is the failure mode this test exists for.
func TestPreCommitHook_AbsentBanlistWarnsLoudly(t *testing.T) {
	blocked, out := hookRun(t, "", "widgetworks ships today\n")
	if blocked {
		t.Fatalf("commit blocked with no banlist present; want it to proceed\n%s", out)
	}
	for _, want := range []string{"WARNING", "INACTIVE", "private-names.txt"} {
		if !strings.Contains(out, want) {
			t.Errorf("hook output does not mention %q; the inactive layer must announce itself\n%s", want, out)
		}
	}
}

// TestPreCommitHook_EntrylessStoreWarnsLoudly is the other half of AC4, and the
// one an emptied store hits: a store that exists but yields no entries checks
// exactly as much as an absent one, so it must be exactly as loud. A refresh that
// truncates the store must not convert the warning into silence.
func TestPreCommitHook_EntrylessStoreWarnsLoudly(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"empty file", "\n"},
		{"comments only", "# abcd-banlist: keyed\n# nothing yet\n\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			blocked, out := hookRun(t, tc.body, "widgetworks ships today\n")
			if blocked {
				t.Fatalf("commit blocked by an entryless store\n%s", out)
			}
			for _, want := range []string{"WARNING", "NO ENTRIES"} {
				if !strings.Contains(out, want) {
					t.Errorf("hook output does not mention %q; an entryless store checks nothing and must say so\n%s", want, out)
				}
			}
		})
	}
}

// TestPreCommitHook_RefusesByKeyOnly pins AC2: the refusal names the entry key
// and nothing else — not the matched string, not the pattern value.
func TestPreCommitHook_RefusesByKeyOnly(t *testing.T) {
	const banlist = "# abcd-banlist: keyed\nwidget-partner   widgetworks\n"
	blocked, out := hookRun(t, banlist, "the widgetworks deal closes friday\n")
	if !blocked {
		t.Fatalf("commit not blocked by a matching banlist entry\n%s", out)
	}
	if !strings.Contains(out, "widget-partner") {
		t.Errorf("refusal does not name the entry key\n%s", out)
	}
	for _, leak := range []string{"widgetworks", "WIDGETWORKS", "friday"} {
		if strings.Contains(strings.ToLower(out), strings.ToLower(leak)) {
			t.Errorf("output leaks %q; neither the pattern nor the matched line may be echoed\n%s", leak, out)
		}
	}
}

// TestPreCommitHook_KeyedCorpus pins AC3 on the keyed corpus: hostnames, IPv4/IPv6
// addresses, CIDR prefixes, and MAC addresses are matched exactly as name entries
// are, and a tab separates fields exactly as spaces do. Every value is reserved
// for documentation (RFC 5737/3849/2606/7042) or derived from the persona registry.
func TestPreCommitHook_KeyedCorpus(t *testing.T) {
	body := corpus(t, "parse-corpus.txt")
	for _, tc := range corpusMustBlock {
		t.Run(tc.name, func(t *testing.T) {
			blocked, out := hookRun(t, body, tc.text)
			if !blocked {
				t.Fatalf("commit not blocked; want a refusal naming %q\n%s", tc.key, out)
			}
			if !strings.Contains(out, tc.key) {
				t.Errorf("refusal does not name key %q\n%s", tc.key, out)
			}
			if strings.Contains(out, strings.TrimSpace(tc.text)) {
				t.Errorf("output echoes the matched line\n%s", out)
			}
		})
	}
}

// TestPreCommitHook_LegacyStoreReadsWholeLines pins the compatibility rule at its
// exact strength: a store with no format declaration is read one WHOLE-LINE
// pattern per line, so it keeps matching precisely what it always matched, and no
// part of a line is ever read — or printed — as a key.
func TestPreCommitHook_LegacyStoreReadsWholeLines(t *testing.T) {
	body := corpus(t, "parse-corpus-legacy.txt")
	for _, tc := range legacyMustBlock {
		t.Run(tc.name, func(t *testing.T) {
			blocked, out := hookRun(t, body, tc.text)
			if !blocked {
				t.Fatalf("legacy line did not block; want a refusal naming %q\n%s", tc.key, out)
			}
			if !strings.Contains(out, tc.key) {
				t.Errorf("refusal does not name the synthetic key %q\n%s", tc.key, out)
			}
			for _, leak := range append([]string{"partnerco"}, legacyFirstFields...) {
				if strings.Contains(out, leak) {
					t.Errorf("output leaks %q: on a legacy line the first field is PART OF THE PATTERN, never a key\n%s", leak, out)
				}
			}
		})
	}
}

// TestPreCommitHook_LegacyStoreDoesNotSplitKeys is the must-pass half of the same
// rule and the detector for the key-splitting leak: if the hook split a legacy line
// on whitespace, the remainder-only patterns below would start matching and the
// first field would be printed as a key.
func TestPreCommitHook_LegacyStoreDoesNotSplitKeys(t *testing.T) {
	body := corpus(t, "parse-corpus-legacy.txt")
	for _, tc := range legacyMustPass {
		t.Run(tc.name, func(t *testing.T) {
			blocked, out := hookRun(t, body, tc.text)
			if blocked {
				t.Fatalf("commit refused for content matching no whole-line pattern\n%s", out)
			}
		})
	}
}

// TestPreCommitHook_PermittedCorpusPasses is the must-pass half of the keyed
// guard's bidirectional proof (guards-prove-themselves): content that matches no
// entry commits cleanly, so the guard is not simply refusing everything. It also
// pins that comment and blank lines — including the format declaration itself —
// are skipped rather than read as patterns.
func TestPreCommitHook_PermittedCorpusPasses(t *testing.T) {
	body := corpus(t, "parse-corpus.txt")
	for _, tc := range corpusMustPass {
		t.Run(tc.name, func(t *testing.T) {
			blocked, out := hookRun(t, body, tc.text)
			if blocked {
				t.Fatalf("commit refused for content matching no entry\n%s", out)
			}
		})
	}
}

// TestPreCommitHook_UnusableLinesFailSafe pins the malformed-entry contract on the
// shared corpus: every unusable line class refuses the commit by LINE NUMBER, none
// is silently skipped, and no line's content is echoed. A keyed store's unparseable
// line is the important case — its first field may be the secret, so it has no key
// and the hook must not invent one out of the line's bytes.
func TestPreCommitHook_UnusableLinesFailSafe(t *testing.T) {
	blocked, out := hookRun(t, corpus(t, "parse-corpus-malformed.txt"), "nothing sensitive here\n")
	if !blocked {
		t.Fatalf("unusable banlist lines did not fail safe\n%s", out)
	}
	for _, line := range malformedUnusableLines {
		want := "line " + itoa(line)
		if !strings.Contains(out, want) {
			t.Errorf("refusal does not name %q; every unusable line must be reported\n%s", want, out)
		}
	}
	for _, leak := range []string{"unclosed", "partnerco", "nbsp-key", "vt-key"} {
		if strings.Contains(out, leak) {
			t.Errorf("output echoes %q from an unusable line; the content is withheld by design\n%s", leak, out)
		}
	}
}

// itoa keeps the assertions above free of a strconv import in a file that is
// otherwise all shell-driving.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// keyedBanlist is the one-entry store the staged-shape tests below share.
const keyedBanlist = "# abcd-banlist: keyed\nwidget-partner   widgetworks\n"

// TestPreCommitHook_StagedShapesThatDefeatDiffText is the reason the guard reads
// staged BLOBS rather than diff text. Each shape below stages the banned name and
// produces a textual diff in which it does not appear as an added line — so a guard
// that greps `git diff --cached` output passes the commit while the name goes into
// history. A guard that reads `git show :<path>` sees the content itself and cannot
// be shaped around.
func TestPreCommitHook_StagedShapesThatDefeatDiffText(t *testing.T) {
	t.Run("content line beginning ++", func(t *testing.T) {
		// In a unified diff this line is emitted as "+++widgetworks", which every
		// header filter drops along with the real "+++ b/path" header.
		r := newHookRepo(t, keyedBanlist)
		r.write("note.md", "++widgetworks\n")
		r.git("add", "note.md")
		blocked, out := r.commit()
		if !blocked {
			t.Fatalf("commit not blocked; the name is in the staged content\n%s", out)
		}
	})

	t.Run("binary blob", func(t *testing.T) {
		// A NUL makes git call the file binary: the textual diff carries no + lines
		// at all, so there is nothing for a diff-text guard to match.
		r := newHookRepo(t, keyedBanlist)
		r.write("blob.bin", "\x00\x01\x02widgetworks\x00trailer\n")
		r.git("add", "blob.bin")
		blocked, out := r.commit()
		if !blocked {
			t.Fatalf("commit not blocked; the name is in a staged binary blob\n%s", out)
		}
	})

	t.Run("gitattributes suppressing the diff", func(t *testing.T) {
		// A committed `* -diff` suppresses the textual diff repo-wide, which turns a
		// diff-text guard off for every file in the repo at once.
		r := newHookRepo(t, keyedBanlist)
		r.write(".gitattributes", "* -diff\n")
		r.write("seed.md", "nothing sensitive here\n")
		r.git("add", ".gitattributes", "seed.md")
		if blocked, out := r.commit(); blocked {
			t.Fatalf("seed commit refused\n%s", out)
		}
		r.write("note.md", "widgetworks ships today\n")
		r.git("add", "note.md")
		blocked, out := r.commit()
		if !blocked {
			t.Fatalf("commit not blocked; `* -diff` must not switch the guard off\n%s", out)
		}
	})

	t.Run("rename plus modification", func(t *testing.T) {
		// Status R, which --diff-filter=ACM excludes outright.
		r := newHookRepo(t, keyedBanlist)
		r.write("old.md", strings.Repeat("filler line for rename detection\n", 20))
		r.git("add", "old.md")
		if blocked, out := r.commit(); blocked {
			t.Fatalf("seed commit refused\n%s", out)
		}
		r.git("mv", "old.md", "new.md")
		r.write("new.md", strings.Repeat("filler line for rename detection\n", 20)+"widgetworks\n")
		r.git("add", "-A")
		blocked, out := r.commit()
		if !blocked {
			t.Fatalf("commit not blocked; a renamed-and-modified file is staged content\n%s", out)
		}
	})
}

// TestPreCommitHook_StageExplicitBlobRead pins the fix for the `git show` rev-magic
// bypass. A file literally named `0:README.md` fed to the ambiguous `git show
// ":$path"` is parsed as "stage 0 of README.md" — so a hostile blob under that name
// is never scanned, and `2:notes.md` mis-resolves the same way. The stage-explicit
// `git show ":0:$path"` names stage 0 and the path unambiguously.
func TestPreCommitHook_StageExplicitBlobRead(t *testing.T) {
	for _, name := range []string{"0:README.md", "2:notes.md"} {
		t.Run(name, func(t *testing.T) {
			r := newHookRepo(t, keyedBanlist)
			// A decoy at the path the rev-magic would resolve to, holding nothing
			// banned: the OLD hook scanned THIS instead of the hostile file.
			r.write("README.md", "a clean readme\n")
			r.write("notes.md", "clean notes\n")
			r.write(name, "the widgetworks deal closes friday\n")
			r.git("add", "README.md", "notes.md", name)
			blocked, out := r.commit()
			if !blocked {
				t.Fatalf("commit not blocked; a file named %q hid its banned content behind git rev-magic\n%s", name, out)
			}
		})
	}
}

// TestPreCommitHook_RefusesStagingThePrivateStore pins that the guard refuses to
// commit the private store (or anything in the local-ephemeral tier). The whole
// layer rests on that file being untracked; a `git add -f` of it would otherwise
// commit the plaintext banlist, and the guard cannot catch its own source.
func TestPreCommitHook_RefusesStagingThePrivateStore(t *testing.T) {
	r := newHookRepo(t, keyedBanlist)
	// Force past the gitignore, exactly the leak: stage the store itself.
	r.git("add", "-f", ".abcd/.work.local/private-names.txt")
	blocked, out := r.commit()
	if !blocked {
		t.Fatalf("commit not blocked; the private banlist was committed\n%s", out)
	}
	if !strings.Contains(out, "private-names.txt") || !strings.Contains(out, "never be committed") {
		t.Errorf("refusal does not name the store path and the invariant\n%s", out)
	}
}

// TestPreCommitHook_BannedNameInAFilenameBlocks pins that a banned name in a staged
// PATH is refused by key exactly as one in staged content is: a filename enters
// history just as surely as a file's bytes.
func TestPreCommitHook_BannedNameInAFilenameBlocks(t *testing.T) {
	r := newHookRepo(t, keyedBanlist)
	r.write("widgetworks-notes.md", "nothing sensitive in here\n")
	r.git("add", "widgetworks-notes.md")
	blocked, out := r.commit()
	if !blocked {
		t.Fatalf("commit not blocked; the banned name is in the staged FILENAME\n%s", out)
	}
	if !strings.Contains(out, "widget-partner") {
		t.Errorf("refusal does not name the key\n%s", out)
	}
}

// TestPreCommitHook_SkipsGitlinks pins that a staged submodule/gitlink (mode 160000)
// does not fail-close the guard. A gitlink has no blob in this object store, so
// `git show :0:<path>` cannot succeed; the guard must skip it, not refuse every
// commit forever. The gitlink is staged directly via update-index (a real submodule
// needs a second repo), which is the same index shape a submodule add produces.
func TestPreCommitHook_SkipsGitlinks(t *testing.T) {
	r := newHookRepo(t, keyedBanlist)
	r.write("seed.md", "nothing sensitive here\n")
	r.git("add", "seed.md")
	if blocked, out := r.commit(); blocked {
		t.Fatalf("seed commit refused\n%s", out)
	}
	// A commit sha NOT in this object store — the real submodule shape, where the
	// superproject's index names a commit that lives only in the submodule. `git show
	// :0:subm` then exits 128, which the fail-closed branch would turn into a refusal
	// of every commit forever; the guard must skip the gitlink by its mode instead.
	const fakeSHA = "0000000000000000000000000000000000000001"
	if _, err := r.tryGit("update-index", "--add", "--cacheinfo", "160000,"+fakeSHA+",subm"); err != nil {
		t.Skip("cannot stage a gitlink in this sandbox")
	}
	blocked, out := r.commit()
	if blocked {
		t.Fatalf("commit blocked by a staged gitlink; the guard fail-closed on a mode it cannot read\n%s", out)
	}
}

// TestPreCommitHook_AnnouncesTheFormatItRead pins the visible-downgrade defence:
// the guard prints one line naming the format and entry count it actually read,
// BEFORE the scan. Stripping the `# abcd-banlist: keyed` declaration silently turns
// every keyed entry into a non-matching whole-line pattern, and the announcement is
// what makes that downgrade visible at commit time.
func TestPreCommitHook_AnnouncesTheFormatItRead(t *testing.T) {
	keyedOut := func() string {
		r := newHookRepo(t, keyedBanlist)
		r.write("note.md", "nothing sensitive here\n")
		r.git("add", "note.md")
		_, out := r.commit()
		return out
	}()
	if !strings.Contains(keyedOut, "keyed store") {
		t.Errorf("keyed store not announced\n%s", keyedOut)
	}

	// Strip line 1: the same entry line is now a whole-line pattern (legacy).
	downgraded := strings.TrimPrefix(keyedBanlist, "# abcd-banlist: keyed\n")
	r := newHookRepo(t, downgraded)
	r.write("note.md", "nothing sensitive here\n")
	r.git("add", "note.md")
	_, out := r.commit()
	if !strings.Contains(out, "legacy store") {
		t.Errorf("a stripped declaration was not announced as a legacy store\n%s", out)
	}
	if strings.Contains(out, "keyed store") {
		t.Errorf("the downgraded store was still announced as keyed\n%s", out)
	}
}

// TestPreCommitHook_LargeStagedFileStillBlocks is the regression pin for the
// SIGPIPE class the increment already fixed and never tested: a staged file far
// larger than a pipe buffer, with the name on its FIRST line, so a matching grep
// exits long before the writer finishes. Deleting the temp-file machinery would
// reintroduce a fail-open pass with a green suite.
func TestPreCommitHook_LargeStagedFileStillBlocks(t *testing.T) {
	r := newHookRepo(t, keyedBanlist)
	var b strings.Builder
	b.WriteString("widgetworks on the very first line\n")
	for i := 0; i < 200000; i++ {
		b.WriteString("filler line that matches no banlist entry at all\n")
	}
	r.write("big.md", b.String())
	r.git("add", "big.md")
	blocked, out := r.commit()
	if !blocked {
		t.Fatalf("commit not blocked; a large staged file must not defeat the guard\n%s", out)
	}
}

// TestPreCommitHook_CleansUpItsCandidateFile pins the hygiene half: the guard's
// scratch copy of the staged content lives in the gitignored local tier, not in a
// shared $TMPDIR, and it is gone whichever way the hook exits.
func TestPreCommitHook_CleansUpItsCandidateFile(t *testing.T) {
	r := newHookRepo(t, keyedBanlist)
	r.write("note.md", "the widgetworks deal closes friday\n")
	r.git("add", "note.md")
	if blocked, out := r.commit(); !blocked {
		t.Fatalf("commit not blocked\n%s", out)
	}
	left, err := os.ReadDir(filepath.Join(r.dir, ".abcd", ".work.local"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range left {
		if e.Name() != "private-names.txt" {
			t.Errorf("the guard left %q behind; the candidate copy of the staged content is trapped on every exit", e.Name())
		}
	}
}

// TestPreCommitHook_FailsClosedWhenItCannotReadStagedContent pins the direction the
// guard must fail in: "could not compute" is never allowed to look like "clean". A
// staged path whose blob cannot be produced (the index points at an object that is
// not in the object store) refuses the commit and names the step, not the content.
func TestPreCommitHook_FailsClosedWhenItCannotReadStagedContent(t *testing.T) {
	r := newHookRepo(t, keyedBanlist)
	r.write("note.md", "nothing sensitive here\n")
	r.git("add", "note.md")
	// Empty the object store: the index still names note.md, so listing the staged
	// paths succeeds and `git show :note.md` then cannot produce the blob. This is the
	// "could not compute" case, and the guard must say so rather than fall through.
	objects := filepath.Join(r.dir, ".git", "objects")
	if err := os.RemoveAll(objects); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(objects, 0o755); err != nil {
		t.Fatal(err)
	}
	_, out := r.commit()
	if !strings.Contains(out, "could not read the staged content") {
		t.Errorf("the guard did not report that it could not read the staged content; a check that cannot run must never look like a check that passed\n%s", out)
	}
}

// TestPreCommitHook_RefusesANonRegularStore pins tampering as tampering. A symlink
// at the store's path (or at its directory) silently swaps the guard's list for an
// empty or attacker-chosen one — and `git add -f` defeats the gitignore, so a
// checkout can materialise it. A directory or FIFO there would take the "absent"
// branch and read as "this machine has not opted in". Both are refusals.
func TestPreCommitHook_RefusesANonRegularStore(t *testing.T) {
	t.Run("symlinked store", func(t *testing.T) {
		r := newHookRepo(t, keyedBanlist)
		store := filepath.Join(r.dir, ".abcd", ".work.local", "private-names.txt")
		decoy := filepath.Join(r.dir, "decoy.txt")
		if err := os.WriteFile(decoy, []byte("# abcd-banlist: keyed\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(store); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(decoy, store); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		r.write("note.md", "the widgetworks deal closes friday\n")
		r.git("add", "note.md")
		blocked, out := r.commit()
		if !blocked {
			t.Fatalf("a symlinked banlist silently replaced the guard's list\n%s", out)
		}
		if !strings.Contains(out, "SYMLINK") {
			t.Errorf("refusal does not name the cause\n%s", out)
		}
	})

	t.Run("directory at the store path", func(t *testing.T) {
		r := newHookRepo(t, keyedBanlist)
		store := filepath.Join(r.dir, ".abcd", ".work.local", "private-names.txt")
		if err := os.Remove(store); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(store, 0o700); err != nil {
			t.Fatal(err)
		}
		r.write("note.md", "the widgetworks deal closes friday\n")
		r.git("add", "note.md")
		blocked, out := r.commit()
		if !blocked {
			t.Fatalf("a directory at the store path was read as a machine that never opted in\n%s", out)
		}
		if !strings.Contains(out, "not a regular file") {
			t.Errorf("refusal does not name the cause\n%s", out)
		}
	})
}

// TestPreCommitHook_DoesNotTraceThePatternsUnderXtrace pins the one thing an
// inherited shell option must not be able to do: turn the guard into a printer of
// the very list it protects. An exported SHELLOPTS=xtrace switches tracing on for
// the hook's own shell before its first line runs, so the guard turns it back off.
func TestPreCommitHook_DoesNotTraceThePatternsUnderXtrace(t *testing.T) {
	r := newHookRepo(t, keyedBanlist)
	r.write("note.md", "nothing sensitive here\n")
	r.git("add", "note.md")
	r.env = append(r.env, "SHELLOPTS=xtrace")
	blocked, out := r.commit()
	if blocked {
		t.Fatalf("commit refused for clean content\n%s", out)
	}
	if strings.Contains(out, "widgetworks") {
		t.Errorf("an inherited xtrace printed the banlist patterns:\n%s", out)
	}
}

// TestPreCommitHook_BOMDoesNotDowngradeAKeyedStore is the shell half of the
// fail-open a leading byte-order mark opened in BOTH readers. With the BOM the
// first line no longer equalled the declaration, so the guard read the store as
// LEGACY and every keyed entry became one whole-line pattern
// (`lab-host carol-server\.example\.net`) that matches nothing a commit contains —
// while the entry count stayed >=1, so the loud zero-entry warning never fired.
// The store looked healthy and checked nothing.
func TestPreCommitHook_BOMDoesNotDowngradeAKeyedStore(t *testing.T) {
	blocked, out := hookRun(t,
		"\xef\xbb\xbf# abcd-banlist: keyed\nlab-host carol-server\\.example\\.net\n",
		"ssh carol-server.example.net\n")
	if !blocked {
		t.Fatalf("a BOM before the declaration downgraded the store and let a banned name through\n%s", out)
	}
	if !strings.Contains(out, "lab-host") {
		t.Errorf("the refusal does not name the key\n%s", out)
	}
	if strings.Contains(out, "carol-server") {
		t.Errorf("the refusal echoes the pattern or the matched text\n%s", out)
	}
	if !strings.Contains(out, "keyed store") {
		t.Errorf("the guard did not announce the keyed format it actually read\n%s", out)
	}
}

// TestPreCommitHook_DamagedDeclarationRefuses: a first line that is not the
// declaration but would be after stripping leading blanks is a DAMAGED
// declaration, not a legacy store. Reading it as legacy is the same silent
// downgrade, so the guard fails closed and says which line to fix.
func TestPreCommitHook_DamagedDeclarationRefuses(t *testing.T) {
	blocked, out := hookRun(t,
		"  # abcd-banlist: keyed\nlab-host carol-server\\.example\\.net\n",
		"nothing banned here\n")
	if !blocked {
		t.Fatalf("a damaged format declaration was read as a legacy store\n%s", out)
	}
	if !strings.Contains(out, "line 1") {
		t.Errorf("the refusal does not name the damaged line\n%s", out)
	}
}

// TestPreCommitHook_RefusesAStagedCopyOfTheStore is the leak the store-path
// refusal did not close. It matched only the local tier's path, so a COPY of the
// store anywhere else — notes.txt, a .bak beside it — committed every private
// pattern in clear. The entries cannot catch their own text (they are escaped
// regular expressions, and `carol-server\.example\.net` does not match itself), so
// the guard announced a clean check while the file it exists to protect went into
// history.
func TestPreCommitHook_RefusesAStagedCopyOfTheStore(t *testing.T) {
	body := "# abcd-banlist: keyed\nlab-host carol-server\\.example\\.net\n"
	for _, dest := range []string{"notes.txt", ".abcd/private-names.bak", "docs/copy.md"} {
		t.Run(dest, func(t *testing.T) {
			r := newHookRepo(t, body)
			r.write(dest, body)
			r.git("add", "--", dest)
			blocked, out := r.commit()
			if !blocked {
				t.Fatalf("a verbatim copy of the private store committed clean\n%s", out)
			}
			if strings.Contains(out, "carol-server") {
				t.Errorf("the refusal echoes the store's content\n%s", out)
			}
		})
	}
}

// TestPreCommitHook_RefusesAStoreCopyByFilename covers the store shape the
// first-line test cannot see: a LEGACY store declares no format, so a copy of one
// is indistinguishable from any other text file by content alone. Its filename is
// the remaining signal, and it is a path, not a secret.
func TestPreCommitHook_RefusesAStoreCopyByFilename(t *testing.T) {
	r := newHookRepo(t, "carol-server\\.example\\.net\n")
	r.write("backup/private-names.txt", "carol-server\\.example\\.net\n")
	r.git("add", "--", "backup/private-names.txt")
	blocked, out := r.commit()
	if !blocked {
		t.Fatalf("a legacy store copied under its own filename committed clean\n%s", out)
	}
	if !strings.Contains(out, "backup/private-names.txt") {
		t.Errorf("the refusal does not name the staged path\n%s", out)
	}
}

// TestPreCommitHook_RefusesARenameOfTheStore: for a rename record the loop read the
// source path and then OVERWROTE it with the destination, so `git mv` of the store
// itself presented only a destination outside the tier and walked past the refusal.
func TestPreCommitHook_RefusesARenameOfTheStore(t *testing.T) {
	r := newHookRepo(t, "# abcd-banlist: keyed\nlab-host carol-server\\.example\\.net\n")
	// Track the store first (the shape a `git add -f` accident leaves behind), then
	// rename it out of the tier: the source path is the only evidence left.
	r.git("add", "-f", "--", ".abcd/.work.local/private-names.txt")
	r.git("-c", "core.hooksPath=/dev/null", "commit", "-m", "seed")
	r.git("mv", ".abcd/.work.local/private-names.txt", "keep.md")
	blocked, out := r.commit()
	if !blocked {
		t.Fatalf("renaming the private store out of the tier committed clean\n%s", out)
	}
	if strings.Contains(out, "carol-server") {
		t.Errorf("the refusal echoes the store's content\n%s", out)
	}
}

// TestPreCommitHook_ExemptsADeclaredExample is the escape the copy refusals need.
// The refusals are shape tests, and a repo that legitimately commits a store-shaped
// file — this repo's own fixture corpora, a doc quoting the declaration — had no way
// to say so: `--no-verify` is an off switch for the whole guard, not a per-file
// escape. A blob whose SECOND line is the marker is exempt from the copy refusals
// and from nothing else.
func TestPreCommitHook_ExemptsADeclaredExample(t *testing.T) {
	r := newHookRepo(t, "# abcd-banlist: keyed\nlab-host carol-server\\.example\\.net\n")
	r.write("docs/fixture.txt", "# abcd-banlist: keyed\n# abcd-banlist-example\nlab-host alice-laptop\\.example\\.com\n")
	r.git("add", "--", "docs/fixture.txt")
	if blocked, out := r.commit(); blocked {
		t.Fatalf("a declared example was refused as a copy of the store\n%s", out)
	}
}

// TestPreCommitHook_ExemptExampleIsStillScanned: the escape exempts a blob from the
// COPY refusals only. A banned name inside a declared example is a banned name in
// the commit, and an escape that also stopped pattern matching would be a way to
// commit anything.
func TestPreCommitHook_ExemptExampleIsStillScanned(t *testing.T) {
	r := newHookRepo(t, "# abcd-banlist: keyed\nlab-host carol-server\\.example\\.net\n")
	r.write("docs/fixture.txt", "# abcd-banlist: keyed\n# abcd-banlist-example\nssh carol-server.example.net\n")
	r.git("add", "--", "docs/fixture.txt")
	blocked, out := r.commit()
	if !blocked {
		t.Fatalf("a banned name inside a declared example committed clean\n%s", out)
	}
	if !strings.Contains(out, "lab-host") {
		t.Errorf("the refusal does not name the key\n%s", out)
	}
}

// TestPreCommitHook_CommitsTheSharedCorpora is correctness M1's reproduction: the
// first-line refusal blocked this repo's OWN committed fixtures, so the change that
// added the refusal could not be committed by the guard it was hardening.
func TestPreCommitHook_CommitsTheSharedCorpora(t *testing.T) {
	r := newHookRepo(t, "# abcd-banlist: keyed\nlab-host carol-server\\.example\\.net\n")
	for _, name := range []string{"parse-corpus.txt", "parse-corpus-malformed.txt", "parse-corpus-duplicate-decl.txt"} {
		r.write("testdata/"+name, corpus(t, name))
	}
	r.git("add", "--", "testdata")
	if blocked, out := r.commit(); blocked {
		t.Fatalf("the shared corpora cannot be committed by the guard they prove\n%s", out)
	}
}

// TestPreCommitHook_DuplicatedDeclarationRefuses drives the duplicate-declaration
// corpus through the shell reader. Its Go half is TestKeyedStoreRefusesADuplicated-
// Declaration: one fixture, both readers, so a divergence is a test failure rather
// than a status board that disagrees with the guard.
func TestPreCommitHook_DuplicatedDeclarationRefuses(t *testing.T) {
	blocked, out := hookRun(t, corpus(t, "parse-corpus-duplicate-decl.txt"), "nothing sensitive here\n")
	if !blocked {
		t.Fatalf("a keyed store with a duplicated declaration was accepted\n%s", out)
	}
	if !strings.Contains(out, "line 8") {
		t.Errorf("the refusal does not name the duplicated line\n%s", out)
	}
}

// TestPreCommitHook_LeavesAForeignRepoAlone is security MIN-4. The guard runs in
// whatever repo a clone points at it, including one that never opted in. Creating
// its scratch directory unconditionally left an abcd directory — in a repo with no
// abcd fence, so an untracked one — sitting in the working tree after every commit.
func TestPreCommitHook_LeavesAForeignRepoAlone(t *testing.T) {
	r := newHookRepo(t, "") // no store: this machine has not opted in
	r.write("notes.md", "nothing sensitive here\n")
	r.git("add", "notes.md")
	if blocked, out := r.commit(); blocked {
		t.Fatalf("a clean commit was refused\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(r.dir, ".abcd", ".work.local")); err == nil {
		t.Error("the guard created its local tier in a repo that never opted in")
	}
}

// TestPreCommitHook_RefusesAStoreCopyWithNoLocalStore: the copy refusals are about
// what a COMMIT is carrying, not about what this machine has. They kept working
// after the scratch directory stopped being created unconditionally.
func TestPreCommitHook_RefusesAStoreCopyWithNoLocalStore(t *testing.T) {
	r := newHookRepo(t, "")
	r.write("leaked.txt", "# abcd-banlist: keyed\nlab-host carol-server\\.example\\.net\n")
	r.git("add", "leaked.txt")
	blocked, out := r.commit()
	if !blocked {
		t.Fatalf("a copy of someone's store committed clean on a machine with no store\n%s", out)
	}
	if strings.Contains(out, "carol-server") {
		t.Errorf("the refusal echoes the store's content\n%s", out)
	}
}

// TestPreCommitHook_BOMHeaderCorpusRefuses drives the BOM-header corpus through the
// shell reader. Its Go half is TestParseRefusesTheBOMHeaderCorpus: the shell already
// refused this file while Go called it healthy, which is the divergence the shared
// fixture exists to make impossible.
func TestPreCommitHook_BOMHeaderCorpusRefuses(t *testing.T) {
	blocked, out := hookRun(t, corpus(t, "parse-corpus-bom-header.txt"), "nothing sensitive here\n")
	if !blocked {
		t.Fatalf("a declaration below a BOM'd header line was accepted\n%s", out)
	}
	if !strings.Contains(out, "line 8") {
		t.Errorf("the refusal does not name the misplaced declaration's line\n%s", out)
	}
}

// newWorktreeCase builds a primary checkout carrying primaryBody as its private
// store (an empty body leaves the store absent) and returns a hookRepo bound to a
// LINKED git worktree of it. It is the exact shape iss-370 was found in: the
// local-ephemeral tier is per-worktree and gitignored, so a freshly added worktree
// starts with no store at all while the primary checkout has one, and the SAME
// committed hook runs in both (a linked worktree resolves its hooks through the
// common git dir).
func newWorktreeCase(t *testing.T, primaryBody string) (primary, linked *hookRepo) {
	t.Helper()
	primary = newHookRepo(t, primaryBody)
	// The tier's fence is committed, so the linked worktree inherits it and its own
	// store is ignored there too — without it a `git add -A` in the worktree would
	// stage the store and a shape test could pass for the wrong reason.
	primary.write(".gitignore", ".abcd/.work.local/\n")
	primary.write("seed.md", "nothing sensitive here\n")
	primary.git("add", "--", ".gitignore", "seed.md")
	// The guard is bypassed for the seed alone: `git worktree add` needs a commit to
	// check out, and the fixture is not what this test is proving.
	primary.git("-c", "core.hooksPath=/dev/null", "commit", "-m", "seed")

	dir := filepath.Join(t.TempDir(), "linked")
	if out, err := primary.tryGit("worktree", "add", "-b", "linked", dir); err != nil {
		t.Skipf("git worktree add unavailable: %v\n%s", err, out)
	}
	return primary, &hookRepo{t: t, dir: dir, env: primary.env}
}

// TestPreCommitHook_LinkedWorktreeInheritsThePrimaryStore is iss-370. The private
// store lives in the gitignored per-worktree local tier, so every commit made from
// a `git worktree add` checkout ran with the banlist ABSENT — warned, but
// unprotected, which made the isolated-agent pattern a systematic bypass of a
// protection the main checkout has. The guard resolves the primary checkout's store
// as a fallback layer and blocks on it, naming the key alone as ever.
func TestPreCommitHook_LinkedWorktreeInheritsThePrimaryStore(t *testing.T) {
	_, linked := newWorktreeCase(t, keyedBanlist)
	// No per-worktree banlist setup of any kind has been performed here.
	if _, err := os.Stat(filepath.Join(linked.dir, ".abcd", ".work.local")); err == nil {
		t.Fatal("the linked worktree already carries a local tier; the fixture proves nothing")
	}
	linked.write("note.md", "the widgetworks deal closes friday\n")
	linked.git("add", "note.md")
	blocked, out := linked.commit()
	if !blocked {
		t.Fatalf("a linked worktree committed a name the primary checkout's store bans\n%s", out)
	}
	if !strings.Contains(out, "widget-partner") {
		t.Errorf("the refusal does not name the entry key\n%s", out)
	}
	for _, leak := range []string{"widgetworks", "friday"} {
		if strings.Contains(strings.ToLower(out), leak) {
			t.Errorf("output leaks %q; neither the pattern nor the matched line may be echoed\n%s", leak, out)
		}
	}
}

// TestPreCommitHook_LinkedWorktreeStoreWinsOverThePrimary pins the precedence rule:
// the primary store is a FALLBACK, so an entry the worktree declares for itself is
// enforced there whether or not the primary knows the name — and the primary's own
// entries keep being enforced beside it. A fallback that displaced the local layer
// would silently narrow the guard in the checkout the developer is actually in.
func TestPreCommitHook_LinkedWorktreeStoreWinsOverThePrimary(t *testing.T) {
	// Two fixtures rather than two staged files in one: a blocked commit leaves its
	// file staged, so a second case in the same checkout would be scanned against the
	// first one's content and could pass for the wrong reason.
	const localOnly = "# abcd-banlist: keyed\nlocal-only   sprocketco\n"

	t.Run("local entry blocks", func(t *testing.T) {
		_, linked := newWorktreeCase(t, keyedBanlist)
		linked.writeBanlist(localOnly)
		linked.write("local.md", "the sprocketco pilot starts monday\n")
		linked.git("add", "local.md")
		blocked, out := linked.commit()
		if !blocked {
			t.Fatalf("the worktree's own store did not block\n%s", out)
		}
		if !strings.Contains(out, "local-only") {
			t.Errorf("the refusal does not name the local entry's key\n%s", out)
		}
	})

	t.Run("primary entry still blocks beside it", func(t *testing.T) {
		_, linked := newWorktreeCase(t, keyedBanlist)
		linked.writeBanlist(localOnly)
		linked.write("inherited.md", "the widgetworks deal closes friday\n")
		linked.git("add", "inherited.md")
		blocked, out := linked.commit()
		if !blocked {
			t.Fatalf("a worktree-local store displaced the inherited primary layer\n%s", out)
		}
		if !strings.Contains(out, "widget-partner") {
			t.Errorf("the refusal does not name the inherited entry's key\n%s", out)
		}
	})
}

// noticeLines returns the hook's name-guard NOTICE lines: every `abcd name-guard:`
// line, which is the success-path announcement. Refusals and the loud banners carry
// their own prefixes and are not counted.
func noticeLines(out string) []string {
	var got []string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "abcd name-guard:") {
			got = append(got, l)
		}
	}
	return got
}

// TestPreCommitHook_AnnouncesOncePerCommit is iss-2609181122202952. The guard
// announced each store it read on two lines (the format before the parse, the count
// after it) and prefixed an inherited store with a third, so a clean commit from a
// linked worktree printed three notice lines, and five once the worktree carried a
// store of its own. A notice that repeats reads as a hook running several times. One
// commit, one notice line, however many stores were read — and that line still names
// each store's format, its count, and where an inherited one lives.
func TestPreCommitHook_AnnouncesOncePerCommit(t *testing.T) {
	t.Run("standalone checkout", func(t *testing.T) {
		r := newHookRepo(t, keyedBanlist)
		r.write("note.md", "nothing sensitive here\n")
		r.git("add", "note.md")
		if blocked, out := r.commit(); blocked {
			t.Fatalf("clean content was refused\n%s", out)
		} else if got := noticeLines(out); len(got) != 1 {
			t.Errorf("a clean commit printed %d name-guard notice lines, want 1\n%s", len(got), out)
		} else if !strings.Contains(got[0], "keyed store") || !strings.Contains(got[0], "1 entry") {
			t.Errorf("the notice does not name the format and the count it read\n%s", got[0])
		}
	})

	for name, local := range map[string]string{
		"linked worktree inheriting":                "",
		"linked worktree inheriting beside its own": "legacy-name\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, linked := newWorktreeCase(t, keyedBanlist)
			if local != "" {
				linked.writeBanlist(local)
			}
			linked.write("note.md", "nothing sensitive here\n")
			linked.git("add", "note.md")
			blocked, out := linked.commit()
			if blocked {
				t.Fatalf("clean content was refused\n%s", out)
			}
			got := noticeLines(out)
			if len(got) != 1 {
				t.Fatalf("a clean commit from a linked worktree printed %d name-guard notice lines, want 1\n%s", len(got), out)
			}
			if !strings.Contains(got[0], "primary checkout") || !strings.Contains(got[0], "keyed store") {
				t.Errorf("the one notice does not say which store was inherited and in what format\n%s", got[0])
			}
			if local != "" && !strings.Contains(got[0], "legacy store") {
				t.Errorf("the one notice does not name the worktree's own store beside the inherited one\n%s", got[0])
			}
		})
	}
}

// TestPreCommitHook_LinkedWorktreeSaysWhereTheEntryCameFrom: an inherited refusal
// names a key the developer will not find in the checkout they are standing in, so
// the guard says which store it came from. Without it the remedy — edit the primary
// checkout's store — is invisible, and the key alone reads as a phantom.
func TestPreCommitHook_LinkedWorktreeSaysWhereTheEntryCameFrom(t *testing.T) {
	_, linked := newWorktreeCase(t, keyedBanlist)
	linked.write("note.md", "the widgetworks deal closes friday\n")
	linked.git("add", "note.md")
	_, out := linked.commit()
	if !strings.Contains(out, "primary checkout") {
		t.Errorf("the refusal does not say the entry was inherited from the primary checkout\n%s", out)
	}
}

// TestPreCommitHook_LinkedWorktreeWithNoStoreAnywhereStillWarns is AC4 in the
// worktree shape: resolution is a fallback, never a fail-closed error. A linked
// worktree whose primary checkout has no store either is exactly as unprotected as
// a standalone checkout with none, and must be exactly as loud about it.
func TestPreCommitHook_LinkedWorktreeWithNoStoreAnywhereStillWarns(t *testing.T) {
	_, linked := newWorktreeCase(t, "")
	linked.write("note.md", "widgetworks ships today\n")
	linked.git("add", "note.md")
	blocked, out := linked.commit()
	if blocked {
		t.Fatalf("commit blocked with no store in either checkout; want it to proceed\n%s", out)
	}
	for _, want := range []string{"WARNING", "INACTIVE"} {
		if !strings.Contains(out, want) {
			t.Errorf("hook output does not mention %q; the inactive layer must announce itself\n%s", want, out)
		}
	}
}

// TestPreCommitHook_BareRepoWorktreeDoesNotInheritASiblingStore is the false
// positive the common-dir arithmetic opens if it is trusted alone. A worktree of a
// BARE repository (or one made with `--separate-git-dir`) has a common dir whose
// parent is not a working tree at all — it is whatever directory happens to hold
// the git dir — so the guard would read `<that dir>/.abcd/.work.local/…` and
// enforce a store belonging to some unrelated repository that merely lives next
// door. The primary root is therefore required to BE a working tree.
func TestPreCommitHook_BareRepoWorktreeDoesNotInheritASiblingStore(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	locateHook(t)
	env := gittest.Env(t)
	parent := t.TempDir()
	git := func(dir string, args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	// A neighbour repository's private store, sitting in the directory that holds
	// the bare git dir. Nothing about this worktree entitles it to that list.
	neighbour := filepath.Join(parent, ".abcd", ".work.local")
	if err := os.MkdirAll(neighbour, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(neighbour, "private-names.txt"), []byte(keyedBanlist), 0o600); err != nil {
		t.Fatal(err)
	}

	// A source repo with one commit, then a bare clone whose worktree we take.
	src := newHookRepo(t, "")
	src.write("seed.md", "nothing sensitive here\n")
	src.git("add", "seed.md")
	src.git("-c", "core.hooksPath=/dev/null", "commit", "-m", "seed")
	bare := filepath.Join(parent, "repo.git")
	if out, err := git(parent, "clone", "--bare", src.dir, bare); err != nil {
		t.Skipf("bare clone unavailable: %v\n%s", err, out)
	}
	linkedDir := filepath.Join(t.TempDir(), "linked")
	if out, err := git(bare, "worktree", "add", linkedDir, "HEAD"); err != nil {
		t.Skipf("git worktree add unavailable: %v\n%s", err, out)
	}
	// Install the hook where a worktree of the bare repo resolves it.
	hooksDir := filepath.Join(bare, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src2, err := os.ReadFile(locateHook(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), src2, 0o755); err != nil {
		t.Fatal(err)
	}

	linked := &hookRepo{t: t, dir: linkedDir, env: env}
	linked.git("config", "user.name", "Alice Example")
	linked.git("config", "user.email", "alice@example.com")
	linked.write("note.md", "the widgetworks deal closes friday\n")
	linked.git("add", "note.md")
	blocked, out := linked.commit()
	if blocked {
		t.Fatalf("the guard enforced a store belonging to a repository next door to the git dir\n%s", out)
	}
	if strings.Contains(out, "inheriting") {
		t.Errorf("the guard claimed to inherit a store from a directory that is not a working tree\n%s", out)
	}
}

// TestPreCommitHook_AnEmptiedWorktreeStoreStillWarns is the fail-quiet the
// two-store guard opens if the zero-entry banner is keyed on the SUM. The
// worktree's own store is truncated to its declaration line — the shape a refresh
// that garbles the store produces — while the inherited store still has an entry.
// Summed, the count is 1 and the banner never fires: every entry the developer
// declared in this worktree has stopped being checked, silently, which is exactly
// what the banner exists to catch.
func TestPreCommitHook_AnEmptiedWorktreeStoreStillWarns(t *testing.T) {
	_, linked := newWorktreeCase(t, keyedBanlist)
	linked.writeBanlist("# abcd-banlist: keyed\n")
	linked.write("note.md", "nothing sensitive here\n")
	linked.git("add", "note.md")
	blocked, out := linked.commit()
	if blocked {
		t.Fatalf("clean content was refused\n%s", out)
	}
	if !strings.Contains(out, "NO ENTRIES") {
		t.Errorf("an emptied worktree store was silent because the inherited one had an entry\n%s", out)
	}
}

// TestPreCommitHook_NeverPrintsThePrimaryCheckoutsPath is the guard's own
// confidentiality contract applied to the new fallback. A checkout's directory name
// is very often the private name its store bans — a project codename is the
// commonest entry there is — and the inheritance announcement runs on the SUCCESS
// path of EVERY commit, so an absolute path there prints the banned string to
// stderr, scrollback and any log that captures hook output, routinely.
func TestPreCommitHook_NeverPrintsThePrimaryCheckoutsPath(t *testing.T) {
	primary, linked := newWorktreeCase(t, keyedBanlist)
	linked.write("note.md", "nothing sensitive here\n")
	linked.git("add", "note.md")
	if blocked, out := linked.commit(); blocked {
		t.Fatalf("clean content was refused\n%s", out)
	}
	_, clean := linked.commit()
	linked.write("banned.md", "the widgetworks deal closes friday\n")
	linked.git("add", "banned.md")
	blocked, refusal := linked.commit()
	if !blocked {
		t.Fatalf("the inherited entry did not block\n%s", refusal)
	}
	for name, out := range map[string]string{"clean commit": clean, "refusal": refusal} {
		if strings.Contains(out, primary.dir) {
			t.Errorf("the %s prints the primary checkout's absolute path, whose directory name may itself be a banned string\n%s", name, out)
		}
		if !strings.Contains(out, "primary checkout") {
			t.Errorf("the %s does not say the store was inherited, so the remedy is invisible\n%s", name, out)
		}
	}
}

// TestPreCommitHook_UnreadableStoreRefusesLoudly: a store that cannot be OPENED
// checks exactly as little as one that cannot be parsed, and it must refuse the
// same way. Left to `set -e` it surfaced as a bare "Permission denied" and a mute
// non-zero exit, which reads like a broken repo rather than a guard that refused —
// the standard this file already holds a missing TOOL to.
func TestPreCommitHook_UnreadableStoreRefusesLoudly(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: an unreadable file is still readable")
	}
	r := newHookRepo(t, keyedBanlist)
	store := filepath.Join(r.dir, ".abcd", ".work.local", "private-names.txt")
	if err := os.Chmod(store, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(store, 0o600) })
	r.write("note.md", "nothing sensitive here\n")
	r.git("add", "note.md")
	blocked, out := r.commit()
	if !blocked {
		t.Fatalf("an unreadable store let a commit through\n%s", out)
	}
	if !strings.Contains(out, "BLOCKED") || !strings.Contains(out, "cannot be read") {
		t.Errorf("the refusal does not name the cause; a check that could not run must say so\n%s", out)
	}
}

// mirrorAttackCase builds the layout the `.git` existence test does not survive: a
// VICTIM checkout that is a real working tree carrying a private store, and a
// linked worktree of an unrelated repository whose git dir was placed INSIDE that
// checkout. shape selects the route — "bare" clones a mirror into the victim,
// "separate-git-dir" points a fresh repo's git dir there. Both leave the common
// dir's parent equal to the victim, and the victim carries a `.git`.
//
// It returns the victim's root and a hookRepo bound to the linked worktree, with
// the committed hook installed where that worktree resolves it.
func mirrorAttackCase(t *testing.T, shape string) (victim string, linked *hookRepo) {
	t.Helper()
	hook := locateHook(t)
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
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
	victim = filepath.Join(root, "victim")
	seed(victim)
	// The victim's private store. Its key is what a leak would print.
	local := filepath.Join(victim, ".abcd", ".work.local")
	if err := os.MkdirAll(local, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(local, "private-names.txt"), []byte(keyedBanlist), 0o600); err != nil {
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
	case "separate-git-dir":
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
	default:
		t.Fatalf("unknown shape %q", shape)
	}

	dir := filepath.Join(t.TempDir(), "linked")
	if out, err := git(commonDir, "worktree", "add", dir, "HEAD"); err != nil {
		t.Skipf("git worktree add unavailable: %v\n%s", err, out)
	}
	// A linked worktree resolves its hooks through the common git dir, so that is
	// where the guard has to be installed for it to run at all.
	hooksDir := filepath.Join(commonDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(hook)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), body, 0o755); err != nil {
		t.Fatal(err)
	}
	linked = &hookRepo{t: t, dir: dir, env: env}
	linked.git("config", "user.name", "Alice Example")
	linked.git("config", "user.email", "alice@example.com")
	return victim, linked
}

// TestPreCommitHook_AMirrorInsideAnotherCheckoutDoesNotInheritItsStore is the
// escalation of TestPreCommitHook_BareRepoWorktreeDoesNotInheritASiblingStore: the
// `.git` existence test only refuses when the directory holding the git dir is NOT
// a checkout. Put the bare mirror inside a real working tree and the test passes,
// and the guard enforces an unrelated repository's private list on this repo's
// commits — an oracle over its patterns, disclosure of its keys into this repo's
// hook output, and a cross-repo denial of service from one malformed line over
// there.
func TestPreCommitHook_AMirrorInsideAnotherCheckoutDoesNotInheritItsStore(t *testing.T) {
	for _, shape := range []string{"bare", "separate-git-dir"} {
		t.Run(shape, func(t *testing.T) {
			_, linked := mirrorAttackCase(t, shape)
			linked.write("note.md", "the widgetworks deal closes friday\n")
			linked.git("add", "note.md")
			blocked, out := linked.commit()
			if blocked {
				t.Fatalf("the guard enforced the private store of a checkout that merely holds this repository's git dir\n%s", out)
			}
			if strings.Contains(out, "widget-partner") {
				t.Errorf("the hook disclosed a key from an unrelated repository's private store\n%s", out)
			}
			if strings.Contains(out, "primary checkout") {
				t.Errorf("the guard claimed to inherit a store from a directory that is not this repository's working tree\n%s", out)
			}
		})
	}
}
