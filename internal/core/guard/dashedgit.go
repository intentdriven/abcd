package guard

import (
	"path"
	"sort"
	"strings"
)

// dashedGitPrefix opens the name of a program git runs for a subcommand. Each
// one is `git <sub>` by another door: `git-filter-repo` on PATH, and every
// `git-<sub>` in git's exec-path (`git --exec-path`), run directly.
const dashedGitPrefix = "git-"

// expandDashedGit appends, after every segment whose command is a `git-<sub>`
// program, the same command spelled `git <sub>`, so each git entry reads it
// without an entry per program, as expandGitAliases does for an alias. A program
// in command position with at least minArgs words after it is read; the name as
// `command -v` looks it up is not.
func (r Registry) expandDashedGit(segs []segment, minArgs int) []segment {
	out := make([]segment, 0, len(segs))
	for _, s := range segs {
		out = append(out, s)
		for _, a := range commandSites(s) {
			if len(s.tokens)-a.idx-1 < minArgs || lookupOnly(s.tokens, a.idx) {
				continue
			}
			for _, sub := range r.dashedGitSubcommands(s, a) {
				next := segment{tokens: make([]string, 0, len(s.tokens)+1), chain: s.chain}
				next.tokens = append(next.tokens, s.tokens[:a.idx]...)
				next.tokens = append(next.tokens, "git", sub.name)
				next.tokens = append(next.tokens, s.tokens[a.idx+1:]...)
				if s.globbed != nil {
					next.globbed = globAtRange(s.globbed, 0, a.idx)
					next.globbed = append(next.globbed, false, sub.glob)
					next.globbed = append(next.globbed, globAtRange(s.globbed, a.idx+1, len(s.tokens))...)
				}
				out = append(out, next)
			}
		}
	}
	return out
}

// dashedSub is one subcommand a program's word can name, and whether the name is
// still a pattern.
type dashedSub struct {
	name string
	glob bool
}

// dashedGitSubcommands returns the subcommands the command word at a names after
// `git-`: the one its basename spells, or, where a pattern stands in the prefix
// (`g?t-filter-repo`), each subcommand a registry entry names that the pattern
// can expand to. Only the text after the word's last substitution is fixed, so a
// name a substitution ends (`git-$(x)`) spells none.
func (r Registry) dashedGitSubcommands(s segment, a arrival) []dashedSub {
	tok := s.tokens[a.idx]
	globbed := !a.noglob && s.globAt(a.idx)
	base := path.Base(tok)
	if i := strings.LastIndexByte(base, unknownMark); i >= 0 {
		base = base[i+1:]
	}
	if len(base) > len(dashedGitPrefix) && strings.EqualFold(base[:len(dashedGitPrefix)], dashedGitPrefix) {
		return []dashedSub{{strings.ToLower(base[len(dashedGitPrefix):]), globbed}}
	}
	// A pattern is read where it fixes a letter of a name and has an argument
	// after it, which leaves a `case` label its own.
	if !globbed || isUnknown(tok) || len(s.tokens)-a.idx-1 < 1 || !patternFixesALetter(base) {
		return nil
	}
	var out []dashedSub
	pattern := strings.ToLower(base)
	for _, sub := range r.gitSubcommands() {
		if globMatches(pattern, dashedGitPrefix+sub) {
			out = append(out, dashedSub{name: sub})
		}
	}
	return out
}

// patternFixesALetter reports whether a glob pattern holds a letter outside its
// bracket expressions.
func patternFixesALetter(pattern string) bool {
	inBracket := false
	for _, c := range pattern {
		switch {
		case c == '[':
			inBracket = true
		case c == ']':
			inBracket = false
		case !inBracket && (c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'):
			return true
		}
	}
	return false
}

// gitSubcommands is every git subcommand a registry entry names, sorted.
func (r Registry) gitSubcommands() []string {
	seen := map[string]bool{}
	var out []string
	for _, e := range r.Entries {
		if sub := e.Pattern.Subcommand; strings.EqualFold(e.Pattern.Command, "git") && sub != "" && !seen[sub] {
			seen[sub] = true
			out = append(out, sub)
		}
	}
	sort.Strings(out)
	return out
}

// lookupOnly reports whether the word at idx is what a `command -v` or
// `command -V` before it looks up, rather than runs.
func lookupOnly(tokens []string, idx int) bool {
	sawLookup := false
	for i := idx - 1; i >= 0; i-- {
		w := tokens[i]
		if isUnknown(w) {
			return false
		}
		if !strings.HasPrefix(w, "-") {
			return sawLookup && path.Base(w) == "command"
		}
		sawLookup = sawLookup || strings.ContainsAny(w, "vV")
	}
	return false
}
