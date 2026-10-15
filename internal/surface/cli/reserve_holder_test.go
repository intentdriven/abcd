package cli

import (
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// `abcd reserve --holder` (iss-2610152046379890): the account a reservation
// act from this checkout would be made as, and which source named it. It
// reserves nothing, writes nothing and pushes nothing.

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
	calls := fakeForge(t, "octo-person", 4242, nil)

	code, stdout, stderr := reserveRun(t, p, "reserve", "--holder")
	if code != 0 || stdout != "would reserve as octo (from git)\n" || stderr != "" {
		t.Fatalf("exit %d, stdout %q, stderr %q; want the git holder", code, stdout, stderr)
	}
	code, stdout, stderr = reserveRun(t, p, "reserve", "--holder", "--json")
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); code != 0 || err != nil {
		t.Fatalf("--json: exit %d, %v\n%s%s", code, err, stdout, stderr)
	}
	want := map[string]any{"login": "octo", "id": float64(1234), "noreply": octoMail, "source": "git"}
	if len(got) != len(want) {
		t.Fatalf("--json = %v; want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("--json = %v; want %v", got, want)
		}
	}
	if calls.n != 0 {
		t.Fatalf("a noreply user.email asked the forge %d times", calls.n)
	}

	code, stdout, stderr = reserveRun(t, pat, "reserve", "--holder")
	if code != 0 || stdout != "would reserve as octo-person (from token)\n" || calls.n != 1 {
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
