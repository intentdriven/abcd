package credential

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// bypassPatterns are the lines a reader that bypasses the store writes.
var bypassPatterns = []struct {
	why string
	re  *regexp.Regexp
}{
	{"reads the abcd home directly instead of through Store", regexp.MustCompile(`credential\.(Machine|UserMachine)\(`)},
	{"writes the abcd home directly instead of through Set or Walk", regexp.MustCompile(`credential\.SetMachine\(`)},
	{"names a store file instead of resolving by name", regexp.MustCompile(`"[^"]*(credentials\.json|credential-homes\.json)"`)},
	{"runs a keychain command instead of resolving by name", regexp.MustCompile(`"(find-generic-password|add-generic-password|secret-tool)"`)},
	{"reads a secret-shaped environment variable instead of resolving by name",
		regexp.MustCompile(`os\.(Getenv|LookupEnv)\("[A-Za-z0-9_]*(?i:token|secret|passw|api_?key|_key)[A-Za-z0-9_]*"\)`)},
}

// TestEveryReaderGoesThroughTheStore (criterion 4, adr-2609221017021499
// ruling 3): outside this package, no production code reads a credential any
// way but Store(...).Resolve, writes one any way but Set or Walk, names the
// store's files, runs a keychain command, or reads a secret-shaped
// environment variable. It is a drift grep for an accidental bypass, not an
// evasion gate: a new reader written the obvious way fails here, naming the
// file and the line, while one written to slip past it (an aliased or dot
// import, a variable's name held in a variable, os.Environ, an argv built by
// concatenation) does not, and only cmd/ and internal/ are walked.
func TestEveryReaderGoesThroughTheStore(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("the module root is not where this test expects it: %v", err)
	}
	self := filepath.Join(root, "internal", "core", "credential")
	var scanned int
	for _, top := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if p == self || d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			raw, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			scanned++
			for i, line := range strings.Split(string(raw), "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "//") {
					continue
				}
				for _, b := range bypassPatterns {
					if b.re.MatchString(line) {
						rel, _ := filepath.Rel(root, p)
						t.Errorf("%s:%d %s:\n\t%s", filepath.ToSlash(rel), i+1, b.why, strings.TrimSpace(line))
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if scanned < 100 {
		t.Fatalf("scanned %d Go files; the walk did not reach the tree", scanned)
	}
}

// TestTheReaderGrepCatchesABypass proves the patterns bite: each line is one a
// bypassing reader would write.
func TestTheReaderGrepCatchesABypass(t *testing.T) {
	for _, line := range []string{
		`v, err := credential.Machine(home).Resolve("x")`,
		`credential.SetMachine(home, "x", v)`,
		`p := filepath.Join(home, ".abcd", "credentials.json")`,
		`exec.Command("security", "find-generic-password", "-w")`,
		`tok := os.Getenv("CLOUDFLARE_API_TOKEN")`,
		`k, _ := os.LookupEnv("OPENROUTER_API_KEY")`,
	} {
		hit := false
		for _, b := range bypassPatterns {
			hit = hit || b.re.MatchString(line)
		}
		if !hit {
			t.Errorf("the grep misses a bypass: %s", line)
		}
	}
}
