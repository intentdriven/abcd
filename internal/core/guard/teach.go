package guard

import (
	"fmt"
	"sort"
	"strings"
)

// teach.go is the registry's teaching-plane vocabulary (spc-16, "Two planes, one
// registry"; iss-151, ruling CK1). The rules loader builds its SHELL domain
// from these renderings, over the registry the guard enforces in the
// repository (the bundled entries and the repository's own), so what an agent
// is taught before shell-heavy work is what the guard would say at the moment
// of refusal, drawn from the one registry: an entry added or removed changes
// both planes with no second edit.

// maxTaughtValues caps how many operand words a description lists. An entry
// like rm-rf-root-or-home carries every spelling of the home directory, and a
// lesson that listed them all would bury the why; the first few name the
// hazard, and the count says the rest exist.
const maxTaughtValues = 6

// RepoMark is the provenance a lesson carries when its words are a
// repository's own rather than abcd's: an entry the repository's
// .abcd/guard.json added, or a bundled entry it reworded (ruling CK1). It
// follows the entry id, as a rules override's "(repo override)" follows the
// domain name, so whose words an agent is being taught is never invisible
// (GHSA-22f8-qf5r-gjgq).
const RepoMark = "(repo)"

// Lessons renders every entry's Lesson in entry-id order: the rules of the
// teaching plane, one per registry entry.
func (r Registry) Lessons() []string {
	return r.lessons(func(string, Entry) bool { return false })
}

// LessonsOver renders r's lessons as Lessons does, one per entry in id order,
// and marks with RepoMark every lesson bundled does not teach word for word:
// an entry bundled lacks, or one whose lesson a repository layer changed (its
// tier, pattern, why or successor). A change no lesson shows — a fixture —
// leaves the bundled lesson unmarked, because the words taught are still
// abcd's. The registry is the guard's registry in force for the repository, so
// the plane teaches exactly what the guard enforces there.
func (r Registry) LessonsOver(bundled Registry) []string {
	return r.lessons(func(id string, e Entry) bool {
		b, ok := bundled.Entries[id]
		if !ok {
			return true
		}
		b.ID = id
		return b.Lesson() != e.Lesson()
	})
}

// lessons renders every entry in id order, marking those repo reports true for.
func (r Registry) lessons(repo func(id string, e Entry) bool) []string {
	ids := make([]string, 0, len(r.Entries))
	for id := range r.Entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		e := r.Entries[id]
		e.ID = id
		out = append(out, e.lesson(repo(id, e)))
	}
	return out
}

// RecallTerms returns the command head of every entry, sorted and unique: the
// command followed by its subcommands, as it stands in command position
// (`git push`, `gh repo delete`, `rm`). They are the registry's own recall
// vocabulary for the teaching plane — narrow by construction, because the head
// carries the subcommand: `git push` recalls it and the bare word "push" does
// not.
func (r Registry) RecallTerms() []string {
	seen := map[string]bool{}
	var out []string
	for _, e := range r.Entries {
		head := e.Pattern.head()
		if head == "" || seen[head] {
			continue
		}
		seen[head] = true
		out = append(out, head)
	}
	sort.Strings(out)
	return out
}

// Lesson is the one-line rule an entry teaches: whether the guard refuses or
// warns, the entry id, the command it describes, the plain-language why, and
// the safe successor.
func (e Entry) Lesson() string { return e.lesson(false) }

// lesson is Lesson with the repository's provenance mark after the id when
// repo is set.
func (e Entry) lesson(repo bool) string {
	lead := "Refused by the guard"
	if e.Tier == TierWarn {
		lead = "Warned by the guard"
	}
	id := "(" + e.ID + ")"
	if repo {
		id += " " + RepoMark
	}
	return fmt.Sprintf("%s %s: %s. %s Instead: %s",
		lead, id, e.Pattern.Describe(), strings.TrimSpace(e.Why), strings.TrimSpace(e.Successor))
}

// head is the command with its subcommands, space-joined.
func (p Pattern) head() string {
	parts := []string{strings.TrimSpace(p.Command)}
	if p.Subcommand != "" {
		parts = append(parts, p.Subcommand)
		if p.Subcommand2 != "" {
			parts = append(parts, p.Subcommand2)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// Describe renders the pattern as the command it matches, in plain words: the
// command head, then each declared constraint in the order the matcher applies
// them. Every constraint an entry declares appears, so a lesson never
// describes a narrower or broader command than the guard refuses.
func (p Pattern) Describe() string {
	var b strings.Builder
	b.WriteString("`" + p.head() + "`")

	var with []string
	for _, group := range p.Flags {
		with = append(with, orList(alternatives(group)))
	}
	for _, fv := range p.FlagValues {
		with = append(with, orList(alternatives(fv.Flag))+" set to "+orList(quoteAll(fv.Values)))
	}
	for _, pre := range p.ArgPrefixes {
		with = append(with, "an operand starting `"+pre+"`")
	}
	switch {
	case p.MinOperands == 1:
		with = append(with, "an operand")
	case p.MinOperands > 1:
		with = append(with, fmt.Sprintf("at least %d operands", p.MinOperands))
	}
	if len(with) > 0 {
		b.WriteString(" with " + strings.Join(with, " and "))
	}

	if len(p.ArgsFrom) > 0 {
		var sources []string
		seen := map[string]bool{}
		for _, src := range p.ArgsFrom {
			if h := src.head(); !seen[h] {
				seen[h] = true
				sources = append(sources, "`"+h+"`")
			}
		}
		b.WriteString(" given pids printed by " + orList(sources))
	}
	for _, ap := range p.ArgPaths {
		b.WriteString(", on a `" + ap.Root + strings.Repeat("/*", max(ap.Segments-1, 0)) + "` path")
	}
	if len(p.ArgValues) > 0 {
		shown := quoteAll(p.ArgValues)
		if len(shown) > maxTaughtValues {
			rest := len(shown) - maxTaughtValues
			shown = append(shown[:maxTaughtValues:maxTaughtValues], fmt.Sprintf("%d more spellings like them", rest))
		}
		b.WriteString(", on " + orList(shown))
	}
	if p.AfterCD != nil && *p.AfterCD {
		b.WriteString(", after a `cd`, `pushd` or `popd` earlier in the same chain")
	}
	return b.String()
}

// alternatives splits a "|"-separated alternation group into quoted words,
// dropping empty alternatives (Validate refuses a group with none).
func alternatives(group string) []string {
	var out []string
	for _, alt := range strings.Split(group, "|") {
		if alt = strings.TrimSpace(alt); alt != "" {
			out = append(out, "`"+alt+"`")
		}
	}
	return out
}

func quoteAll(words []string) []string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = "`" + w + "`"
	}
	return out
}

// orList joins words as prose: "a", "a or b", "a, b or c".
func orList(words []string) string {
	switch len(words) {
	case 0:
		return ""
	case 1:
		return words[0]
	}
	return strings.Join(words[:len(words)-1], ", ") + " or " + words[len(words)-1]
}
