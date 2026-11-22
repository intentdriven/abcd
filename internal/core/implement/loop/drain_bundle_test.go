package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// The cap on a bundle and who joins one (itd-2611201450052326,
// spc-2611201455051897 step 3): drain.bundle_max through the layers, kept on
// the drain's state, and bundleCandidates over a ledger.

// bundleRoots is a repository and a home, each with the drain namespace's
// object given in its config file, or no file for "".
func bundleRoots(t *testing.T, repo, machine string) layered.Roots {
	t.Helper()
	r := layered.Roots{Repo: t.TempDir(), Home: t.TempDir()}
	if repo != "" {
		writeConfigAt(t, filepath.Join(r.Repo, ".abcd", "config.json"), repo)
	}
	if machine != "" {
		writeConfigAt(t, abcdhome.Path(r.Home, "config.json"), machine)
	}
	return r
}

func writeConfigAt(t *testing.T, file, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestTheBundleCapIsReadThroughTheLayers: the repository's value over the
// machine's over the bundled default of 1, each with the layer it came from.
func TestTheBundleCapIsReadThroughTheLayers(t *testing.T) {
	cases := []struct {
		name, repo, machine string
		value               int
		layer, origin       string
	}{
		{"repository", `{"drain":{"bundle_max":3}}`, "", 3, "repo", ".abcd/config.json"},
		{"machine", "", `{"drain":{"bundle_max":4}}`, 4, "machine", "config.json"},
		{"both", `{"drain":{"bundle_max":3}}`, `{"drain":{"bundle_max":4}}`, 3, "repo", ".abcd/config.json"},
		{"neither", "", "", 1, "bundled", "bundled"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, err := resolveBundleMax(bundleRoots(t, c.repo, c.machine))
			if err != nil {
				t.Fatal(err)
			}
			if v.Value != c.value || v.Layer != c.layer || !strings.Contains(v.Origin, c.origin) {
				t.Fatalf("drain.bundle_max = %+v, want %d from the %s layer (%s)", v, c.value, c.layer, c.origin)
			}
		})
	}
	if BundledBundleMax != 1 {
		t.Fatalf("the bundled cap is %d, want 1 until the split and the figures exist", BundledBundleMax)
	}
}

// TestABundleCapOutOfRangeOrMisspeltIsRefusedNamingTheFile: 0, 11 and a
// misspelt key under drain are refused, each naming the file it sits in.
func TestABundleCapOutOfRangeOrMisspeltIsRefusedNamingTheFile(t *testing.T) {
	cases := []struct {
		name, repo, machine, file, says string
	}{
		{"zero", `{"drain":{"bundle_max":0}}`, "", ".abcd/config.json", "from 1 to 10"},
		{"eleven", "", `{"drain":{"bundle_max":11}}`, "config.json", "from 1 to 10"},
		{"misspelt", `{"drain":{"bundle_mx":3}}`, "", ".abcd/config.json", "bundle_mx"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := resolveBundleMax(bundleRoots(t, c.repo, c.machine))
			rf := mustRefusal(t, err)
			if rf.Stage != StageDrain || !strings.Contains(rf.Reason, c.file) || !strings.Contains(rf.Reason, c.says) {
				t.Fatalf("refused at %s, naming %s and %q: %+v", StageDrain, c.file, c.says, rf)
			}
			if !strings.Contains(rf.Remedy, "drain.bundle_max") {
				t.Fatalf("the remedy names the key and its range: %q", rf.Remedy)
			}
		})
	}
}

// TestTheBundleCapIsKeptOnTheDrainState: a new drain records the cap with its
// layer and origin, and a running drain keeps it when the file changes.
func TestTheBundleCapIsKeptOnTheDrainState(t *testing.T) {
	repo := drainRepo(t)
	home := t.TempDir()
	cfg := abcdhome.Path(home, "config.json")
	writeConfigAt(t, cfg, `{"drain":{"bundle_max":3}}`)
	o := Options{Roots: &layered.Roots{Repo: repo.Root(), Home: home}}
	if _, err := Drain(repo.Root(), o, DrainOptions{}); err != nil {
		t.Fatal(err)
	}
	st, live, err := readDrain(repo.Root())
	if err != nil || !live {
		t.Fatalf("the drain started: %v %v", live, err)
	}
	if b := st.BundleMax; b == nil || b.Value != 3 || b.Layer != "machine" || !strings.Contains(b.Origin, "config.json") {
		t.Fatalf("the drain keeps the cap it started with and where it came from: %+v", b)
	}
	writeConfigAt(t, cfg, `{"drain":{"bundle_max":5}}`)
	if _, err := Drain(repo.Root(), o, DrainOptions{}); err != nil {
		t.Fatal(err)
	}
	st, _, err = readDrain(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if b := st.BundleMax; b == nil || b.Value != 3 {
		t.Fatalf("a running drain keeps its cap whatever the file says by then: %+v", b)
	}
	for _, l := range st.Lanes {
		if len(l.Bundle) > 0 {
			t.Fatalf("no lane is formed with members before the drain forms bundles: %+v", l)
		}
	}
	if st.Forming != nil {
		t.Fatalf("nothing forms a bundle yet: %+v", st.Forming)
	}
}

// bundleLedger is a plan in drain order and each issue's area.
func bundleLedger() ([]capture.DrainVerdict, map[string]string) {
	v := func(id string, sev capture.Severity, cat capture.Category, out capture.DrainOutcome) capture.DrainVerdict {
		return capture.DrainVerdict{ID: id, Severity: sev, Category: cat, Outcome: out}
	}
	order := []capture.DrainVerdict{
		v("iss-2611200000000001", "minor", "bug", capture.DrainEligible),           // before the lead
		v("iss-2611200000000002", "minor", "bug", capture.DrainEligible),           // the lead
		v("iss-2611200000000003", "major", "bug", capture.DrainEligible),           // joins: another severity
		v("iss-2611200000000004", "minor", "documentation", capture.DrainEligible), // another category
		v("iss-2611200000000005", "minor", "bug", capture.DrainEligible),           // another directory
		v("iss-2611200000000006", "minor", "security", capture.DrainEligible),      // security
		v("iss-2611200000000007", "minor", "bug", capture.DrainEligible),           // a root file
		v("iss-2611200000000008", "minor", "bug", capture.DrainHandBack),           // not eligible
		v("iss-2611200000000009", "nitpick", "bug", capture.DrainEligible),         // joins
		v("iss-2611200000000010", "minor", "bug", capture.DrainEligible),           // a root file, as a lead
		v("iss-2611200000000011", "minor", "security", capture.DrainEligible),      // security, as a lead
		v("iss-2611200000000012", "minor", "security", capture.DrainEligible),      // security beside it
		v("iss-2611200000000013", "minor", "bug", capture.DrainEligible),           // no found_at path
	}
	area := map[string]string{
		"iss-2611200000000001": "internal/a", "iss-2611200000000002": "internal/a", "iss-2611200000000003": "internal/a",
		"iss-2611200000000004": "internal/a", "iss-2611200000000005": "internal/b", "iss-2611200000000006": "internal/a",
		"iss-2611200000000007": ".", "iss-2611200000000008": "internal/a", "iss-2611200000000009": "internal/a",
		"iss-2611200000000010": ".", "iss-2611200000000011": "internal/s", "iss-2611200000000012": "internal/s",
		"iss-2611200000000013": "",
	}
	return order, area
}

// TestWhoJoinsABundle: the eligible issues after the lead of its category and
// in its area join, whatever their severity, up to the cap; another category,
// another directory, security, the root and an ineligible issue do not; a
// lead that is not bundleable has none, and a cap of 1 takes none.
func TestWhoJoinsABundle(t *testing.T) {
	order, areas := bundleLedger()
	area := func(id string) string { return areas[id] }
	lead := "iss-2611200000000002"
	if got := bundleCandidates(order, lead, 10, area); !slices.Equal(got, []string{"iss-2611200000000003", "iss-2611200000000009"}) {
		t.Fatalf("who joins the lead: %v", got)
	}
	if got := bundleCandidates(order, lead, 2, area); !slices.Equal(got, []string{"iss-2611200000000003"}) {
		t.Fatalf("a cap of 2 takes the lead and one member: %v", got)
	}
	if got := bundleCandidates(order, lead, 1, area); len(got) != 0 {
		t.Fatalf("a cap of 1 takes no member: %v", got)
	}
	for _, alone := range []string{"iss-2611200000000010", "iss-2611200000000011", "iss-2611200000000013", "iss-2611200000000008"} {
		if got := bundleCandidates(order, alone, 10, area); len(got) != 0 {
			t.Fatalf("a lead that is not bundleable (%s: area %q) opens alone: %v", alone, areas[alone], got)
		}
	}
	if got := bundleCandidates(order, "iss-2611200000000005", 10, area); len(got) != 0 {
		t.Fatalf("a lead with no issue of its kind in its place has none: %v", got)
	}
}

// TestAnIssuesAreaIsItsDirectory: the directory holding the found_at path,
// the path itself when it is a directory in the checkout, "." for a root file
// and "" for no path.
func TestAnIssuesAreaIsItsDirectory(t *testing.T) {
	root := t.TempDir()
	for _, f := range []string{"internal/a/x.go", "Makefile"} {
		writeConfigAt(t, filepath.Join(root, filepath.FromSlash(f)), "x")
	}
	for foundAt, want := range map[string]string{
		"internal/a/x.go:12": "internal/a",
		"internal/a":         "internal/a",
		"internal/gone/y.go": "internal/gone",
		"Makefile":           ".",
		"go.mod":             ".",
		"the parser":         "",
	} {
		if got := issueArea(root, capture.Issue{FoundAt: foundAt}); got != want {
			t.Errorf("the area of found_at %q is %q, want %q", foundAt, got, want)
		}
	}
}

// TestThePreviewTakesEachIssueOnce: the dry run's bundles take each lead in
// the drain order with its candidates, and an issue a lead took never leads.
func TestThePreviewTakesEachIssueOnce(t *testing.T) {
	order, areas := bundleLedger()
	area := func(id string) string { return areas[id] }
	got := previewBundles(capture.DrainPlan{Dispositions: order}, 3, area)
	want := []DrainBundlePreview{{Lead: "iss-2611200000000001", Members: []string{"iss-2611200000000002", "iss-2611200000000003"}}}
	if len(got) != 1 || got[0].Lead != want[0].Lead || !slices.Equal(got[0].Members, want[0].Members) {
		t.Fatalf("the preview: %+v, want %+v", got, want)
	}
	got = previewBundles(capture.DrainPlan{Dispositions: order}, 10, area)
	if len(got) != 1 || len(got[0].Members) != 3 {
		t.Fatalf("one bundle takes every issue of its kind in its place: %+v", got)
	}
	if got := previewBundles(capture.DrainPlan{Dispositions: order}, 1, area); len(got) != 0 {
		t.Fatalf("a cap of 1 previews no bundle: %+v", got)
	}
}
