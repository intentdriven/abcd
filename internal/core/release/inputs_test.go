package release

import (
	"strings"
	"testing"
)

func TestChangelogExamplePassesCheck(t *testing.T) {
	if err := CheckChangelog(t.TempDir(), []byte(ChangelogExample)); err != nil {
		t.Fatalf("ChangelogExample: %v", err)
	}
	if err := CheckChangelog(t.TempDir(), []byte(strings.Replace(ChangelogExample, `"section": "Fixed"`, `"section": "Removed"`, 1))); err == nil {
		t.Error("CheckChangelog accepted a section the composer may not write")
	}
}
