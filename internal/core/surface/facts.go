package surface

import (
	"sort"
	"strings"
)

// facts.go — the one source of each command's facts (itd-2610301346085246,
// spc-2610301431075964 step 1). The compatibility snapshot (snapshot.go)
// records what a release may not silently break; the facts record what an
// agent needs to call the command right first time: its usage line, every
// flag's type, default and closed set of values, the filled-in inputs it reads
// and their schema versions, and its worked example.
//
// Both come from ONE walk of the live command tree in the front door
// (internal/surface/cli, surfaceWalk), so the snapshot and the facts can never
// describe two different trees. The snapshot keeps its schema-4 shape: none of
// what the facts add is a compatibility claim, and the release guard's
// comparison with the last tag reads the snapshot alone.
//
// Like the snapshot, the facts are data: nothing here reads cobra, the clock or
// the environment, and every collection is sorted by a stable key, because the
// per-command JSON files and the page blocks written from them are committed
// and gated by drift tests.

// FactsFormatVersion is the version of the facts' shape, written into every
// derived copy (the per-command JSON files and the page blocks). It moves only
// when the shape does, never with the binary's version: a binary version
// changes at every cut and in every development build, and would make every
// derived copy drift each time (the spec's second decision).
const FactsFormatVersion = 1

// The flag types the facts name. The flag library has more spellings than an
// agent needs (stringSlice and stringArray differ only in whether a comma
// splits a value), so the facts fold them into the few kinds that decide what
// may be typed after the flag.
const (
	FlagBool     = "bool"
	FlagString   = "string"
	FlagInt      = "int"
	FlagList     = "list"
	FlagDuration = "duration"
	FlagNumber   = "number"
)

// The kinds of filled-in input a command reads.
const (
	// InputJSON is a `--*-json` flag: a document the command decodes against
	// a Go type in its owning package, at a schema version, and whose shape
	// the input registry carries.
	InputJSON = "json"
	// InputAnswers is an interview's `--answers` file. Its shape is printed
	// by the interview that asks the questions, so the facts name it and no
	// more (the spec's note on answers files).
	InputAnswers = "answers"
)

// Facts is one command's facts.
//
// The field order is the JSON key order, as it is for the snapshot.
type Facts struct {
	// Path is the full command path, "abcd ideate record".
	Path string `json:"path"`
	// Hidden is what the command declares, as in the snapshot. A hidden
	// command keeps its facts, so a refusal on the hook plane can point at
	// them.
	Hidden bool `json:"hidden,omitempty"`
	// Sentence is the command's one sentence from the manifest in
	// sentences.go; empty for a command the manifest declares none for.
	Sentence string `json:"sentence,omitempty"`
	// Usage is the usage line the command's help prints.
	Usage string `json:"usage"`
	// Flags are the flags the command declares itself: its own and the
	// persistent flags it defines (marked Persistent), never the ones it
	// inherits, so a persistent flag has one home and a reader finds the rest
	// on the command's path.
	Flags []FlagFact `json:"flags"`
	// Inputs are the filled-in inputs the command reads.
	Inputs []InputFact `json:"inputs"`
	// Example is the command's worked example, as its help prints it.
	Example string `json:"example,omitempty"`
	// Subcommands are the names of the command's visible sub-commands.
	Subcommands []string `json:"subcommands,omitempty"`
}

// FlagFact is one flag a command declares.
type FlagFact struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	// Type is one of the Flag* kinds, or the flag library's own name for a
	// type the kinds do not fold.
	Type string `json:"type"`
	// Default is the flag's default as the flag library prints it, omitted
	// when it is the type's zero value (an empty string, false, 0, an empty
	// list).
	Default string `json:"default,omitempty"`
	// Required is cobra's own requiredness, read the way the snapshot reads it.
	Required bool `json:"required,omitempty"`
	// Allowed is the closed set the flag's reader checks a value against,
	// when it has one: the same slice its help line renders.
	Allowed []string `json:"allowed,omitempty"`
	// Persistent marks a flag every sub-command inherits.
	Persistent bool `json:"persistent,omitempty"`
	Hidden     bool `json:"hidden,omitempty"`
}

// InputFact is one filled-in input a command reads, as the facts name it.
// The input's whole shape (its fields and its example) is the registry's
// InputShape; the facts carry what every derived copy names.
type InputFact struct {
	// Flag is the flag's name without its dashes, "verdict-json".
	Flag string `json:"flag"`
	// Kind is InputJSON or InputAnswers.
	Kind string `json:"kind"`
	// Name says what the input is, "ideate verdict".
	Name string `json:"name"`
	// SchemaVersion is the version this binary reads, for an InputJSON input.
	SchemaVersion int `json:"schema_version,omitempty"`
	// VersionIn says where the document carries that version: one of the
	// Version* carriers. VersionNone means the version is the registry's
	// alone.
	VersionIn string `json:"version_in,omitempty"`
}

// FlagKind folds the flag library's type name into the kind the facts name.
func FlagKind(libraryType string) string {
	switch libraryType {
	case "bool":
		return FlagBool
	case "string":
		return FlagString
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "count":
		return FlagInt
	case "float32", "float64":
		return FlagNumber
	case "duration":
		return FlagDuration
	}
	if strings.HasSuffix(libraryType, "Slice") || strings.HasSuffix(libraryType, "Array") {
		return FlagList
	}
	return libraryType
}

// ZeroDefault reports whether def is the zero value the flag library prints
// for a flag of the given kind, which the facts leave out.
func ZeroDefault(kind, def string) bool {
	switch def {
	case "":
		return true
	case "false":
		return kind == FlagBool
	case "0":
		return kind == FlagInt || kind == FlagNumber
	case "0s":
		return kind == FlagDuration
	case "[]":
		return kind == FlagList
	}
	return false
}

// CanonicalFacts returns the facts sorted by path, each command's flags by
// name and inputs by flag, with nil collections made empty, so that two walks
// of one tree encode to the same bytes. The input is copied, never sorted in
// place.
func CanonicalFacts(in []Facts) []Facts {
	out := make([]Facts, len(in))
	copy(out, in)
	for i := range out {
		flags := append([]FlagFact{}, out[i].Flags...)
		sort.Slice(flags, func(a, b int) bool { return flags[a].Name < flags[b].Name })
		for j := range flags {
			if flags[j].Allowed != nil {
				flags[j].Allowed = append([]string{}, flags[j].Allowed...)
			}
		}
		out[i].Flags = flags
		inputs := append([]InputFact{}, out[i].Inputs...)
		sort.Slice(inputs, func(a, b int) bool { return inputs[a].Flag < inputs[b].Flag })
		out[i].Inputs = inputs
		if out[i].Subcommands != nil {
			subs := append([]string{}, out[i].Subcommands...)
			sort.Strings(subs)
			out[i].Subcommands = subs
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Path < out[b].Path })
	return out
}
