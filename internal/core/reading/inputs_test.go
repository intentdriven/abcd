package reading

import (
	"strconv"
	"strings"
	"testing"
)

func TestOutputTypeCarriesOutputSchemaVersion(t *testing.T) {
	if want := "/" + strconv.Itoa(OutputSchemaVersion); !strings.HasSuffix(OutputType, want) {
		t.Errorf("OutputType %q does not end in %q, the version the input registry names", OutputType, want)
	}
}

func TestOutputExamplePassesCheck(t *testing.T) {
	if err := CheckOutput(t.TempDir(), []byte(OutputExample)); err != nil {
		t.Fatalf("OutputExample: %v", err)
	}
	if err := CheckOutput(t.TempDir(), []byte(strings.Replace(OutputExample, `"rdg-2609250000000001"`, `"run-1"`, 1))); err == nil {
		t.Error("CheckOutput accepted a malformed run id")
	}
}
