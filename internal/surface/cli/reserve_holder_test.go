package cli

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
)

// `abcd reserve --holder` (iss-2610152046379890): the account a reservation
// act from this checkout would be made as, and which source named it, and the
// holder id the act pushes in its place (iss-2610290809526033). It reserves
// nothing, pushes nothing, and writes only the holder id when none is kept.

// forgeCalls is a fake of the forge's user endpoint the reserve verb is
// given, counting its calls.
type forgeCalls struct{ n int }

// fakeForge makes the reserve verb's user-endpoint call answer login and id,
// or err, for the rest of the test.
func fakeForge(t *testing.T, login string, id int64, err error) *forgeCalls {
	t.Helper()
	c := &forgeCalls{}
	orig := reserveForgeUser
	reserveForgeUser = func(string) (string, int64, error) { c.n++; return login, id, err }
	t.Cleanup(func() { reserveForgeUser = orig })
	return c
}

func TestReserveHolderPlainAndJSON(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	pat := w.clone(t, "pat", "pat@example.com")
	// The same person's second checkout on this machine, whose git identity
	// is not a no-reply address: it keeps its holder id in the same home.
	cloneHomes[pat] = w.home
	calls := fakeForge(t, "octo-person", 4242, nil)

	code, stdout, stderr := reserveRunRaw(t, p, "reserve", "--holder")
	kept, err := os.ReadFile(abcdhome.Path(w.home, "holder-id"))
	id := strings.TrimSpace(string(kept))
	if err != nil || !regexp.MustCompile(`^h_[0-9a-f]{32}$`).MatchString(id) {
		t.Fatalf("--holder kept %q, %v; want a minted holder id", kept, err)
	}
	idLine := "holder id: " + id + " (kept in " + abcdhome.Display("holder-id") + "); no trailer and no branch name abcd pushes carries your login; " +
		"each commit's author and committer carry your git identity when it is a person's, otherwise the token account's login and no-reply address, " +
		"and holds already pushed are not rewritten\n"
	if code != 0 || stdout != "would reserve as octo (from git)\n"+idLine || stderr != "abcd: "+claim.MintNoticeLineForTest(id)+"\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q; want the git holder and the holder id, and the mint told once", code, stdout, stderr)
	}
	code, stdout, stderr = reserveRunRaw(t, p, "reserve", "--holder", "--json")
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); code != 0 || err != nil || stderr != "" {
		t.Fatalf("--json: exit %d, %v\n%s%s; want the kept id read and no second mint told", code, err, stdout, stderr)
	}
	want := map[string]any{"login": "octo", "id": float64(1234), "noreply": octoMail, "source": "git",
		"holder_id": id, "holder_id_from": abcdhome.Display("holder-id"), "holder_id_file": abcdhome.Display("holder-id"),
		"declared_holder_ids_file": abcdhome.Display("holder-ids")}
	if len(got) != len(want)+1 {
		t.Fatalf("--json = %v; want %v and declared_holder_ids", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("--json = %v; want %v", got, want)
		}
	}
	if d, ok := got["declared_holder_ids"].([]any); !ok || len(d) != 0 {
		t.Fatalf("--json declared_holder_ids = %v; want []", got["declared_holder_ids"])
	}
	if calls.n != 0 {
		t.Fatalf("a noreply user.email asked the forge %d times", calls.n)
	}

	other := "h_0123456789abcdef0123456789abcdef"
	if err := os.WriteFile(abcdhome.Path(w.home, "holder-ids"), []byte(other+" # the laptop\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = reserveRun(t, pat, "reserve", "--holder")
	if code != 0 || stdout != "would reserve as octo-person (from token)\n"+idLine+
		"also yours: "+other+" (declared in "+abcdhome.Display("holder-ids")+")\n" || calls.n != 1 {
		t.Fatalf("exit %d, stdout %q, stderr %q after %d calls; want the token's holder after one", code, stdout, stderr, calls.n)
	}
}

func TestReserveHolderRefusesOtherArgs(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	calls := fakeForge(t, "octo-person", 4242, nil)
	for _, args := range [][]string{
		{"reserve", "--holder", "iss-5"},
		{"reserve", "--holder", "--footprint", "a.go"},
		{"reserve", "--holder", "--release"},
		{"reserve", "--holder", "--take-over"},
		{"reserve", "--holder", "--lease", "1h"},
	} {
		code, stdout, stderr := reserveRun(t, p, args...)
		if code != 2 || stdout != "" || !strings.Contains(stderr, "--holder") {
			t.Fatalf("%v: exit %d, stdout %q, stderr %q; want exit 2 naming --holder", args, code, stdout, stderr)
		}
	}
	if calls.n != 0 {
		t.Fatalf("a refused --holder asked the forge %d times", calls.n)
	}
	// The token cannot answer: a refusal at exit 2.
	pat := w.clone(t, "pat", "pat@example.com")
	fakeForge(t, "", 0, errors.New("gh: not logged in"))
	code, stdout, stderr := reserveRun(t, pat, "reserve", "--holder")
	if code != 2 || stdout != "" || !strings.Contains(stderr, "could not work out whose GitHub account this session uses") {
		t.Fatalf("a failed lookup: exit %d, stdout %q, stderr %q; want exit 2", code, stdout, stderr)
	}
}

func TestReserveHolderWritesNothing(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", "pat@example.com")
	fakeForge(t, "octo-person", 4242, nil)
	snapshot := func() string {
		var b strings.Builder
		for _, dir := range []string{p, w.origin} {
			b.WriteString(gitCmd(t, dir, "for-each-ref", "--format=%(refname) %(objectname)") + "\n")
			err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				fi, err := d.Info()
				if err != nil {
					return err
				}
				b.WriteString(path + " " + fi.Mode().String() + " " + fi.ModTime().Format(time.RFC3339Nano))
				if !d.IsDir() {
					b.WriteString(" " + strconv.FormatInt(fi.Size(), 10))
				}
				b.WriteString("\n")
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		return b.String()
	}
	before := snapshot()
	for _, args := range [][]string{{"reserve", "--holder"}, {"reserve", "--holder", "--json"}} {
		if code, stdout, stderr := reserveRun(t, p, args...); code != 0 {
			t.Fatalf("%v: exit %d\n%s%s", args, code, stdout, stderr)
		}
	}
	if after := snapshot(); after != before {
		t.Fatalf("--holder changed a ref or a file:\nbefore\n%s\nafter\n%s", before, after)
	}
}

// TestReserveHolderRealForgeClasses runs the real user-endpoint call against a
// gh stub, so the classification the refusal names is the one the code makes:
// gh's own words never reach the output, an identity on its stderr included.
func TestReserveHolderRealForgeClasses(t *testing.T) {
	const prefix = "could not work out whose GitHub account this session uses: "
	const leak = "HTTP 401: Bad credentials for Pat Example <pat@example.com> as octo-secret"
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "pat@example.com")
	path := os.Getenv("PATH")
	for _, c := range []struct {
		name, body, class string
	}{
		{"gh fails, naming an identity", "printf '%s\\n' '" + leak + "' >&2\nexit 1\n",
			"gh could not read the account (it is not logged in to github.com, holds no usable token, or GitHub was unreachable)"},
		{"not JSON", "printf 'not json'\n", "GitHub's answer did not name an account"},
		{"an id past int64", `printf '{"login":"x","id":99999999999999999999}'` + "\n", "GitHub's answer did not name an account"},
		{"a negative id", `printf '{"login":"x","id":-1}'` + "\n", "GitHub's answer did not name an account"},
		{"a zero id", `printf '{"login":"x","id":0}'` + "\n", "GitHub's answer did not name an account"},
	} {
		t.Setenv("PATH", path)
		calls := stubGHScript(t, c.body)
		code, stdout, stderr := reserveRun(t, pat, "reserve", "--holder")
		if code != 2 || stdout != "" || !strings.Contains(stderr, prefix+c.class+"; nothing was sent or pushed") {
			t.Fatalf("%s: exit %d, stdout %q, stderr %q; want the class %q", c.name, code, stdout, stderr, c.class)
		}
		for _, s := range []string{leak, "Bad credentials", "octo-secret", "pat@example.com", "Pat Example"} {
			if strings.Contains(stdout+stderr, s) {
				t.Fatalf("%s: the output carries %q: %s", c.name, s, stderr)
			}
		}
		if got := calls(); len(got) != 1 || got[0] != "api --hostname github.com user" {
			t.Fatalf("%s: gh was asked %q; want the user endpoint once", c.name, got)
		}
	}

	// gh missing from PATH: a directory holding git alone.
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	only := t.TempDir()
	if err := os.Symlink(git, filepath.Join(only, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", only)
	code, stdout, stderr := reserveRun(t, pat, "reserve", "--holder")
	if want := prefix + "the GitHub CLI (gh) is not on PATH; nothing was sent or pushed"; code != 2 || stdout != "" || !strings.Contains(stderr, want) {
		t.Fatalf("gh missing: exit %d, stdout %q, stderr %q; want %q", code, stdout, stderr, want)
	}
}

// TestReserveHolderNamesTheEnvironmentsID: with ABCD_HOLDER_ID set, --holder
// prints that id and says it came from there, mints nothing and tells no
// mint; set to anything but an id, it refuses, naming the variable and not
// its value.
func TestReserveHolderNamesTheEnvironmentsID(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	env := "h_00112233445566778899aabbccddeeff"
	t.Setenv(claim.HolderIDEnv, env)
	code, stdout, stderr := reserveRunRaw(t, p, "reserve", "--holder")
	if code != 0 || !strings.Contains(stdout, "holder id: "+env+" (from ABCD_HOLDER_ID); no trailer and no branch name") || stderr != "" {
		t.Fatalf("exit %d, stdout %q, stderr %q; want the variable's id, from the variable, and no mint told", code, stdout, stderr)
	}
	if _, err := os.Lstat(abcdhome.Path(w.home, "holder-id")); !os.IsNotExist(err) {
		t.Fatalf("--holder under the variable wrote the home's holder-id: %v", err)
	}
	code, stdout, _ = reserveRun(t, p, "reserve", "--holder", "--json")
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); code != 0 || err != nil || got["holder_id"] != env || got["holder_id_from"] != "ABCD_HOLDER_ID" {
		t.Fatalf("--json: exit %d, %v, %v; want the variable's id from ABCD_HOLDER_ID", code, err, got)
	}
	t.Setenv(claim.HolderIDEnv, "not-an-id")
	code, stdout, stderr = reserveRun(t, p, "reserve", "--holder")
	if code != 2 || stdout != "" || !strings.Contains(stderr, "ABCD_HOLDER_ID is set but is not a holder id") || strings.Contains(stderr, "not-an-id") {
		t.Fatalf("a malformed variable: exit %d, stdout %q, stderr %q; want exit 2 naming the variable only", code, stdout, stderr)
	}
}
