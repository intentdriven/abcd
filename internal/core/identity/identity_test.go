package identity

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/provenance"
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

func writePin(t *testing.T, root, body string) {
	t.Helper()
	dir := filepath.Join(root, ".abcd", "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "identity.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheck_Match(t *testing.T) {
	isolate(t)
	dir := gitRepo(t, "Alex Reppel", "alex@example.com")
	writePin(t, dir, `{"name":"Alex Reppel","email":"alex@example.com"}`)
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
	writePin(t, dir, `{"name":"Alex Reppel","email":"alex@example.com"}`)
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusMismatch {
		t.Fatalf("want StatusMismatch, got %v — %s", res.Status, res.Reason)
	}
	if res.Effective.Email != "test@example.com" || res.Pin.Email != "alex@example.com" {
		t.Fatalf("result did not carry both identities: %+v", res)
	}
}

func TestCheck_NoPin(t *testing.T) {
	isolate(t)
	dir := gitRepo(t, "Alex Reppel", "alex@example.com")
	// no identity.json written
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusNoPin {
		t.Fatalf("want StatusNoPin, got %v — %s", res.Status, res.Reason)
	}
}

func TestCheck_UnsetIdentity(t *testing.T) {
	isolate(t)
	dir := gitRepo(t, "", "") // no user.name/email anywhere
	writePin(t, dir, `{"name":"Alex Reppel","email":"alex@example.com"}`)
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
	dir := gitRepo(t, "Alex Reppel", "alex@example.com") // config matches the pin
	writePin(t, dir, `{"name":"Alex Reppel","email":"alex@example.com"}`)
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

func TestLoadPin_Malformed(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	writePin(t, dir, `{"name": "no closing quote`)
	if _, _, err := LoadPin(dir); err == nil {
		t.Fatal("want error on malformed identity.json, got nil")
	}
}

func TestLoadPin_MissingFields(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	writePin(t, dir, `{"name":"Alex Reppel"}`) // no email
	if _, _, err := LoadPin(dir); err == nil {
		t.Fatal("want error when email is empty, got nil")
	}
}

func TestWritePin_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := Pin{Name: "Alex Reppel", Email: "alex@example.com"}
	if err := WritePin(dir, want); err != nil {
		t.Fatal(err)
	}
	got, ok, err := LoadPin(dir)
	if err != nil || !ok {
		t.Fatalf("LoadPin after WritePin: ok=%v err=%v", ok, err)
	}
	if got != want {
		t.Fatalf("round-trip mismatch: got %+v want %+v", got, want)
	}
}

// TestWritePin_DoesNotFollowSymlink proves the atomic write does not clobber a
// symlink target: a plain os.WriteFile at a symlinked identity.json followed the
// link and wrote through to the victim; the canonical temp+rename replaces the
// symlink itself, leaving the target untouched.
func TestWritePin_DoesNotFollowSymlink(t *testing.T) {
	dir := t.TempDir()
	victim := filepath.Join(dir, "victim.txt")
	const sentinel = "do-not-touch"
	if err := os.WriteFile(victim, []byte(sentinel), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(dir, ".abcd", "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, filepath.Join(cfgDir, "identity.json")); err != nil {
		t.Skipf("cannot symlink: %v", err)
	}
	if err := WritePin(dir, Pin{Name: "Alex", Email: "alex@example.com"}); err != nil {
		t.Fatalf("WritePin: %v", err)
	}
	got, err := os.ReadFile(victim)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != sentinel {
		t.Errorf("WritePin followed the symlink and clobbered the target: %q", got)
	}
}

func TestWritePin_RequiresBothFields(t *testing.T) {
	dir := t.TempDir()
	if err := WritePin(dir, Pin{Name: "Alex"}); err == nil {
		t.Fatal("want error when email is missing")
	}
}

// The self-contained pre-commit hook reads the pin with a naive sed, so the
// stored value must round-trip through it (iss-63). WritePin refuses the
// characters JSON must always escape (double-quote, backslash, control), which
// the sed cannot read back, and stores &, <, > literally (no HTML escaping) so
// a legal git identity like "Marks & Spencer" is not fail-closed.
func TestWritePin_RejectsUnpinnableChars(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []Pin{
		{Name: `Alex "the dev"`, Email: "alex@example.com"}, // double-quote
		{Name: `back\slash`, Email: "alex@example.com"},     // backslash
		{Name: "line\nbreak", Email: "alex@example.com"},    // control char
		{Name: "Alex", Email: `a"x@example.com`},            // quote in email
	} {
		if err := WritePin(dir, p); err == nil {
			t.Fatalf("want error for unpinnable pin %+v", p)
		}
	}
}

func TestWritePin_LegalIdentitiesRoundTrip(t *testing.T) {
	for _, want := range []Pin{
		{Name: "Marks & Spencer", Email: "ci@m-and-s.example"}, // & is legal, must not HTML-escape
		{Name: "O'Brien", Email: "obrien@example.com"},         // apostrophe
		{Name: "José <team>", Email: "jose@example.com"},       // unicode + < >
	} {
		dir := t.TempDir()
		if err := WritePin(dir, want); err != nil {
			t.Fatalf("legal identity %+v must write: %v", want, err)
		}
		got, ok, err := LoadPin(dir)
		if err != nil || !ok || got != want {
			t.Fatalf("round-trip %+v: got %+v ok=%v err=%v", want, got, ok, err)
		}
		// With HTML escaping off, the raw pin carries &, <, > literally; the bug
		// (escaping on) would store the \uXXXX form instead, so the literal char
		// would be ABSENT from the file — which the hook's sed cannot read back.
		raw, err := os.ReadFile(filepath.Join(dir, PinRelPath))
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range []rune{'&', '<', '>'} {
			if strings.ContainsRune(want.Name, r) && !strings.ContainsRune(string(raw), r) {
				t.Fatalf("pin did not store %q literally (HTML-escaped?); the hook sed cannot read it back:\n%s", r, raw)
			}
		}
	}
}

// Blocks reports whether a pre-commit hook should refuse the commit: a mismatch
// or an unset identity blocks; OK and NoPin (opted-out) do not.
func TestBlocks(t *testing.T) {
	cases := map[Status]bool{
		StatusOK:       false,
		StatusNoPin:    false,
		StatusMismatch: true,
		StatusUnset:    true,
	}
	for s, want := range cases {
		if got := (Result{Status: s}).Blocks(); got != want {
			t.Errorf("Status %v: Blocks()=%v, want %v", s, got, want)
		}
	}
}

// TestLoadPinReadsProductionModeDefault proves the itd-91 attribution seam is
// EXTENDED rather than duplicated: the repo's default production mode is an
// optional member of the pin the repo already commits, read by the one reader
// that already exists (spc-56). A second config file would be a second reader.
func TestLoadPinReadsProductionModeDefault(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	writePin(t, dir, `{"name":"A Maintainer","email":"maintainer@example.com","production_mode":"dictated-and-formatted"}`)
	pin, ok, err := LoadPin(dir)
	if err != nil || !ok {
		t.Fatalf("LoadPin: ok=%v err=%v", ok, err)
	}
	if pin.ProductionMode != "dictated-and-formatted" {
		t.Fatalf("pin production_mode = %q, want dictated-and-formatted", pin.ProductionMode)
	}
	got, err := DeclaredProductionMode(dir)
	if err != nil {
		t.Fatalf("DeclaredProductionMode: %v", err)
	}
	if string(got) != "dictated-and-formatted" {
		t.Fatalf("DeclaredProductionMode = %q, want dictated-and-formatted", got)
	}
}

// TestLoadPinRefusesUnknownProductionMode proves the member is validated at the
// boundary exactly as a malformed pin already is: a value outside the closed set
// is an error, never a silently-ignored member that would let every record the
// repo mints carry a mode nothing can read.
func TestLoadPinRefusesUnknownProductionMode(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	writePin(t, dir, `{"name":"A Maintainer","email":"maintainer@example.com","production_mode":"typed"}`)
	if _, _, err := LoadPin(dir); err == nil {
		t.Fatal("want an error on an out-of-vocabulary production_mode, got nil")
	}
	if _, err := DeclaredProductionMode(dir); err == nil {
		t.Fatal("DeclaredProductionMode: want an error on an out-of-vocabulary pin, got nil")
	}
}

// TestLoadPinAbsentMemberDefaultsHandWritten proves the member is optional in
// both directions: a pin written before it existed loads unchanged, and a repo
// with no pin at all still has a default. An absent member means hand-written.
func TestLoadPinAbsentMemberDefaultsHandWritten(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	writePin(t, dir, `{"name":"A Maintainer","email":"maintainer@example.com"}`)
	pin, ok, err := LoadPin(dir)
	if err != nil || !ok {
		t.Fatalf("LoadPin: ok=%v err=%v", ok, err)
	}
	if pin.ProductionMode != "" {
		t.Fatalf("pin production_mode = %q, want the member absent", pin.ProductionMode)
	}
	got, err := DeclaredProductionMode(dir)
	if err != nil || got != provenance.DefaultMode {
		t.Fatalf("DeclaredProductionMode with an absent member = %q, %v; want %q", got, err, provenance.DefaultMode)
	}
	// A repo that has not adopted the identity gate at all still has a default.
	unpinned := t.TempDir()
	got, err = DeclaredProductionMode(unpinned)
	if err != nil || got != provenance.DefaultMode {
		t.Fatalf("DeclaredProductionMode with no pin = %q, %v; want %q", got, err, provenance.DefaultMode)
	}
}
