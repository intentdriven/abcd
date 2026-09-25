// Package surface models abcd's public compatibility surface — the commands,
// flags, and manifest entries a consumer can bind to — as a transport-agnostic
// value that is snapshotted, committed, and diffed between releases (itd-73,
// spc-10). The snapshot is the artefact the release guardrail compares: a
// removed command, a removed flag, a flag that becomes required, or a removed
// manifest entry is a structural break that a release must declare.
//
// It lives under internal/core, not beside the CLI front door, because the
// snapshot is DATA and the guardrail that diffs two of them is domain logic:
// neither may depend on cobra. The walk that reads the live command tree needs
// cobra, so it lives in internal/surface/cli and hands its result in here. The
// dependency never points the other way. The package name shares a word with the
// internal/surface/* front-door tier and nothing else: that tier is about
// transports, this package is about what those transports expose.
//
// Determinism is this package's contract, not a convenience. The snapshot is a
// committed file gated by a drift test, so the same tree must encode to the same
// bytes on every machine and every run. Every collection is sorted by a stable
// key, nothing is derived from map iteration order, and nothing reads the clock
// or the environment.
package surface

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// SchemaVersion is the version of the snapshot's on-disk shape. It is written
// into every encoded snapshot and checked on decode, so a guardrail can never
// silently compare two snapshots written to different shapes — a mismatch there
// would report phantom breaks or, worse, miss real ones.
//
// Version 2 added each command's help placement (Command.Group and
// Command.Block, itd-146).
const SchemaVersion = 2

// readableVersions is every shape Decode accepts. Version 1 stays readable
// because the release guardrail reads its baseline out of the last release tag,
// and every tag cut before version 2 carries a version-1 file. Version 1 is
// version 2 with no placement recorded, and Diff compares no placement, so
// reading one as the other loses nothing the guardrail judges.
var readableVersions = map[int]bool{1: true, SchemaVersion: true}

// SnapshotPath is where the committed snapshot lives, repo-relative and
// slash-separated.
//
// It sits with the type rather than with the front door that generates the file,
// because the release guardrail reads the baseline out of a git TREE (`ls-tree`/
// `cat-file` take slash-separated repo-relative paths) and must name the same
// location the generator writes and the drift test gates. One constant is what
// keeps a guardrail from silently reading a path nothing writes and concluding
// there is no baseline.
const SnapshotPath = ".abcd/development/release/surface.json"

// Snapshot is the whole compatibility surface at one commit.
//
// The field order is the JSON key order: encoding/json emits struct fields in
// declaration order, so the shape on disk is fixed by this declaration and by
// nothing else. Maps are deliberately absent from the whole type — a map would
// make the encoded bytes depend on iteration order.
type Snapshot struct {
	SchemaVersion int             `json:"schema_version"`
	Commands      []Command       `json:"commands"`
	Manifest      []ManifestEntry `json:"manifest"`
}

// Command is one command in the tree, identified by its full command path
// ("abcd intent plan") because that path is what a caller types and therefore
// what a rename breaks.
//
// Hidden commands are included. The operator-internal `hook` subtree is hidden
// from the documentation page but is still public surface for compatibility
// purposes: harness wiring invokes it by name, so removing or renaming it breaks
// installations even though no user ever reads about it. Recording Hidden lets
// the guardrail report what kind of surface changed without letting it skip any.
//
// Hidden is what the command DECLARES, not whether it is reachable in help: a
// visible subcommand of a hidden parent records Hidden=false, because that is
// what the tree says and the hiding is the help renderer's doing. The field is
// descriptive; presence in the snapshot is what decides a break.
type Command struct {
	Path   string `json:"path"`
	Hidden bool   `json:"hidden"`
	// Group is the help group a visible top-level verb is listed under
	// (itd-146): "set-up", "records", "checks", "portability", "release", or
	// "agents" for the agents-and-hosts block. Empty for a sub-command and for
	// a hidden command, and omitted from the encoding when empty.
	Group string `json:"group,omitempty"`
	// Block is which of the two help blocks lists the command: "people" or
	// "agents". A visible top-level verb always carries one; a sub-command
	// carries one only when it is listed in its own right, which today means a
	// sub-verb of the agents block (`guard hook`). Omitted when empty.
	//
	// Neither field is a compatibility claim. Diff never reads them, because a
	// regroup changes no invocation; they are here so that a regroup is VISIBLE
	// in the committed tree, its drift test and the release gate's stale-surface
	// refusal, which is where PlacementChanges names it.
	Block string `json:"block,omitempty"`
	Flags []Flag `json:"flags"`
}

// Flag is one flag declared ON a command — its own flags plus the persistent
// flags it declares, never the persistent flags it inherits. A persistent flag
// is therefore recorded exactly once, on the command that declares it, and its
// removal there is one break rather than one per descendant.
type Flag struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand"`
	// Type is the flag's value type as the flag library names it ("bool",
	// "string", "stringSlice"). A type change is a compatibility event in its own
	// right: `--since` going from string to stringSlice changes what callers may
	// pass.
	Type string `json:"type"`
	// Required records whether the flag must be supplied. Nothing in the tree
	// marks a flag required today, so every entry is false; the field exists so
	// that a flag LATER becoming required is diffed as the break it is, rather
	// than being invisible because the snapshot never modelled requiredness.
	Required bool `json:"required"`
	// Hidden mirrors Command.Hidden: an undocumented flag is still a flag a
	// script may pass.
	Hidden bool `json:"hidden"`
}

// ManifestEntry is one declared key path in one plugin manifest — the unit the
// guardrail counts as "a manifest surface entry".
//
// An entry is a PRESENCE, not a value. Key paths are leaf paths through the
// manifest JSON (see ManifestEntries for the flattening rules) and the value is
// deliberately not recorded, because the break taxonomy makes a removed entry a
// break while a changed description or a reordered list is not. Recording values
// would make every prose edit to a manifest look like a surface change and would
// bury the removals the guardrail exists to catch.
type ManifestEntry struct {
	File string `json:"file"`
	Key  string `json:"key"`
}

// NewSnapshot is the one constructor: it stamps the schema version and returns
// the canonical form of the surface it is given.
//
// Callers hand it whatever order their walk produced — cobra's registration
// order, a directory read, a map range — and canonicalisation here is what makes
// two runs over the same tree byte-identical. The input slices are copied, so a
// caller that keeps mutating its working slices cannot reorder a snapshot after
// the fact.
func NewSnapshot(commands []Command, entries []ManifestEntry) Snapshot {
	return canonical(Snapshot{SchemaVersion: SchemaVersion, Commands: commands, Manifest: entries})
}

// canonical returns s with every collection copied, sorted by a stable key, and
// nil collections normalised to empty ones.
//
// It is the single definition of "canonical" that both the constructor and the
// encoder use, so there is exactly one place where the ordering of the committed
// artefact is decided. Sort keys are the identity fields: a command's path and a
// flag's name are unique within their scope, and a manifest entry is identified
// by its file and key together.
func canonical(s Snapshot) Snapshot {
	out := Snapshot{
		SchemaVersion: s.SchemaVersion,
		Commands:      make([]Command, len(s.Commands)),
		Manifest:      make([]ManifestEntry, len(s.Manifest)),
	}
	copy(out.Commands, s.Commands)
	copy(out.Manifest, s.Manifest)

	for i := range out.Commands {
		flags := make([]Flag, len(out.Commands[i].Flags))
		copy(flags, out.Commands[i].Flags)
		sort.Slice(flags, func(a, b int) bool { return flags[a].Name < flags[b].Name })
		out.Commands[i].Flags = flags
	}
	sort.Slice(out.Commands, func(a, b int) bool { return out.Commands[a].Path < out.Commands[b].Path })
	sortEntries(out.Manifest)
	return out
}

// sortEntries orders manifest entries by file then key — the pair that identifies
// an entry. It is shared with ManifestEntries so the extractor's own output and
// the canonical form agree on one order, rather than the extractor emitting an
// array-positional order that only the constructor happens to repair.
func sortEntries(entries []ManifestEntry) {
	sort.Slice(entries, func(a, b int) bool {
		if entries[a].File != entries[b].File {
			return entries[a].File < entries[b].File
		}
		return entries[a].Key < entries[b].Key
	})
}

// Encode renders the snapshot as the committed artefact: canonical order, fixed
// key order, two-space indent, and exactly one trailing newline.
//
// It canonicalises defensively rather than trusting the caller to have gone
// through NewSnapshot, because Encode is the boundary that writes a file a drift
// test then gates — a snapshot that encodes differently depending on how it was
// built would turn that gate into a coin flip. HTML escaping is off so a key
// containing `<`, `>`, or `&` is written literally instead of as an escape that
// a human reader would have to decode.
func Encode(s Snapshot) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(canonical(s)); err != nil {
		return nil, fmt.Errorf("encoding surface snapshot: %w", err)
	}
	return buf.Bytes(), nil
}

// Decode reads a committed snapshot back, so a guardrail can diff the released
// baseline against the current tree.
//
// It is strict in every direction. Unknown fields are rejected because a field
// this binary does not understand means the file describes surface it cannot
// compare, and a schema version other than the one this binary writes is
// rejected for the same reason. Trailing content is rejected because the decoder
// is a STREAM decoder: it stops at the first value and would otherwise accept a
// blob that is a snapshot followed by anything at all. All three are fail-closed
// on purpose: a baseline that parses "successfully" into a partial or empty
// surface would make every removal look like a surface that never existed.
func Decode(data []byte) (Snapshot, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var s Snapshot
	if err := dec.Decode(&s); err != nil {
		return Snapshot{}, fmt.Errorf("decoding surface snapshot: %w", err)
	}
	if dec.More() {
		return Snapshot{}, fmt.Errorf("decoding surface snapshot: trailing content after the snapshot")
	}
	if !readableVersions[s.SchemaVersion] {
		return Snapshot{}, fmt.Errorf("surface snapshot schema version %d, want %d (or 1, the shape before help placement)",
			s.SchemaVersion, SchemaVersion)
	}
	return canonical(s), nil
}

// PlacementChanges names every command whose help group or block differs
// between committed and current, one line each in canonical path order:
// "abcd capture: group records → checks".
//
// It exists because a byte comparison of two snapshots can say only THAT they
// differ. The drift test and the release gate's stale-surface refusal both call
// it, so a verb moved between groups without regenerating the snapshot is named
// rather than left for the operator to find in a file of several thousand
// lines (itd-146 criterion 4). A command present on one side only is an
// addition or a removal, which Diff and the drift test already report; it is
// not a placement change and is not listed.
func PlacementChanges(committed, current Snapshot) []string {
	committed, current = canonical(committed), canonical(current)
	was := make(map[string]Command, len(committed.Commands))
	for _, c := range committed.Commands {
		was[c.Path] = c
	}
	var out []string
	for _, now := range current.Commands {
		before, ok := was[now.Path]
		if !ok {
			continue
		}
		if before.Group != now.Group {
			out = append(out, fmt.Sprintf("%s: group %s → %s", now.Path, placementName(before.Group), placementName(now.Group)))
		}
		if before.Block != now.Block {
			out = append(out, fmt.Sprintf("%s: block %s → %s", now.Path, placementName(before.Block), placementName(now.Block)))
		}
	}
	return out
}

// placementName spells an empty placement so a line never reads "group  → x".
func placementName(v string) string {
	if v == "" {
		return "(none)"
	}
	return v
}
