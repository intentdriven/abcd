package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// launch_preview_test.go — step 3 of spc-2610100613109045: `abcd changelog`
// merges into launch as a preview (decision 5). `abcd launch --dry-run` renders
// the deterministic cut beside its bundle report, a refused cut is information
// that changes no exit code the dry run gives, and the changelog verb is gone
// with no alias (adr-40).

// previewFixture is shipFixture with the launch configuration a plugin's dry
// run reads, and no record shipped since the tag: each test adds the records
// its cut is made of.
func previewFixture(t *testing.T) *gittest.Repo {
	t.Helper()
	r := shipFixture(t)
	r.Write(".abcd/config/artefact.json", `{"kind": "plugin"}`+"\n")
	r.Write(".abcd/config/version-location.json",
		`{"manifest_path": ".claude-plugin/plugin.json", "json_pointer": "/version"}`+"\n")
	r.Write(".abcd/config/launch-payload.json",
		`{"includes": [".claude-plugin", "CHANGELOG.md"]}`+"\n")
	r.Commit("the release configuration")
	refreshSurface(t, r)
	return r
}

// cutHeader is the first line of the cut the dry run renders.
const cutHeader = "abcd launch --dry-run — "

// cutSection is the cut half of a plain dry run: everything from the cut's own
// header line on, so an assertion about the cut cannot be met by a line of the
// bundle report above it.
func cutSection(t *testing.T, out []byte) string {
	t.Helper()
	i := strings.Index(string(out), cutHeader)
	if i < 0 {
		t.Fatalf("the dry run renders no cut (no %q line):\n%s", cutHeader, out)
	}
	return string(out)[i:]
}

// dryRunCut is the slice of the dry run's JSON these tests read: the bundle
// report's version beside the cut.
type dryRunCut struct {
	Version string `json:"version"`
	Cut     *struct {
		Ready    bool   `json:"ready"`
		NextTag  string `json:"next_tag"`
		Impact   string `json:"impact"`
		Guard    any    `json:"guard"`
		Refusals []struct {
			Kind   string `json:"kind"`
			Reason string `json:"reason"`
		} `json:"refusals"`
		Added []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"added"`
	} `json:"cut"`
}

func decodeDryRunCut(t *testing.T, out []byte) dryRunCut {
	t.Helper()
	var got dryRunCut
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("launch --dry-run --json is not JSON: %v\n%s", err, out)
	}
	if got.Cut == nil {
		t.Fatalf("launch --dry-run --json carries no cut:\n%s", out)
	}
	return got
}

// TestLaunchDryRunRendersTheCut is A10's preview half: the dry run renders the
// cut the release would make, its derived version, its records and its guard
// verdict, in text and in --json, beside the bundle report; and a refused cut
// is reported as information, exit 0, as the dry run's exit is today.
func TestLaunchDryRunRendersTheCut(t *testing.T) {
	t.Run("ready", func(t *testing.T) {
		r := previewFixture(t)
		r.Write(".abcd/development/intents/shipped/itd-73-derived-versioning.md",
			"---\nid: itd-73\nimpact: additive\n---\n\n# A Version Is A Fact\n\nthe version is derived.\n")
		r.Commit("ship an intent")

		out, err := shipIn(t, r, "launch", "--dry-run")
		if code := exitCodeOf(err); code != 0 {
			t.Fatalf("exit = %d, want 0\n%s\n%v", code, out, err)
		}
		if !strings.HasPrefix(string(out), "abcd launch (dry-run) — version ") {
			t.Errorf("the bundle report no longer leads the dry run:\n%s", out)
		}
		cut := cutSection(t, out)
		for _, want := range []string{
			cutHeader + "v0.4.0 -> v0.4.1 (additive)",
			"guard:      ",
			"itd-73", "A Version Is A Fact",
		} {
			if !strings.Contains(cut, want) {
				t.Errorf("the dry run's cut does not mention %q:\n%s", want, out)
			}
		}

		js, err := shipIn(t, r, "launch", "--dry-run", "--json")
		if code := exitCodeOf(err); code != 0 {
			t.Fatalf("--json exit = %d, want 0\n%s", code, js)
		}
		got := decodeDryRunCut(t, js)
		if got.Version == "" {
			t.Errorf("the bundle report's fields left the JSON:\n%s", js)
		}
		if !got.Cut.Ready || got.Cut.NextTag != "v0.4.1" || got.Cut.Impact != "additive" || got.Cut.Guard == nil {
			t.Errorf("cut = %+v, want a ready v0.4.1 additive cut with its guard", *got.Cut)
		}
		if len(got.Cut.Added) != 1 || got.Cut.Added[0].ID != "itd-73" {
			t.Errorf("cut.added = %+v, want itd-73", got.Cut.Added)
		}
	})

	t.Run("refused", func(t *testing.T) {
		r := previewFixture(t)

		out, err := shipIn(t, r, "launch", "--dry-run")
		if code := exitCodeOf(err); code != 0 {
			t.Fatalf("a refused cut changed the dry run's exit: %d, want 0\n%s\n%v", code, out, err)
		}
		cut := cutSection(t, out)
		for _, want := range []string{cutHeader + "REFUSED", "empty-cut"} {
			if !strings.Contains(cut, want) {
				t.Errorf("the dry run's cut does not mention %q:\n%s", want, out)
			}
		}

		js, err := shipIn(t, r, "launch", "--dry-run", "--json")
		if code := exitCodeOf(err); code != 0 {
			t.Fatalf("--json exit = %d, want 0\n%s", code, js)
		}
		if got := decodeDryRunCut(t, js); got.Cut.Ready || len(got.Cut.Refusals) == 0 {
			t.Errorf("cut = %+v, want a refused cut with its refusal", *got.Cut)
		}
	})
}

// TestChangelogIsAnUnknownCommand is A10's removal half: with the preview in
// launch, `abcd changelog` is an unknown command (no alias, adr-40), and its
// note names what answers instead rather than calling the binary stale for not
// knowing it.
func TestChangelogIsAnUnknownCommand(t *testing.T) {
	root := stalePluginRoot(t)
	setExecutable(t, filepath.Join(root, "abcd"))
	code, stdout, stderr := runMain(t, "changelog")
	if code != 2 || stdout != "" {
		t.Fatalf("`abcd changelog` exit = %d, stdout %q; want exit 2 and an empty stdout", code, stdout)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("`abcd changelog` is not refused as an unknown command:\n%s", stderr)
	}
	if !strings.Contains(stderr, "abcd launch --dry-run") {
		t.Errorf("the refusal does not name the preview that replaced it:\n%s", stderr)
	}
	for _, not := range []string{"predates", "make build", "abcd update"} {
		if strings.Contains(stderr, not) {
			t.Errorf("the refusal names %q:\n%s", not, stderr)
		}
	}
}

// TestPreviewCutOutsideACheckoutNamesTheCheckout carries the retired changelog
// verb's outside-a-checkout refusal onto the preview: the cut the dry run
// would render is not read, and the reason names the missing checkout rather
// than git's bare exit status or the caller's path.
func TestPreviewCutOutsideACheckoutNamesTheCheckout(t *testing.T) {
	dir := t.TempDir()
	cut, reason := previewCut(dir)
	if cut != nil {
		t.Fatalf("a cut was read outside a checkout: %+v", *cut)
	}
	if !strings.Contains(reason, "not inside a git repository") || strings.Contains(reason, "exit status") {
		t.Errorf("reason = %q, want it to name the missing checkout", reason)
	}
	if strings.Contains(reason, dir) {
		t.Errorf("reason leaks an absolute path: %q", reason)
	}
}
