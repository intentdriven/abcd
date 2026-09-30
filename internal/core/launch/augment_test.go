package launch

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner/augmenttest"
)

func augmentedPayload(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, ArtefactRelPath, `{"kind": "plugin"}`)
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": ["commands"]}`)
	writeFile(t, root, "commands/doc.md", "the config holds "+augmenttest.Value+" in prose\n")
	return root
}

// TestDryRunReportsTheAugmentersFinding: the repository's opt-in augmenter
// reaches the launch scan, so what it flags refuses the release
// (iss-2608291814575788).
func TestDryRunReportsTheAugmentersFinding(t *testing.T) {
	augmenttest.Install(t, augmenttest.Fake())
	root := augmentedPayload(t)
	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range report.Scan.Findings {
		found = found || (f.File == "commands/doc.md" && f.Kind == augmenttest.Kind)
	}
	if !found || report.Scan.HardFails == 0 || report.WouldPublish {
		t.Fatalf("the augmented finding did not refuse the dry run: %+v", report.Scan)
	}
}

// TestLaunchFailsClosedOnTheAugmenterGap: a configured augmenter that is not
// installed refuses the release, as an Unscanned coverage gap with its reason
// and a hard fail, where the write paths only record it.
func TestLaunchFailsClosedOnTheAugmenterGap(t *testing.T) {
	augmenttest.Install(t, augmenttest.NotFound())
	root := augmentedPayload(t)
	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Scan.HardFails == 0 || report.WouldPublish {
		t.Fatalf("the gap did not fail the dry run closed: %+v", report.Scan)
	}
	var reason string
	for _, r := range report.WouldRefuseOn {
		if strings.Contains(r, "fake augmenter not on PATH") {
			reason = r
		}
	}
	if reason == "" || strings.Contains(reason, "payload file") {
		t.Fatalf("WouldRefuseOn does not name the augmenter gap as such: %v", report.WouldRefuseOn)
	}
	if _, err := Ship(ShipRequest{RepoRoot: root, Version: "1.0.0"}); !errors.Is(err, ErrShipBlocked) {
		t.Fatalf("ship was not blocked on the gap: %v", err)
	}
}
