package cli

import (
	"encoding/json"
	"testing"
)

// TestCaptureWontfixDuplicatesWritesTheTypedLink: the --duplicates flag on
// capture wontfix reaches the record's typed `duplicates` link, which is what a
// machine reader of the ledger reads, and not only the reason's prose
// (iss-2609291118049254).
func TestCaptureWontfixDuplicatesWritesTheTypedLink(t *testing.T) {
	captureLedgerRepo(t)
	mint := func(text string) string {
		t.Helper()
		var m struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(runCLI(t, "capture", text, "--json"), &m); err != nil || m.ID == "" {
			t.Fatalf("capture envelope unreadable: %v", err)
		}
		return m.ID
	}
	original := mint("the parser drops a trailing comma in the manifest")
	dup := mint("zebra quartz vellum xylophone, filed twice by another lane")

	runCLI(t, "capture", "wontfix", dup, "a duplicate of the original", "--duplicates", original)

	var listed struct {
		Issues []struct {
			ID         string   `json:"id"`
			Status     string   `json:"status"`
			Duplicates []string `json:"duplicates"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(runCLI(t, "capture", "list", "--wontfix", "--json"), &listed); err != nil {
		t.Fatal(err)
	}
	for _, iss := range listed.Issues {
		if iss.ID != dup {
			continue
		}
		if len(iss.Duplicates) != 1 || iss.Duplicates[0] != original {
			t.Fatalf("%s duplicates = %v, want [%s]", dup, iss.Duplicates, original)
		}
		return
	}
	t.Fatalf("%s is not in wontfix/: %+v", dup, listed.Issues)
}
