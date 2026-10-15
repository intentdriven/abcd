//go:build smoke

package evals

// The session page smoke (spc-2610141709430146): `abcd session page --out`
// runs for real through the built binary into a scratch directory, and the
// test asserts the six files it reports are the ones on disk, the page loads
// its own script, and a base path moves the page's asset references under it.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestSessionPageWritesItsFiles(t *testing.T) {
	want := []string{"404.html", "index.html", "session.css", "session.js", "vendor/LICENSES.txt", "vendor/collab.js"}
	for _, c := range []struct {
		name, base, script string
	}{
		{"at the root", "", `src="/session.js"`},
		{"under a base path", "abcdesign", `src="/abcdesign/session.js"`},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "page")
			args := []string{"session", "page", "--out", dir, "--json"}
			if c.base != "" {
				args = append(args, "--base-path", c.base)
			}
			out, code := runIn(t, t.TempDir(), nil, args...)
			if panicked(out) || code != 0 {
				t.Fatalf("`abcd %s` exit=%d\n%s", strings.Join(args, " "), code, out)
			}
			var res struct {
				Files []string `json:"files"`
			}
			if err := json.Unmarshal([]byte(out[strings.Index(out, "{"):]), &res); err != nil {
				t.Fatalf("--json: %v\n%s", err, out)
			}
			got := append([]string(nil), res.Files...)
			sort.Strings(got)
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Fatalf("reported %v, want %v", got, want)
			}
			for _, f := range want {
				info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(f)))
				if err != nil || info.Size() == 0 {
					t.Fatalf("%s: %v (or empty)", f, err)
				}
			}
			index, err := os.ReadFile(filepath.Join(dir, "index.html"))
			if err != nil || !strings.Contains(string(index), c.script) {
				t.Fatalf("index.html does not load %s: %v", c.script, err)
			}
			notFound, err := os.ReadFile(filepath.Join(dir, "404.html"))
			if err != nil || string(notFound) != string(index) {
				t.Fatal("404.html is not the index")
			}
		})
	}
}
