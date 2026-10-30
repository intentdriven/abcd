package identity

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// isolate points git at empty global/system config so the test only sees the
// temp repo's local config, making identity resolution hermetic regardless of
// the machine's real git identity.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	// EffectiveIdentity now honours the GIT_AUTHOR_* overrides (as git does), so
	// clear any ambient values to keep the config-based cases hermetic.
	t.Setenv("GIT_AUTHOR_NAME", "")
	t.Setenv("GIT_AUTHOR_EMAIL", "")
}

func gitRepo(t *testing.T, name, email string) string {
	t.Helper()
	dir := t.TempDir()
	runGitT(t, dir, "init")
	if name != "" {
		runGitT(t, dir, "config", "user.name", name)
	}
	if email != "" {
		runGitT(t, dir, "config", "user.email", email)
	}
	return dir
}

func runGitT(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// writePin records body as the person's own machine-wide identity record,
// under the test's own HOME (gittest.Env pins HOME to a per-test temp dir).
// The name is kept from when the identity was committed into the repository;
// root is unused, and nothing is written there: the repository names no one
// (itd-2610240918547728).
func writePin(t *testing.T, _ string, body string) {
	t.Helper()
	writeHomeFile(t, MachineRecordRef().Rel, body)
}

// writeHomeFile writes body at rel under the test HOME, the folders 0700 and
// the file 0600 as abcd writes them.
func writeHomeFile(t *testing.T, rel, body string) string {
	t.Helper()
	gittest.Env(t)
	p := filepath.Join(os.Getenv("HOME"), filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCheck_Match(t *testing.T) {
	isolate(t)
	dir := gitRepo(t, "Alice", "alice@example.com")
	writePin(t, dir, `{"name":"Alice","email":"alice@example.com"}`)
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusOK {
		t.Fatalf("want StatusOK, got %v — %s", res.Status, res.Reason)
	}
}

func TestCheck_Mismatch(t *testing.T) {
	isolate(t)
	dir := gitRepo(t, "Test User", "test@example.com")
	writePin(t, dir, `{"name":"Alice","email":"alice@example.com"}`)
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusMismatch {
		t.Fatalf("want StatusMismatch, got %v — %s", res.Status, res.Reason)
	}
	if res.Effective.Email != "test@example.com" || res.Record.Email != "alice@example.com" {
		t.Fatalf("result did not carry both identities: %+v", res)
	}
}

func TestCheck_NoRecord(t *testing.T) {
	isolate(t)
	dir := gitRepo(t, "Alice", "alice@example.com")
	// no record written
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusNoRecord {
		t.Fatalf("want StatusNoRecord, got %v — %s", res.Status, res.Reason)
	}
	if res.Blocks() {
		t.Fatal("a human identity with no record must commit (criterion 5: whatever launched the run supplied it)")
	}
}

// TestCheck_IgnoresACommittedPin is criterion 2 at the gate: a repository that
// still commits another person's identity no longer decides who may commit.
// The contributor's own human identity, with no record of their own, is not
// refused and the result names no one but them.
func TestCheck_IgnoresACommittedPin(t *testing.T) {
	isolate(t)
	dir := gitRepo(t, "Second Contributor", "second@example.com")
	writeRepoPin(t, dir, `{"name":"A Maintainer","email":"maintainer@example.com"}`)
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusNoRecord || res.Blocks() {
		t.Fatalf("a committed pin decided the gate: %v blocks=%v — %s", res.Status, res.Blocks(), res.Reason)
	}
	if strings.Contains(res.Reason+res.CommitterReason, "Maintainer") {
		t.Fatalf("the result names the maintainer: %+v", res)
	}
}

// TestCheck_RepoRecordOutranksTheMachineRecord: a person who commits to one
// repository under a work identity records it for that repository, keyed on
// its root commit, and it wins over their machine-wide default there.
func TestCheck_RepoRecordOutranksTheMachineRecord(t *testing.T) {
	isolate(t)
	dir := gitRepo(t, "Alice at Work", "alice@work.example")
	// isolate blanks GIT_AUTHOR_*, which git reads as an empty ident.
	t.Setenv("GIT_AUTHOR_NAME", "Alice at Work")
	t.Setenv("GIT_AUTHOR_EMAIL", "alice@work.example")
	runGitT(t, dir, "commit", "--allow-empty", "-m", "root", "--no-verify")
	sha := gittest.RootCommit(t, dir)
	writePin(t, dir, `{"name":"Alice","email":"alice@example.com"}`)
	ref, err := repoRecordRef(sha)
	if err != nil {
		t.Fatal(err)
	}
	writeHomeFile(t, ref.Rel, `{"name":"Alice at Work","email":"alice@work.example"}`)
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusOK || res.RecordAt != ref {
		t.Fatalf("want the repository's own record to decide: %v at %+v — %s", res.Status, res.RecordAt, res.Reason)
	}
}

// TestCheck_RefusedRecordFailsClosed: a record that is there and cannot be
// trusted (writable by another account, a symlink, unparseable) is an error,
// never a fall-through to "no record".
func TestCheck_RefusedRecordFailsClosed(t *testing.T) {
	for name, mutate := range map[string]func(t *testing.T, p string){
		"group-writable": func(t *testing.T, p string) {
			if err := os.Chmod(p, 0o620); err != nil {
				t.Fatal(err)
			}
		},
		"symlink": func(t *testing.T, p string) {
			target := p + ".real"
			if err := os.Rename(p, target); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, p); err != nil {
				t.Fatal(err)
			}
		},
		"malformed": func(t *testing.T, p string) {
			if err := os.WriteFile(p, []byte("{garbage"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"incomplete": func(t *testing.T, p string) {
			if err := os.WriteFile(p, []byte(`{"name":"Alice"}`), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			isolate(t)
			t.Setenv("HOME", t.TempDir())
			dir := gitRepo(t, "Alice", "alice@example.com")
			p := writeHomeFile(t, MachineRecordRef().Rel, `{"name":"Alice","email":"alice@example.com"}`)
			mutate(t, p)
			if _, err := Check(dir); err == nil || !errors.Is(err, ErrRecordRefused) {
				t.Fatalf("want ErrRecordRefused, got %v", err)
			}
		})
	}
}

// TestCheck_RecordBehindASymlinkedHomeIsRefused: the record is refused behind
// a symlinked user-level home, as the rules loader refuses rules.json there.
func TestCheck_RecordBehindASymlinkedHomeIsRefused(t *testing.T) {
	isolate(t)
	dir := gitRepo(t, "Alice", "alice@example.com")
	home := os.Getenv("HOME")
	real := filepath.Join(t.TempDir(), "real-home")
	if err := os.MkdirAll(real, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "identity.json"), []byte(`{"name":"Alice","email":"alice@example.com"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(home, filepath.FromSlash(abcdhome.Rel()))); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(dir); err == nil || !errors.Is(err, ErrRecordRefused) {
		t.Fatalf("want a record behind a symlinked home refused, got %v", err)
	}
}

func writeRepoPin(t *testing.T, root, body string) {
	t.Helper()
	cfg := filepath.Join(root, ".abcd", "config")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "identity.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheck_UnsetIdentity(t *testing.T) {
	isolate(t)
	dir := gitRepo(t, "", "") // no user.name/email anywhere
	writePin(t, dir, `{"name":"Alice","email":"alice@example.com"}`)
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusUnset {
		t.Fatalf("want StatusUnset, got %v — %s", res.Status, res.Reason)
	}
}

// TestCheck_AuthorEnvOverride guards B14: git stamps the author from
// GIT_AUTHOR_NAME/GIT_AUTHOR_EMAIL with higher precedence than user.name/email
// config, so a matching config must not pass the gate when the env overrides
// diverge from the pin. Pre-fix (config-only EffectiveIdentity) this returned
// StatusOK, letting a mis-attributed commit through.
func TestCheck_AuthorEnvOverride(t *testing.T) {
	isolate(t)
	dir := gitRepo(t, "Alice", "alice@example.com") // config matches the pin
	writePin(t, dir, `{"name":"Alice","email":"alice@example.com"}`)
	t.Setenv("GIT_AUTHOR_NAME", "Test User")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusMismatch {
		t.Fatalf("want StatusMismatch (env override diverges from pin), got %v — %s", res.Status, res.Reason)
	}
	if res.Effective.Name != "Test User" || res.Effective.Email != "test@example.com" {
		t.Fatalf("effective identity did not reflect GIT_AUTHOR_* override: %+v", res.Effective)
	}
}

func recordHome(t *testing.T) string {
	t.Helper()
	gittest.Env(t)
	return os.Getenv("HOME")
}

func TestWriteRecord_RoundTrip(t *testing.T) {
	home := recordHome(t)
	want := Record{Name: "Alice", Email: "alice@example.com"}
	if err := WriteRecord(home, MachineRecordRef(), want); err != nil {
		t.Fatal(err)
	}
	got, ref, ok, err := loadRecord(home, "")
	if err != nil || !ok {
		t.Fatalf("loadRecord after WriteRecord: ok=%v err=%v", ok, err)
	}
	if got != want || ref != MachineRecordRef() {
		t.Fatalf("round-trip mismatch: got %+v at %+v, want %+v", got, ref, want)
	}
	st, err := os.Stat(filepath.Join(home, filepath.FromSlash(ref.Rel)))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("the record was written %v, want 0600", st.Mode().Perm())
	}
}

func TestWriteRecord_RepoScope(t *testing.T) {
	home := recordHome(t)
	ref, err := repoRecordRef(strings.Repeat("ab", 20))
	if err != nil {
		t.Fatal(err)
	}
	want := Record{Name: "Alice at Work", Email: "alice@work.example"}
	if err := WriteRecord(home, ref, want); err != nil {
		t.Fatal(err)
	}
	got, at, ok, err := loadRecord(home, strings.Repeat("ab", 20))
	if err != nil || !ok || got != want || at != ref {
		t.Fatalf("got %+v at %+v ok=%v err=%v", got, at, ok, err)
	}
	if _, err := repoRecordRef("../../etc"); err == nil {
		t.Fatal("a root name that is not an object name must be refused")
	}
}

func TestWriteRecord_RequiresBothFields(t *testing.T) {
	if err := WriteRecord(recordHome(t), MachineRecordRef(), Record{Name: "Alice"}); err == nil {
		t.Fatal("want error when email is missing")
	}
}

// A record never names a machine: it is the person's own.
func TestWriteRecord_RefusesAMachineIdentity(t *testing.T) {
	if err := WriteRecord(recordHome(t), MachineRecordRef(), Record{Name: "Claude", Email: "noreply@anthropic.com"}); err == nil {
		t.Fatal("want a machine identity refused")
	}
}

// The self-contained pre-commit hook reads the record with a naive sed, so
// the stored value must round-trip through it (iss-63). WriteRecord refuses
// the characters JSON must always escape (double-quote, backslash, control),
// which the sed cannot read back, and stores &, <, > literally (no HTML
// escaping) so a legal git identity like "Marks & Spencer" is not
// fail-closed.
func TestWriteRecord_RejectsUnpinnableChars(t *testing.T) {
	home := recordHome(t)
	for _, r := range []Record{
		{Name: `Alice "the dev"`, Email: "alice@example.com"}, // double-quote
		{Name: `back\slash`, Email: "alice@example.com"},      // backslash
		{Name: "line\nbreak", Email: "alice@example.com"},     // control char
		{Name: "Alice", Email: `a"x@example.com`},             // quote in email
	} {
		if err := WriteRecord(home, MachineRecordRef(), r); err == nil {
			t.Fatalf("want error for unpinnable record %+v", r)
		}
	}
}

func TestWriteRecord_LegalIdentitiesRoundTrip(t *testing.T) {
	for _, want := range []Record{
		{Name: "Marks & Spencer", Email: "ci@m-and-s.example"}, // & is legal, must not HTML-escape
		{Name: "O'Brien", Email: "obrien@example.com"},         // apostrophe
		{Name: "José <team>", Email: "jose@example.com"},       // unicode + < >
	} {
		home := recordHome(t)
		if err := WriteRecord(home, MachineRecordRef(), want); err != nil {
			t.Fatalf("legal identity %+v must write: %v", want, err)
		}
		got, _, ok, err := loadRecord(home, "")
		if err != nil || !ok || got != want {
			t.Fatalf("round-trip %+v: got %+v ok=%v err=%v", want, got, ok, err)
		}
		raw, err := os.ReadFile(filepath.Join(home, filepath.FromSlash(MachineRecordRef().Rel)))
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range []rune{'&', '<', '>'} {
			if strings.ContainsRune(want.Name, r) && !strings.ContainsRune(string(raw), r) {
				t.Fatalf("record did not store %q literally (HTML-escaped?); the hook sed cannot read it back:\n%s", r, raw)
			}
		}
	}
}

// Blocks reports whether a pre-commit hook should refuse the commit: a
// mismatch or an unset identity blocks; OK and NoRecord do not, unless a
// machine identity is in either role, which blocks whatever the record.
func TestBlocks(t *testing.T) {
	cases := map[Status]bool{
		StatusOK:       false,
		StatusNoRecord: false,
		StatusMismatch: true,
		StatusUnset:    true,
	}
	for s, want := range cases {
		if got := (Result{Status: s}).Blocks(); got != want {
			t.Errorf("Status %v: Blocks()=%v, want %v", s, got, want)
		}
		if !(Result{Status: s, AuthorIsTool: true}).Blocks() || !(Result{Status: s, CommitterIsTool: true}).Blocks() {
			t.Errorf("Status %v: a machine identity must block", s)
		}
	}
}

// TestCheck_AuthorRoleConfigOutranksUser guards iss-2609261454332615: git stamps
// the author from author.name/author.email ahead of user.name/user.email (git
// 2.22+), so a repo-local author.name must be read as the author even when
// user.name matches the pin. Reading user.* alone reported the pinned identity
// as matching while every commit was authored by someone else.
func TestCheck_AuthorRoleConfigOutranksUser(t *testing.T) {
	isolate(t)
	dir := gitRepo(t, "Alice", "alice@example.com") // user.* matches the pin
	runGitT(t, dir, "config", "author.name", "Test User")
	writePin(t, dir, `{"name":"Alice","email":"alice@example.com"}`)
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusMismatch {
		t.Fatalf("want StatusMismatch (author.name outranks user.name), got %v — %s", res.Status, res.Reason)
	}
	if res.Effective.Name != "Test User" || res.Effective.Email != "alice@example.com" {
		t.Fatalf("effective author did not follow git's author.name precedence: %+v", res.Effective)
	}
}
