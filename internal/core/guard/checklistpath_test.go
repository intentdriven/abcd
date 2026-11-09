package guard

import (
	"strings"
	"testing"
)

// TestChecklistStorePathNamesTheTwoFilesTheVerbsWrite is spc-2610230805429429
// step 5: the host's own file tools are refused the checklist and its last good
// copy in any working copy's local tier, whatever case the volume folds and
// however the path is spelled, and nothing else is caught: a file of that name
// outside the tier, the tier's other files, and the shared working tier.
func TestChecklistStorePathNamesTheTwoFilesTheVerbsWrite(t *testing.T) {
	for p, want := range map[string]bool{
		"/home/x/repo/.abcd/.work.local/checklist.json":                   true,
		"/home/x/repo/.abcd/.work.local/checklist.last-good.json":         true,
		"/home/x/repo/sub/../.abcd/.work.local/checklist.json":            true,
		"/home/x/repo/.abcd/.work.local/./checklist.json":                 true,
		"/home/x/REPO/.ABCD/.Work.Local/Checklist.JSON":                   true,
		"/home/x/repo/.abcd/.work.local/CHECKLIST.LAST-GOOD.JSON":         true,
		"/home/x/repo/.abcd/.work.local/checklist.json/../checklist.json": true,
		"/home/x/repo/checklist.json":                                     false,
		"/home/x/repo/.abcd/work/checklist.json":                          false,
		"/home/x/repo/.abcd/.work.local/scratch/checklist.json":           false,
		"/home/x/repo/.abcd/.work.local/checklist.lock":                   false,
		"/home/x/repo/.abcd/.work.local/NEXT.md":                          false,
		"/home/x/repo/.work.local/checklist.json":                         false,
		"": false,
	} {
		if got := ChecklistStorePath(p); got != want {
			t.Errorf("ChecklistStorePath(%q) = %v, want %v", p, got, want)
		}
	}
}

// TestChecklistStoreRefusalNamesTheVerbs: the refusal says why the file is the
// verbs' alone and what to run instead, so a refused session moves on.
func TestChecklistStoreRefusalNamesTheVerbs(t *testing.T) {
	r := ChecklistStoreRefusal()
	for _, want := range []string{"checklist.json", "abcd checklist add", "abcd checklist set", "secret"} {
		if !strings.Contains(r, want) {
			t.Errorf("the refusal must say %q; got %q", want, r)
		}
	}
}
