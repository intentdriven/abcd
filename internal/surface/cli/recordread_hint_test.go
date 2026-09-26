package cli

import (
	"strings"
	"testing"
)

// recordread_hint_test.go — a "status"/"show" sub-verb a caller tries first is
// answered with the record dispatcher that does what they asked
// (iss-2609190337466942). The record verbs have no status or show sub-verb:
// `abcd <record-id>` prints the bucket, the path and the next move. The
// refusal stands (unrecognized-input-never-writes); only its message grows.

func TestIntentStatusAndShowNameTheRecordDispatcher(t *testing.T) {
	intentTestRepo(t)
	for _, args := range [][]string{
		{"intent", "status", "itd-5"},
		{"intent", "show", "itd-5"},
		{"intent", "status"},
	} {
		out, err := runCLIErr(t, args...)
		if err == nil {
			t.Fatalf("`abcd %s` must refuse:\n%s", strings.Join(args, " "), out)
		}
		want := "`abcd itd-5`"
		if len(args) == 2 {
			want = "`abcd <itd-N>`"
		}
		if !strings.Contains(err.Error(), want) {
			t.Errorf("`abcd %s` refusal does not name %s:\n%v", strings.Join(args, " "), want, err)
		}
		if !strings.Contains(err.Error(), "nothing created") {
			t.Errorf("`abcd %s` refusal no longer says nothing was written:\n%v", strings.Join(args, " "), err)
		}
	}
}

func TestCaptureShowNamesTheRecordDispatcher(t *testing.T) {
	captureLedgerRepo(t)
	out, err := runCLIErr(t, "capture", "show", "iss-7")
	if err == nil {
		t.Fatalf("`abcd capture show iss-7` must refuse:\n%s", out)
	}
	if !strings.Contains(err.Error(), "`abcd iss-7`") {
		t.Errorf("refusal does not name `abcd iss-7`:\n%v", err)
	}
}

func TestSpecShowNamesTheRecordDispatcher(t *testing.T) {
	stalePluginRoot(t)
	code, _, stderr := runMain(t, "spec", "show", "spc-3")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "`abcd spc-3`") {
		t.Errorf("`abcd spec show spc-3` does not name `abcd spc-3`:\n%s", stderr)
	}
}

// Prose that merely begins with the word still files: the hint fires on the
// sub-verb shape only.
func TestIntentProseBeginningWithShowStillFiles(t *testing.T) {
	intentTestRepo(t)
	if _, err := runCLIErr(t, "intent", "show the operator what the ledger holds"); err != nil {
		t.Fatalf("prose beginning with show must file a draft: %v", err)
	}
}
