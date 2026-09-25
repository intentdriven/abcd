package lint

import (
	"strings"
	"testing"
)

// An intent's bucket and its specs' buckets say one thing about the same work:
// a planned intent has an open spec to build against, and a shipped intent has
// none left open. A merge whose rename detection files a new planned intent's
// spec into closed/, or the intent into shipped/ beside an open spec, passes
// every rule that reads one record at a time (iss-2609181121522692).
func TestSpecLifecycleRefusesAnIntentWhoseBucketDisagreesWithItsSpecs(t *testing.T) {
	for name, tc := range map[string]struct {
		intentBucket string
		specs        map[string]string // spec file -> bucket
		want         string            // "" = no bucket-agreement finding
	}{
		"planned, its only spec closed": {"planned", map[string]string{"spc-1-a.md": "closed"},
			"planned intent 'itd-10' has no open spec"},
		"planned, a closed spec and an open remainder": {"planned", map[string]string{"spc-1-a.md": "closed", "spc-2-b.md": "open"}, ""},
		"planned, its spec open":                       {"planned", map[string]string{"spc-1-a.md": "open"}, ""},
		"shipped, its spec still open": {"shipped", map[string]string{"spc-1-a.md": "open"},
			"shipped intent 'itd-10' has a spec still open: spc-1"},
		"shipped, one closed and one open": {"shipped", map[string]string{"spc-1-a.md": "closed", "spc-2-b.md": "open"},
			"shipped intent 'itd-10' has a spec still open: spc-2"},
		"shipped, every spec closed": {"shipped", map[string]string{"spc-1-a.md": "closed", "spc-2-b.md": "closed"}, ""},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "rec/intents/"+tc.intentBucket+"/itd-10-alpha.md",
				"---\nid: itd-10\nkind: standalone\nspec_id: spc-1\n---\n# ok\n")
			for file, bucket := range tc.specs {
				id := strings.SplitN(file, "-", 3)
				writeFile(t, root, "rec/specs/"+bucket+"/"+file,
					"---\nid: "+id[0]+"-"+id[1]+"\nslug: s\nintent: itd-10\n---\n# ok\n")
			}
			cfg := Config{Roots: []string{"rec"}, Rules: map[string]RuleConfig{
				"spec_lifecycle": {Enabled: true, Severity: "blocker", SpecsDir: "specs", IntentsDir: "intents"},
			}}
			fs, err := Lint(cfg, root)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, f := range fs {
				if f.RuleID == "spec_lifecycle" && strings.Contains(f.Message, "intent 'itd-10' has") {
					got = append(got, f.File+": "+f.Message)
				}
			}
			if tc.want == "" {
				if len(got) > 0 {
					t.Fatalf("agreeing buckets reported: %q", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) || !strings.Contains(got[0], "itd-10-alpha.md") {
				t.Fatalf("want one finding on the intent containing %q, got %q", tc.want, got)
			}
		})
	}
}
