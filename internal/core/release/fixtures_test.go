package release

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// composerFixture is the shape of a release-changelog-composer fixture: the
// records a cut is built from, and the payload a well-behaved composer emits.
type composerFixture struct {
	Fixture string `json:"fixture"`
	Input   struct {
		Cut struct {
			NextTag string `json:"next_tag"`
			Added   []struct {
				ID             string `json:"id"`
				InPressRelease bool   `json:"in_press_release"`
			} `json:"added"`
		} `json:"cut"`
		Records []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"records"`
	} `json:"input"`
	Expected struct {
		MustNotContain []string        `json:"must_not_contain"`
		Example        json.RawMessage `json:"emitted_payload_example"`
	} `json:"expected"`
}

// TestComposerFixtureExamplesIngest runs every composer fixture's expected
// payload through the real ingest, against a repository built from the
// fixture's own records. It proves each expected output is VALID (the binary
// accepts it, so the fixture teaches a shape that lands) and CLEAN (neither the
// changelog nor the release page carries any of the fixture's forbidden strings:
// the injection canary's control text, the no-forecast fixture's dates and
// promises). Whether a given model obeys the prompt is shown when a host runs
// the fixture; this pins the answer it is measured against.
func TestComposerFixtureExamplesIngest(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := fsutil.ModuleRoot(cwd)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(root, "agents", "release-changelog-composer", "fixtures", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	names := make([]string, 0, len(paths))
	for _, p := range paths {
		names = append(names, filepath.Base(p))
	}
	for _, want := range []string{"injection-canary.json", "injection-canary-press-release.json", "no-forecast.json"} {
		if !strings.Contains(strings.Join(names, " "), want) {
			t.Errorf("the composer ships no %s fixture", want)
		}
	}

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var fx composerFixture
			if err := json.Unmarshal(data, &fx); err != nil {
				t.Fatalf("the fixture does not parse: %v", err)
			}
			if len(fx.Expected.MustNotContain) == 0 {
				t.Error("the fixture names nothing the output must not contain")
			}

			r := releasedRepo(t)
			r.Write("CHANGELOG.md", baseChangelog)
			for _, rec := range fx.Input.Records {
				r.Write(rec.Path, rec.Content)
			}
			r.Commit("the fixture's records")

			// The fixture's cut is what the emit step derives from its records.
			cut := emit(t, r)
			derived := map[string]bool{}
			for _, e := range cut.Added {
				derived[e.ID] = e.InPressRelease
			}
			for _, e := range fx.Input.Cut.Added {
				if got, ok := derived[e.ID]; !ok || got != e.InPressRelease {
					t.Errorf("fixture cut entry %s (in_press_release=%v) is not what the emit step derives (%v, present=%v)",
						e.ID, e.InPressRelease, got, ok)
				}
			}
			if cut.NextTag != fx.Input.Cut.NextTag {
				t.Errorf("fixture next_tag %q, emit derives %q", fx.Input.Cut.NextTag, cut.NextTag)
			}

			res, err := Ingest(r.Root(), liveSurface(), fx.Expected.Example, cutAt)
			if err != nil {
				t.Fatalf("the expected payload is refused: %v", err)
			}
			if !res.Written {
				t.Fatalf("nothing was written; refusals = %v", refusalKinds(res.Cut))
			}
			written := readChangelog(t, r.Root())
			if res.Page.Written {
				written += readPage(t, r.Root())
			}
			for _, bad := range fx.Expected.MustNotContain {
				if strings.Contains(strings.ToLower(written), strings.ToLower(bad)) {
					t.Errorf("the written release carries the forbidden %q", bad)
				}
			}
		})
	}
}
