package surface

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeManifests lays down a throwaway repo root carrying the two plugin
// manifests, so every manifest test states exactly the JSON it is about.
func writeManifests(t *testing.T, plugin, marketplace string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".claude-plugin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if plugin != "" {
		if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(plugin), 0o644); err != nil {
			t.Fatalf("write plugin.json: %v", err)
		}
	}
	if marketplace != "" {
		if err := os.WriteFile(filepath.Join(dir, "marketplace.json"), []byte(marketplace), 0o644); err != nil {
			t.Fatalf("write marketplace.json: %v", err)
		}
	}
	return root
}

func keysFor(t *testing.T, entries []ManifestEntry, file string) []string {
	t.Helper()
	var out []string
	for _, e := range entries {
		if strings.HasSuffix(e.File, file) {
			out = append(out, e.Key)
		}
	}
	return out
}

// TestManifestEntriesFlattensDeclaredKeys pins what an "entry" is: one leaf key
// path per declared value, objects flattened with dots, arrays of named objects
// keyed by their name, and any other array recorded as a single leaf so that
// reordering or editing its members is invisible to the guardrail.
func TestManifestEntriesFlattensDeclaredKeys(t *testing.T) {
	root := writeManifests(t,
		`{"name":"abcd","author":{"name":"REPPL","url":"https://example.invalid"},"keywords":["a","b"]}`,
		`{"name":"abcd-marketplace","owner":{"name":"REPPL"},"plugins":[{"name":"abcd","source":"./"}]}`)

	entries, err := ManifestEntries(root)
	if err != nil {
		t.Fatalf("ManifestEntries: %v", err)
	}

	wantPlugin := []string{"author.name", "author.url", "keywords", "name"}
	if got := keysFor(t, entries, "plugin.json"); !equalStrings(got, wantPlugin) {
		t.Fatalf("plugin.json keys = %v, want %v", got, wantPlugin)
	}
	wantMarket := []string{"name", "owner.name", "plugins[abcd].name", "plugins[abcd].source"}
	if got := keysFor(t, entries, "marketplace.json"); !equalStrings(got, wantMarket) {
		t.Fatalf("marketplace.json keys = %v, want %v", got, wantMarket)
	}
}

// TestManifestEntriesArrayKeying covers the array rules one at a time: named
// objects are keyed by name (so reordering the array is not a change), duplicate
// or missing names collapse to one leaf (the keying is ambiguous, so recording
// per-element entries would invent surface), and an empty container still
// registers its own presence.
func TestManifestEntriesArrayKeying(t *testing.T) {
	tests := []struct {
		name   string
		plugin string
		want   []string
	}{
		{
			name:   "named objects keyed by name",
			plugin: `{"items":[{"name":"b","v":1},{"name":"a","v":2}]}`,
			want:   []string{"items[a].name", "items[a].v", "items[b].name", "items[b].v"},
		},
		{
			name:   "duplicate names collapse to one leaf",
			plugin: `{"items":[{"name":"a"},{"name":"a"}]}`,
			want:   []string{"items"},
		},
		{
			name:   "unnamed objects collapse to one leaf",
			plugin: `{"items":[{"v":1}]}`,
			want:   []string{"items"},
		},
		{
			name:   "scalar array is one leaf",
			plugin: `{"items":["a","b"]}`,
			want:   []string{"items"},
		},
		{
			name:   "empty containers still register",
			plugin: `{"items":[],"obj":{}}`,
			want:   []string{"items", "obj"},
		},
		{
			name:   "null is a declared key",
			plugin: `{"a":null}`,
			want:   []string{"a"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := writeManifests(t, tc.plugin, `{"name":"m"}`)
			entries, err := ManifestEntries(root)
			if err != nil {
				t.Fatalf("ManifestEntries: %v", err)
			}
			if got := keysFor(t, entries, "plugin.json"); !equalStrings(got, tc.want) {
				t.Fatalf("keys = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestManifestEntriesTreatsVersionAsOrdinary is the guard for the release
// payload: the development tree carries no `version` in plugin.json and the
// rendered release payload does. Absence must not be an error or a special case,
// and presence must read as one ordinary added entry.
func TestManifestEntriesTreatsVersionAsOrdinary(t *testing.T) {
	without := writeManifests(t, `{"name":"abcd"}`, `{"name":"m"}`)
	entries, err := ManifestEntries(without)
	if err != nil {
		t.Fatalf("ManifestEntries without version: %v", err)
	}
	if got := keysFor(t, entries, "plugin.json"); !equalStrings(got, []string{"name"}) {
		t.Fatalf("keys without version = %v, want [name]", got)
	}

	with := writeManifests(t, `{"name":"abcd","version":"0.4.0"}`, `{"name":"m"}`)
	entries, err = ManifestEntries(with)
	if err != nil {
		t.Fatalf("ManifestEntries with version: %v", err)
	}
	if got := keysFor(t, entries, "plugin.json"); !equalStrings(got, []string{"name", "version"}) {
		t.Fatalf("keys with version = %v, want [name version]", got)
	}
}

// TestManifestEntriesRefusesUnreadableManifests keeps the snapshot fail-closed. A
// manifest that is THERE and malformed must be an error: reporting it as "no
// entries" would make every declared key look like a surface that was never
// declared.
//
// Absence is deliberately not in this table any more — it is a declaration that
// was never made, not a payload that failed to load, and
// TestManifestEntriesTreatsAnAbsentManifestAsNoDeclaration pins the other side
// (iss-2609100506255436). The removal an absent manifest could hide is still
// caught, one layer up, as a manifest_entry_removed break against the release
// baseline.
func TestManifestEntriesRefusesUnreadableManifests(t *testing.T) {
	tests := []struct {
		name        string
		plugin      string
		marketplace string
		want        string
	}{
		{"plugin malformed", `{`, `{"name":"m"}`, "plugin.json"},
		{"plugin not an object", `["a"]`, `{"name":"m"}`, "plugin.json"},
		{"marketplace malformed", `{"name":"p"}`, `{`, "marketplace.json"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := writeManifests(t, tc.plugin, tc.marketplace)
			if _, err := ManifestEntries(root); err == nil {
				t.Fatalf("ManifestEntries = nil error, want one naming %s", tc.want)
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want it to name %s", err, tc.want)
			}
		})
	}
}

// TestManifestEntriesRefusesAPresentManifestItCannotRead separates "not there"
// from "there and unreadable", which is the whole of the change absence made: a
// path occupied by something that is not a readable regular file is still a
// refusal, so the guarded read's fail-closed behaviour is not what the
// absent-manifest case relaxed.
func TestManifestEntriesRefusesAPresentManifestItCannotRead(t *testing.T) {
	root := writeManifests(t, "", `{"name":"m"}`)
	if err := os.Mkdir(filepath.Join(root, ".claude-plugin", "plugin.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ManifestEntries(root); err == nil {
		t.Fatal("a directory occupying plugin.json was read as an absent manifest")
	} else if !strings.Contains(err.Error(), "plugin.json") {
		t.Fatalf("error = %q, want it to name plugin.json", err)
	}
}

// TestManifestEntriesUsesRepoRelativePaths keeps the artefact machine-independent
// and privacy-safe: an absolute path from the machine that generated it would
// both leak a local path into a committed file and make the drift test fail on
// every other machine.
func TestManifestEntriesUsesRepoRelativePaths(t *testing.T) {
	root := writeManifests(t, `{"name":"p"}`, `{"name":"m"}`)
	entries, err := ManifestEntries(root)
	if err != nil {
		t.Fatalf("ManifestEntries: %v", err)
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.File, ".claude-plugin/") {
			t.Fatalf("entry file %q is not repo-relative", e.File)
		}
	}
}

// TestManifestEntriesTreatsAnAbsentManifestAsNoDeclaration is
// iss-2609100506255436 at this surface. An ABSENT plugin manifest and an
// UNREADABLE one are different facts: where the manifest lives is fixed by the
// harness's discovery rule, but whether the artefact has one at all is a
// per-repo fact — adr-19's version-location contract is abcd's own precedent for
// declaring such a fact rather than assuming it. A repo whose artefact is a
// binary, an application bundle or a library has no plugin manifest, and reading
// its absence as a broken payload is what stops `abcd changelog` before anything
// else runs.
//
// Nothing about the guardrail is weakened: the removal case
// (TestManifestEntriesAbsenceStillReportsTheRemoval) is caught as a BREAK
// against the release baseline, which is the channel that exists for it, rather
// than as an error that produces no verdict at all.
func TestManifestEntriesTreatsAnAbsentManifestAsNoDeclaration(t *testing.T) {
	neither := writeManifests(t, "", "")
	entries, err := ManifestEntries(neither)
	if err != nil {
		t.Fatalf("a repo that declares no plugin must resolve, not refuse: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("entries = %v, want none", entries)
	}

	onlyMarketplace := writeManifests(t, "", `{"name":"m"}`)
	entries, err = ManifestEntries(onlyMarketplace)
	if err != nil {
		t.Fatalf("ManifestEntries with no plugin.json: %v", err)
	}
	if got := keysFor(t, entries, "plugin.json"); len(got) != 0 {
		t.Fatalf("plugin.json keys = %v, want none", got)
	}
	if got := keysFor(t, entries, "marketplace.json"); !equalStrings(got, []string{"name"}) {
		t.Fatalf("marketplace.json keys = %v, want [name]", got)
	}
}

// TestManifestEntriesAbsenceStillReportsTheRemoval is the other half, and the
// reason absence may be empty rather than an error: a manifest that WAS declared
// at the last release and is gone now must still stop the cut. It does — as a
// manifest_entry_removed break, which the guardrail weighs against the cut's own
// records, instead of as an error that yields no verdict.
func TestManifestEntriesAbsenceStillReportsTheRemoval(t *testing.T) {
	before := writeManifests(t, `{"name":"abcd"}`, `{"name":"m"}`)
	baseEntries, err := ManifestEntries(before)
	if err != nil {
		t.Fatalf("ManifestEntries(before): %v", err)
	}
	after := writeManifests(t, "", `{"name":"m"}`)
	curEntries, err := ManifestEntries(after)
	if err != nil {
		t.Fatalf("ManifestEntries(after): %v", err)
	}

	breaks := Diff(NewSnapshot(nil, baseEntries), NewSnapshot(nil, curEntries))
	var found bool
	for _, b := range breaks {
		if b.Kind == BreakManifestRemoved && strings.Contains(b.Surface, "plugin.json") {
			found = true
		}
	}
	if !found {
		t.Fatalf("deleting plugin.json reported no manifest_entry_removed break: %+v", breaks)
	}
}
