package launch

import (
	"errors"
	"strings"
	"testing"
)

// TestPreviewRetentionRefusesWhenTagsAreUnreadable is iss-194's code half: a
// tag listing that fails must not read as an empty release set, which would
// preview "nothing to prune" for a repository whose tags were never seen.
func TestPreviewRetentionRefusesWhenTagsAreUnreadable(t *testing.T) {
	root := t.TempDir() // not a git repository: the tag listing fails
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": [".claude-plugin", "README.md"]}`)
	writeFile(t, root, "README.md", "readme\n")
	writeLockstepTree(t, root, "", "", "")

	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3"})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	if !report.Retention.Refused || !strings.Contains(report.Retention.RefusalReason, "release tags could not be listed") {
		t.Fatalf("retention = %+v, want a refusal naming the unreadable tags", report.Retention)
	}
	if !anyReasonContains(report.WouldRefuseOn, "release tags could not be listed") {
		t.Errorf("the unread tag set must reach would_refuse_on, got %v", report.WouldRefuseOn)
	}
}

// TestMissingPayloadConfigIsNamed is iss-2608270559313719's core half: a
// repository with no launch payload gets an error the front door can recognise
// and explain, not only a raw missing-file message.
func TestMissingPayloadConfigIsNamed(t *testing.T) {
	_, err := DryRun(DryRunRequest{RepoRoot: t.TempDir()})
	if !errors.Is(err, ErrNoLaunchPayload) {
		t.Fatalf("a missing include config must carry ErrNoLaunchPayload, got %v", err)
	}
}

func anyReasonContains(reasons []string, fragment string) bool {
	for _, r := range reasons {
		if strings.Contains(r, fragment) {
			return true
		}
	}
	return false
}
