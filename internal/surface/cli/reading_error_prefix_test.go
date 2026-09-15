package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// One rule for the whole reading surface: the front door names the sub-verb, and
// the core's own package tag is never printed a second time behind it
// (iss-2608311145286014). The failure it detects is what an operator sees first
// on a refusal — `abcd: reading: reading: …` — so the assertion is made on the
// rendered line, on both output planes, for every sub-verb rather than the one
// that was noticed.
//
// The refusals below all come from the CORE, deliberately: a surface-authored
// message carries no package tag and could never double one, so a table built
// from those would pass whatever the front door did with a core error.

// assertNoDoubledReadingTag holds the rule on one rendered message: it opens with
// the sub-verb's tag exactly once, and what follows it does not open with the
// core package's own tag.
func assertNoDoubledReadingTag(t *testing.T, verb, msg string) {
	t.Helper()
	tag := verb + ": "
	if !strings.HasPrefix(msg, tag) {
		t.Fatalf("`abcd %s` refusal does not open with %q:\n%s", verb, tag, msg)
	}
	if rest := strings.TrimPrefix(msg, tag); strings.HasPrefix(rest, "reading: ") {
		t.Fatalf("`abcd %s` doubles the reading tag; an operator reads %q:\nabcd: %s",
			verb, tag+"reading: ", msg)
	}
	if strings.Contains(msg, "reading: reading: ") {
		t.Fatalf("`abcd %s` renders a doubled reading tag:\nabcd: %s", verb, msg)
	}
}

// readingRefusalRepo builds the smallest tree each sub-verb can be driven into a
// CORE refusal from, and returns the argument list that does it.
func readingRefusalRepo(t *testing.T, verb string) (root string, args []string) {
	t.Helper()
	switch verb {
	case "reading":
		// The status render resolves the definitions the ingest verb would
		// resolve, so a definition present but silent about its position is a
		// fault the locator reports — with its package tag on the front.
		root = t.TempDir()
		def := filepath.Join(root, "agents", "cold-reading-widening.md")
		if err := os.MkdirAll(filepath.Dir(def), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(def, []byte("---\nregime: generative\n---\n\n# Widening\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return root, []string{"reading"}
	case "reading assemble":
		// The comparative position refuses to assemble: its object is not
		// repository material (itd-199, adr-58).
		return readingRepo(t), []string{
			"reading", "assemble",
			"--position", "comparative", "--target", "HEAD",
		}
	case "reading ingest":
		root = t.TempDir()
		return root, []string{
			"reading", "ingest", "--reading-json", filepath.Join(root, "no-such-output.json"),
		}
	}
	t.Fatalf("no refusal fixture for %q", verb)
	return "", nil
}

// TestReadingSurfaceNeverDoublesItsErrorPrefix drives every reading sub-verb into
// a core refusal and holds the rendered line to one tag, on the text plane and in
// the --json envelope alike.
func TestReadingSurfaceNeverDoublesItsErrorPrefix(t *testing.T) {
	for _, verb := range []string{"reading", "reading assemble", "reading ingest"} {
		t.Run(verb, func(t *testing.T) {
			t.Run("text", func(t *testing.T) {
				root, args := readingRefusalRepo(t, verb)
				t.Chdir(root)
				var stdout, stderr bytes.Buffer
				if code := Run(args, &stdout, &stderr); code == 0 {
					t.Fatalf("`abcd %s` was expected to refuse; stdout=%q stderr=%q",
						verb, stdout.String(), stderr.String())
				}
				line := strings.TrimSpace(stderr.String())
				if !strings.HasPrefix(line, "abcd: ") {
					t.Fatalf("`abcd %s` refusal is not the single diagnostic line:\n%s", verb, line)
				}
				assertNoDoubledReadingTag(t, verb, strings.TrimPrefix(line, "abcd: "))
			})
			t.Run("json", func(t *testing.T) {
				root, args := readingRefusalRepo(t, verb)
				t.Chdir(root)
				var stdout, stderr bytes.Buffer
				if code := Run(append(args, "--json"), &stdout, &stderr); code == 0 {
					t.Fatalf("`abcd %s --json` was expected to refuse; stdout=%q stderr=%q",
						verb, stdout.String(), stderr.String())
				}
				var env struct {
					Error string `json:"error"`
				}
				if err := json.Unmarshal(stderr.Bytes(), &env); err != nil {
					t.Fatalf("`abcd %s --json` refusal is not a JSON envelope: %v\nstderr: %q",
						verb, err, stderr.String())
				}
				assertNoDoubledReadingTag(t, verb, env.Error)
			})
		})
	}
}
