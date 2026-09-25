package guard

import (
	"path"
	"strings"
)

// hooksPathKey is git's core.hooksPath, folded: the directory git runs a
// repository's hooks from.
const hooksPathKey = "core.hookspath"

// noVerifyFlag is the flag a hooks-path override is read as carrying.
const noVerifyFlag = "--no-verify"

// expandHooksPathOverrides appends, for every git segment that points
// core.hooksPath anywhere in its own text, the same command with --no-verify
// inserted directly after its subcommand, placed right after its source and
// keeping its chain. Moving the hooks for one command skips the repository's
// hooks exactly as the flag does, and the matcher stepped the `-c` value over
// unread, so the no-verify entries never saw it (iss-2609251640464212). Any
// value counts: the guard cannot tell a directory of real hooks from an empty
// one, and the repository's own hooks are the ones the entries protect. The
// rewrite is offered to every entry, so only the subcommands an entry names —
// commit and push in the bundled set — are refused.
//
// It runs after the alias pre-pass, so the command an alias expands to is read
// too, and after the payload expansion, so a git command inside `sh -c` is.
func expandHooksPathOverrides(segs []segment, valueFlags []string) []segment {
	var out []segment
	for i, s := range segs {
		next, ok := hooksPathRewrite(s, valueFlags)
		if !ok {
			if out != nil {
				out = append(out, s)
			}
			continue
		}
		if out == nil {
			out = append(make([]segment, 0, len(segs)+1), segs[:i]...)
		}
		out = append(out, s, next)
	}
	if out == nil {
		return segs
	}
	return out
}

// hooksPathRewrite returns s with --no-verify inserted after its subcommand when
// s is a git command whose own text sets core.hooksPath.
func hooksPathRewrite(s segment, valueFlags []string) (segment, bool) {
	ci, noglob := commandIndex(s)
	if ci < 0 {
		return segment{}, false
	}
	base := path.Base(s.tokens[ci])
	if !strings.EqualFold(base, "git") &&
		!(!noglob && s.globAt(ci) && globMatches(strings.ToLower(base), "git")) {
		return segment{}, false
	}
	args := s.tokens[ci+1:]
	if !readGitConfig(s.tokens[:ci], args, valueFlags).hooksPath {
		return segment{}, false
	}
	idx := operandIndexes(args, valueFlags)
	if len(idx) == 0 {
		return segment{}, false
	}
	at := ci + 1 + idx[0] + 1
	tokens := make([]string, 0, len(s.tokens)+1)
	tokens = append(append(append(tokens, s.tokens[:at]...), noVerifyFlag), s.tokens[at:]...)
	next := segment{tokens: tokens, chain: s.chain}
	if s.globbed != nil {
		g := globAtRange(s.globbed, 0, len(s.tokens))
		next.globbed = append(append(append(make([]bool, 0, len(g)+1), g[:at]...), false), g[at:]...)
	}
	return next, true
}
