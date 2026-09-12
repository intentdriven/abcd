package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// personaNames is documented as the persona registry's given-name sequence,
// embedded rather than read at run time because the released binaries do not
// carry .abcd/development/. Nothing checked that claim, and it now carries more
// weight than it used to: the list gates a privacy EXEMPTION as well as the
// device-name heuristic (iss-2609100505145554), so drift is a correctness
// problem in both directions. A name added to the registry but missing here
// keeps firing as a leak; a name dropped from the registry but left here stays
// exempt, which is the unsafe direction.
func TestPersonaNamesMatchTheRegistry(t *testing.T) {
	root := moduleRoot(t)
	data, err := os.ReadFile(filepath.Join(root, ".abcd", "development", "personas.json"))
	if err != nil {
		t.Skipf("registry not present in this checkout: %v", err)
	}
	var reg struct {
		Personas []struct {
			Name string `json:"name"`
		} `json:"personas"`
	}
	if err := json.Unmarshal(data, &reg); err != nil {
		t.Fatalf("registry does not parse: %v", err)
	}
	if len(reg.Personas) == 0 {
		t.Fatal("registry lists no personas")
	}
	want := make(map[string]bool, len(reg.Personas))
	for _, p := range reg.Personas {
		want[strings.ToLower(p.Name)] = true
	}
	for name := range want {
		if !personaNames[name] {
			t.Errorf("registry persona %q is missing from personaNames — it will be flagged as a leak", name)
		}
	}
	for name := range personaNames {
		if !want[name] {
			t.Errorf("personaNames carries %q, which the registry does not list — it is exempt for no recorded reason", name)
		}
	}
}

// moduleRoot walks up from the package directory to the directory holding go.mod.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod found above the package directory")
		}
		dir = parent
	}
}
