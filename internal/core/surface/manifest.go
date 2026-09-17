package surface

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The two plugin manifests that together declare abcd's distribution surface: the
// plugin's own identity and the marketplace listing that points at it. They are
// repo-relative and slash-separated, because the path travels into the committed
// snapshot and an absolute path would both leak the generating machine and make
// the artefact fail to reproduce anywhere else.
const (
	pluginManifestPath      = ".claude-plugin/plugin.json"
	marketplaceManifestPath = ".claude-plugin/marketplace.json"
)

// manifestPaths is the ordered list the entries are collected from; the snapshot
// sorts them again, so this order is for readability only.
var manifestPaths = []string{pluginManifestPath, marketplaceManifestPath}

// maxManifestBytes caps the guarded read. A plugin manifest is a short
// declaration, so a file that is not one must not stream unbounded input into a
// generator that runs in CI.
const maxManifestBytes = 1 << 20

// nameKey is the field an array element is keyed by when the array holds objects.
// Both manifests identify list members this way (marketplace.json's plugins[]),
// so the name is the member's identity and its position is not.
const nameKey = "name"

// ManifestEntries reads both plugin manifests under repoRoot and flattens them
// into the set of declared key paths — the manifest half of the surface
// snapshot.
//
// An ENTRY is one leaf key path, and only its presence is recorded. The
// flattening rules are:
//
//   - an object contributes one entry per leaf beneath it, joined with dots
//     ("author.name"); an empty object contributes one entry at its own path, so
//     that emptying it is still visible;
//   - an array whose elements are all objects carrying a unique, non-empty
//     "name" contributes entries beneath each element, keyed by that name
//     ("plugins[abcd].source") — so reordering the array changes nothing while
//     removing or renaming a member is a removed entry;
//   - any other array (scalars such as keywords, unnamed objects, ambiguous
//     duplicate names) contributes exactly one entry at its own path. Keying
//     those by position would report a reorder as a break, and reordering is
//     explicitly not one;
//   - a scalar, including null, contributes one entry at its own path.
//
// Values are not recorded, deliberately: the break taxonomy makes a removed
// entry a break and a changed description or a reordered list not one, so
// carrying values would report every prose edit as a surface change.
//
// Nothing here treats any particular key as expected or required. plugin.json
// declares no version in the development tree and the rendered release payload
// adds one; both are ordinary entry sets, so the absence of a version is not an
// anomaly and its later presence reads as one added entry.
//
// A manifest that is PRESENT and cannot be read or parsed, or whose root is not
// a JSON object, is an error rather than an empty entry set: reporting "no
// entries" for a manifest that failed to load would make every declared entry
// look like surface that was never there.
//
// An ABSENT manifest is not that case. It contributes no entries, because a repo
// whose artefact is not a plugin — a binary, an application bundle, a library —
// declares no plugin surface, and treating the absence as an unreadable payload
// stopped every caller of the snapshot before it ran, `abcd changelog` included
// (iss-2609100506255436). Where a plugin manifest lives is fixed; whether this
// artefact has one is a per-repo fact, the same distinction adr-19 already drew
// for the version location. Nor does absence hide a removal: a manifest that WAS
// declared at the last release and is gone now yields a manifest_entry_removed
// break against that baseline, which is the channel the guardrail exists to
// report through — an error at this seam produced no verdict at all.
func ManifestEntries(repoRoot string) ([]ManifestEntry, error) {
	var out []ManifestEntry
	for _, rel := range manifestPaths {
		data, err := fsutil.ReadGuarded(filepath.Join(repoRoot, filepath.FromSlash(rel)), maxManifestBytes)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", rel, err)
		}
		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", rel, err)
		}
		out = append(out, flatten(rel, "", doc)...)
	}
	sortEntries(out)
	return out, nil
}

// flatten walks one decoded manifest value and returns the leaf entries beneath
// it. Object keys are visited in sorted order and array elements in document
// order; ManifestEntries sorts the result, so neither the JSON decoder's map
// iteration nor an element's position can reach the artefact.
func flatten(file string, prefix string, value any) []ManifestEntry {
	switch v := value.(type) {
	case map[string]any:
		if len(v) == 0 {
			return leaf(file, prefix)
		}
		var out []ManifestEntry
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			out = append(out, flatten(file, join(prefix, k), v[k])...)
		}
		return out
	case []any:
		names, ok := elementNames(v)
		if !ok {
			return leaf(file, prefix)
		}
		var out []ManifestEntry
		for i, elem := range v {
			out = append(out, flatten(file, prefix+"["+names[i]+"]", elem)...)
		}
		return out
	default:
		return leaf(file, prefix)
	}
}

// elementNames reports the per-element key for an array, and whether the array
// may be keyed at all. It may only be keyed when every element is an object with
// a non-empty string name and no two names collide — anything less makes the key
// ambiguous, and an ambiguous key would either invent entries or silently merge
// two of them.
func elementNames(arr []any) ([]string, bool) {
	if len(arr) == 0 {
		return nil, false
	}
	names := make([]string, 0, len(arr))
	seen := make(map[string]bool, len(arr))
	for _, elem := range arr {
		obj, isObject := elem.(map[string]any)
		if !isObject {
			return nil, false
		}
		name, isString := obj[nameKey].(string)
		if !isString || name == "" || seen[name] {
			return nil, false
		}
		seen[name] = true
		names = append(names, name)
	}
	return names, true
}

// leaf records one entry. An empty prefix means the whole document was a scalar
// or an empty object; that is a manifest with no declared keys, and recording an
// unnamed entry for it would be meaningless, so it contributes nothing.
func leaf(file, prefix string) []ManifestEntry {
	if prefix == "" {
		return nil
	}
	return []ManifestEntry{{File: file, Key: prefix}}
}

// join builds a dotted key path. Keys containing a dot or a bracket would render
// an ambiguous path; neither manifest uses such a key, and the snapshot is a
// comparison artefact rather than a parser input, so the paths stay readable
// instead of escaped.
func join(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}
