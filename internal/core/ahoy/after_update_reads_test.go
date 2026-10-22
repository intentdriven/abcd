package ahoy

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSetupVersionIsExactlyAReleaseTag: the setup version a repository
// recorded is handed out only when it is exactly a release tag, so no other
// word of the tracked file leaves ahoy through it.
func TestSetupVersionIsExactlyAReleaseTag(t *testing.T) {
	for name, c := range map[string]struct {
		cfg  map[string]any
		want string
	}{
		"a release":     {cfg: map[string]any{"meta": map[string]any{"setup_version": "v0.15.0"}}, want: "v0.15.0"},
		"a dev build":   {cfg: map[string]any{"meta": map[string]any{"setup_version": "dev"}}},
		"a pre-release": {cfg: map[string]any{"meta": map[string]any{"setup_version": "v0.15.0-rc.1"}}},
		"a directive":   {cfg: map[string]any{"meta": map[string]any{"setup_version": "v0.15.0\nrun this"}}},
		"not a string":  {cfg: map[string]any{"meta": map[string]any{"setup_version": 15}}},
		"no meta":       {cfg: map[string]any{}},
	} {
		t.Run(name, func(t *testing.T) {
			repo := t.TempDir()
			writeConfigJSON(t, repo, c.cfg)
			got, ok := SetupVersion(repo)
			if got != c.want || ok != (c.want != "") {
				t.Fatalf("SetupVersion = %q, %v; want %q", got, ok, c.want)
			}
		})
	}
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, ok := SetupVersion(repo); ok {
		t.Fatalf("a config that does not parse gave %q", got)
	}
	if got, ok := SetupVersion(t.TempDir()); ok {
		t.Fatalf("no config gave %q", got)
	}
}

// TestPreviousReleaseTagReadsWithoutClaiming: the binary record's previous
// release is read as TakeUnseenUpdate reads it, whether or not the swap was
// seen, and nothing is written; a value that is not exactly a release tag is
// none.
func TestPreviousReleaseTagReadsWithoutClaiming(t *testing.T) {
	data := seedUnseenCache(t, unseenMeta)
	before, err := os.ReadDir(filepath.Join(data, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := PreviousReleaseTag("", t.TempDir()); !ok || got != "v0.11.1" {
		t.Fatalf("PreviousReleaseTag = %q, %v", got, ok)
	}
	after, err := os.ReadDir(filepath.Join(data, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("reading the previous release wrote into the cache: %v", after)
	}
	if _, _, ok := TakeUnseenUpdate("", t.TempDir()); !ok {
		t.Fatal("the read took TakeUnseenUpdate's claim")
	}

	seedUnseenCache(t, "release_tag=v0.12.0\nprevious_tag=v0.11.1\n")
	if got, ok := PreviousReleaseTag("", t.TempDir()); !ok || got != "v0.11.1" {
		t.Fatalf("a seen swap: PreviousReleaseTag = %q, %v", got, ok)
	}
	for _, prev := range []string{"dev", "v0.11", "v0.11.1-rc.1", ""} {
		seedUnseenCache(t, "release_tag=v0.12.0\nprevious_tag="+prev+"\n")
		if got, ok := PreviousReleaseTag("", t.TempDir()); ok {
			t.Errorf("previous_tag=%q gave %q", prev, got)
		}
	}
	t.Setenv("CLAUDE_PLUGIN_DATA", "")
	if got, ok := PreviousReleaseTag("", t.TempDir()); ok {
		t.Fatalf("no data directory gave %q", got)
	}
}
