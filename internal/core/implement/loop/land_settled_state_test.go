package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A landing's `spec_settled` is state version 22's: a version-21 file carrying one is
// refused naming it, and one without it reads and comes back at the current version.
func TestASettledSpecIsTheVersionAfterTheUnsettled(t *testing.T) {
	const steps = "1. The parser\n   - packages: internal/core/spec\n2. The loop\n   - packages: internal/core/implement\n"
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		specRel: specWithSteps(steps),
	})
	l := f.twoStepLandWithNoResolutions(t)
	path := filepath.Join(l.Worktree, specRel)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(body), "A written design record.", "A written design record, and more.", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	f.step(t)
	root, id := f.repo.Root(), f.runID
	data := stateBytes(t, root, id)
	if !strings.Contains(string(data), `"spec_settled": true`) {
		t.Fatalf("the landing keeps the field: %s", data)
	}
	state := filepath.Join(root, filepath.FromSlash(StateRelPath(id)))
	at := fmt.Sprintf(`"schema_version": %d,`, SchemaVersion)
	older := fmt.Sprintf(`"schema_version": %d,`, schemaVersionUnsettled)
	if err := os.WriteFile(state, []byte(strings.Replace(string(data), at, older, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadState(root, id); err == nil || !strings.Contains(err.Error(), "spec_settled") {
		t.Fatalf("an older file carrying the field is refused naming it: %v", err)
	}
	if err := os.WriteFile(state, downgraded(t, data, schemaVersionUnsettled), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatalf("an older file without the field reads: %v", err)
	}
	if st.SchemaVersion != SchemaVersion {
		t.Fatalf("it reads as the current version: %d", st.SchemaVersion)
	}
}
