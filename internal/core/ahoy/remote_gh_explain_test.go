package ahoy

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/tools"
)

// emptyPath points PATH at an empty directory so no external tool is found.
func emptyPath(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
}

// TestMissingGhIsExplained is itd-63 criterion 1 at the remote verbs: a missing
// gh is named with the registry's explanation, not a bare sentence.
func TestMissingGhIsExplained(t *testing.T) {
	emptyPath(t)
	_, err := runGH(t.TempDir(), nil, "api", "repos/example/example")
	if err == nil {
		t.Fatal("runGH succeeded with no gh on PATH")
	}
	var missing *tools.MissingError
	if !errors.As(err, &missing) || missing.Explanation.Tool != "gh" {
		t.Fatalf("error carries no registry explanation: %v", err)
	}
	e := tools.Explain("gh", tools.GitHubSettings)
	for _, want := range []string{"not on PATH", "required for", e.StepText(), "gh auth login"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q:\n%s", want, err)
		}
	}
}
