package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// reflect_lifeboat_cli_test.go is itd-24 criteria 5 and 6 at the front door:
// `disembark pack` carries every retrospective, `embark from` writes them back,
// and `embark lessons` shows the few most like the new voyage's brief with the
// rest as a list.

func retroFixtureDoc(tag string, lessons ...string) string {
	var b strings.Builder
	b.WriteString("---\nrelease: " + tag + "\n---\n\n# Retrospective for " + tag + "\n\n## Lessons learned\n\n")
	for _, l := range lessons {
		b.WriteString("- " + l + "\n")
	}
	return b.String()
}

func retroSourceRepo(t *testing.T) string {
	t.Helper()
	repo := embarkSourceRepo(t)
	for tag, lessons := range map[string][]string{
		"v0.1.0": {"Audit every intent before the release cut.", "Pin the toolchain the continuous integration runs.", "Name the parser's refusals in the changelog."},
		"v0.2.0": {"Keep the lifeboat small because embark reads it whole.", "Write the scanner's fixtures at runtime."},
	} {
		p := filepath.Join(repo, ".abcd", "development", "retrospectives", tag, "README.md")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(retroFixtureDoc(tag, lessons...)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

// TestDisembarkPackCarriesEveryRetrospectiveAndEmbarkWritesThemBack is
// criterion 5, and the write-back that makes criterion 6 reachable.
func TestDisembarkPackCarriesEveryRetrospectiveAndEmbarkWritesThemBack(t *testing.T) {
	repo := retroSourceRepo(t)
	t.Setenv("HOME", t.TempDir())
	dest := filepath.Join(t.TempDir(), "lifeboat")
	runCLI(t, "disembark", "pack", repo, dest, "--json")
	for _, tag := range []string{"v0.1.0", "v0.2.0"} {
		src, _ := os.ReadFile(filepath.Join(repo, ".abcd", "development", "retrospectives", tag, "README.md"))
		got, err := os.ReadFile(filepath.Join(dest, "retrospectives", tag, "README.md"))
		if err != nil || string(got) != string(src) {
			t.Errorf("the lifeboat does not carry the %s retrospective verbatim: %v", tag, err)
		}
	}

	target := t.TempDir()
	out := runCLI(t, "embark", "from", dest, target, "--json")
	var res struct {
		Families map[string]int `json:"families"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if res.Families["retrospectives"] != 2 {
		t.Errorf("families = %v, want two retrospectives", res.Families)
	}
	if _, err := os.Stat(filepath.Join(target, ".abcd", "development", "retrospectives", "v0.2.0", "README.md")); err != nil {
		t.Errorf("embark did not write the retrospective back: %v", err)
	}
}

// TestEmbarkLessonsShowsTheFewMostLikeTheBrief is criterion 6: ranked against
// the new voyage's framing chapter, the top three first and the rest as a list,
// with the question the interview asks.
func TestEmbarkLessonsShowsTheFewMostLikeTheBrief(t *testing.T) {
	repo := retroSourceRepo(t)
	lb := packEmbarkLifeboat(t, repo)
	target := t.TempDir()
	framing := filepath.Join(target, ".abcd", "development", "brief", "01-product", "06-framing.md")
	if err := os.MkdirAll(filepath.Dir(framing), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(framing, []byte("# Framing\n\nThis voyage audits every intent before each release cut and keeps the lifeboat small.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := string(runCLI(t, "embark", "lessons", lb, target))
	for _, want := range []string{"which of these apply?", "1. [v0.1.0] Audit every intent", "the rest (2):"} {
		if !strings.Contains(out, want) {
			t.Errorf("the lessons view does not show %q:\n%s", want, out)
		}
	}

	js := runCLI(t, "embark", "lessons", lb, target, "--json")
	var v struct {
		Top      []struct{ Release, Text string } `json:"top"`
		Rest     []struct{ Release, Text string } `json:"rest"`
		Unranked bool                             `json:"unranked"`
		Framing  string                           `json:"framing_source"`
	}
	if err := json.Unmarshal(js, &v); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, js)
	}
	if len(v.Top) != 3 || len(v.Rest) != 2 || v.Unranked || v.Framing != ".abcd/development/brief/01-product/06-framing.md" {
		t.Errorf("lessons = %+v", v)
	}

	// --brief ranks against the press release the interview is writing, for a
	// target with no framing chapter yet.
	brief := filepath.Join(t.TempDir(), "press-release.md")
	if err := os.WriteFile(brief, []byte("We write the scanner's fixtures at runtime.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out = string(runCLI(t, "embark", "lessons", lb, t.TempDir(), "--brief", brief))
	if !strings.Contains(out, "1. [v0.2.0] Write the scanner's fixtures at runtime.") {
		t.Errorf("--brief did not rank against the named text:\n%s", out)
	}
}
