package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// A bundle of one is today's drain (itd-2611201450052326 criterion 8,
// spc-2611201455051897 step 4): with a cap of 1, and with a lead that has no
// candidates under a cap above 1, the drain's first two moves leave the
// drain's state, the run's state, the judgement request and the moves' next
// lines as the drain before bundles left them. The fixtures under
// testdata/drain_bundle_one were taken from the default branch before step 4
// (b8250514c) with ABCD_UPDATE_BUNDLE_ONE=1. The comparison masks the schema
// versions and the drain state's bundle_max, and, as every run differs in
// them, the times, the run ids, the commit names and the fixture's paths.

// bundleOneMasks are the masks every run needs, in order.
var bundleOneMasks = []struct {
	re   *regexp.Regexp
	with string
}{
	{regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})`), "<TIME>"},
	{regexp.MustCompile(`run-\d{16}`), "<RUN>"},
	{regexp.MustCompile(`\b[0-9a-f]{40}\b`), "<SHA>"},
	{regexp.MustCompile(`\(([0-9a-f]{7,12})\)`), "(<SHORT>)"},
	{regexp.MustCompile(`\bh_[0-9a-f]+\b`), "<HOLDER>"},
}

// bundleOneJSON is a state file read as JSON with its schema version, and
// the drain state's bundle_max, masked, written back in a stable form.
func bundleOneJSON(t *testing.T, file string) string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	doc["schema_version"] = "<VERSION>"
	delete(doc, "bundle_max")
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// bundleOneSnapshot is what the two moves left, masked.
func bundleOneSnapshot(t *testing.T, repo *gittest.Repo, request string, first, second DrainResult) string {
	t.Helper()
	if second.Start == nil {
		t.Fatalf("the second move opens the lane: %+v", second)
	}
	parts := []string{
		"== first move's next\n" + first.Next,
		"== judgement request\n" + request,
		"== second move's next\n" + second.Next,
		"== drain state\n" + bundleOneJSON(t, filepath.Join(repo.Root(), filepath.FromSlash(DrainStateRel))),
		"== run state\n" + bundleOneJSON(t, filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(second.Start.RunID)))),
	}
	s := strings.Join(parts, "\n") + "\n"
	for _, p := range []string{repo.Root(), os.Getenv("HOME")} {
		if p != "" {
			s = strings.ReplaceAll(s, p, "<PATH>")
		}
	}
	for _, m := range bundleOneMasks {
		s = m.re.ReplaceAllString(s, m.with)
	}
	return s
}

// TestABundleOfOneIsTodaysDrain is criterion 8.
func TestABundleOfOneIsTodaysDrain(t *testing.T) {
	for _, c := range []struct{ name, config string }{
		{"cap-of-1", ""},
		{"no-candidates", `{"drain":{"bundle_max":4}}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			// The documentation issue leads, and the fixture's other issue
			// is a bug: no candidate joins it whatever the cap.
			repo, _ := reservingDrainRepo(t, "commands/drain.md")
			if c.config != "" {
				writeConfigAt(t, abcdhome.Path(os.Getenv("HOME"), "config.json"), c.config)
			}
			first, err := Drain(repo.Root(), Options{}, DrainOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if first.Judging == nil {
				t.Fatalf("the first move asks a judgement: %+v", first)
			}
			request, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(first.Judging.Request)))
			if err != nil {
				t.Fatal(err)
			}
			second := judge(t, repo, Options{}, first, JudgementNo, "", "the remedy changes neither")
			got := bundleOneSnapshot(t, repo, string(request), first, second)
			golden := filepath.Join("testdata", "drain_bundle_one", c.name+".golden")
			if os.Getenv("ABCD_UPDATE_BUNDLE_ONE") != "" {
				if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("the drain with %s differs from the drain before bundles:\n--- got\n%s\n--- want\n%s", c.name, got, want)
			}
		})
	}
}
