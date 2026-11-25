package spotcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/layered"
)

// configRoots is a repository whose .abcd/config.json holds body ("" for no
// file) and an empty home.
func configRoots(t *testing.T, body string) layered.Roots {
	t.Helper()
	repo := t.TempDir()
	if body != "" {
		if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return layered.Roots{Repo: repo, Home: t.TempDir()}
}

func TestConfigDefaults(t *testing.T) {
	c, err := readConfig(configRoots(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	if c != (Config{WindowDays: 7, WindowChanges: 0, PerGroup: 10}) {
		t.Fatalf("the bundled configuration is %+v", c)
	}
	c, err = readConfig(configRoots(t, `{"spotcheck":{"window_days":3,"window_changes":50,"per_group":4}}`))
	if err != nil || c != (Config{WindowDays: 3, WindowChanges: 50, PerGroup: 4}) {
		t.Fatalf("readConfig = %+v, %v", c, err)
	}
}

// TestPerGroupBounds: per_group 0, 101 and a string are refused naming the
// key, as are a window out of its bounds and a misspelt key.
func TestPerGroupBounds(t *testing.T) {
	for body, key := range map[string]string{
		`{"spotcheck":{"per_group":0}}`:           "per_group",
		`{"spotcheck":{"per_group":101}}`:         "per_group",
		`{"spotcheck":{"per_group":"ten"}}`:       "per_group",
		`{"spotcheck":{"window_days":0}}`:         "window_days",
		`{"spotcheck":{"window_days":91}}`:        "window_days",
		`{"spotcheck":{"window_changes":9}}`:      "window_changes",
		`{"spotcheck":{"window_changes":100001}}`: "window_changes",
		`{"spotcheck":{"per_groups":5}}`:          "per_groups",
	} {
		if _, err := readConfig(configRoots(t, body)); err == nil || !strings.Contains(err.Error(), key) {
			t.Errorf("%s: err %v; want a refusal naming %s", body, err, key)
		}
	}
}
