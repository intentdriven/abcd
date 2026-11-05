package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// TestImplementResumeListsNoneInAnEmptyCheckout: the verb is wired, writes nothing and
// says there is no unfinished run where there is none.
func TestImplementResumeListsNoneInAnEmptyCheckout(t *testing.T) {
	root := checklistCheckout(t)
	t.Chdir(root)
	var out, errb bytes.Buffer
	if code := Run([]string{"implement", "resume"}, &out, &errb); code != 0 {
		t.Fatalf("resume: %d %q", code, errb.String())
	}
	if !strings.Contains(out.String(), "no unfinished run") {
		t.Fatalf("the listing: %q", out.String())
	}
	out.Reset()
	if code := Run([]string{"--json", "implement", "resume"}, &out, &errb); code != 0 {
		t.Fatalf("resume --json: %d %q", code, errb.String())
	}
	var got loop.Listing
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got.Runs == nil {
		t.Fatalf("the JSON: %q %v", out.String(), err)
	}
}

// TestImplementResumeTakeRefusesAnUnknownRun: a take names a run this checkout
// holds.
func TestImplementResumeTakeRefusesAnUnknownRun(t *testing.T) {
	root := checklistCheckout(t)
	t.Chdir(root)
	var out, errb bytes.Buffer
	if code := Run([]string{"implement", "resume", "--take", "run-2611040000000001"}, &out, &errb); code != 2 {
		t.Fatalf("resume --take over nothing: %d %q %q", code, out.String(), errb.String())
	}
}
