package layered

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

func writeRepoConfig(t *testing.T, body string) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}

func TestRecordSlugCapDefaultsToForty(t *testing.T) {
	got, err := RecordSlugCap(t.TempDir())
	if err != nil || got != recordid.DefaultMaxSlugLen {
		t.Fatalf("no config: cap %d, err %v; want 40", got, err)
	}
	got, err = RecordSlugCap(writeRepoConfig(t, `{"docs": {"target": "agents_md"}}`))
	if err != nil || got != 40 {
		t.Fatalf("a config without the key: cap %d, err %v; want 40", got, err)
	}
}

func TestRecordSlugCapReadsTheRepositorysValue(t *testing.T) {
	for _, n := range []string{"16", "60", "200"} {
		got, err := RecordSlugCap(writeRepoConfig(t, `{"records": {"slug_cap": `+n+`}}`))
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		if strconv.Itoa(got) != n {
			t.Fatalf("cap = %d, want %s", got, n)
		}
	}
}

// A value the repository cannot have meant refuses, naming the key, the value and
// the file, and never falls back to the default.
func TestRecordSlugCapRefusesANonsenseValue(t *testing.T) {
	for _, value := range []string{"0", "-1", "15", "201", "9000", `"forty"`, "40.5", "null", "true", "[40]"} {
		_, err := RecordSlugCap(writeRepoConfig(t, `{"records": {"slug_cap": `+value+`}}`))
		if err == nil {
			t.Errorf("%s: no refusal", value)
			continue
		}
		for _, want := range []string{SlugCapKey, ".abcd/config.json"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s: the refusal does not name %q: %v", value, want, err)
			}
		}
	}
	_, err := RecordSlugCap(writeRepoConfig(t, `{"records": {"slug_cap": 0}}`))
	if err == nil || !strings.Contains(err.Error(), "from 16 to 200") {
		t.Errorf("the refusal does not give the bounds: %v", err)
	}
}

func TestRecordSlugCapRefusesAMisspeltKey(t *testing.T) {
	_, err := RecordSlugCap(writeRepoConfig(t, `{"records": {"slug_cpa": 60}}`))
	if err == nil || !strings.Contains(err.Error(), "slug_cpa") {
		t.Fatalf("a misspelt key: %v; want it named", err)
	}
}

func TestRecordSlugCapRefusesAnUnreadableFile(t *testing.T) {
	if _, err := RecordSlugCap(writeRepoConfig(t, `{not json`)); err == nil {
		t.Fatal("a config that is not JSON read as the default")
	}
}

// The cap bounds names committed for every clone, so a machine's file sets nothing.
func TestRecordSlugCapIgnoresTheMachineLayer(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, "config.json"), []byte(`{"records": {"slug_cap": 60}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	defer SwapUserHomeForTest(func() (string, error) { return home, nil })()
	got, err := RecordSlugCap(t.TempDir())
	if err != nil || got != 40 {
		t.Fatalf("a machine's value: cap %d, err %v; want the default 40", got, err)
	}
}
