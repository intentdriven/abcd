package guard

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
// s is a git command whose own text sets core.hooksPath. Every place git can
// sit is read, and so is every word that can be its subcommand: the flag goes
// after each, which can only add flags to the readings the matcher takes.
func hooksPathRewrite(s segment, valueFlags []string) (segment, bool) {
	for _, site := range sitesNamed(s, "git") {
		ci := site.idx
		args := s.tokens[ci+1:]
		if !readGitConfig(s.tokens[:ci], args, valueFlags).hooksPath {
			continue
		}
		firsts := firstOperands(args, valueFlags)
		if len(firsts) == 0 {
			continue
		}
		after := map[int]bool{}
		for _, f := range firsts {
			after[ci+1+f] = true
		}
		var g []bool
		if s.globbed != nil {
			g = globAtRange(s.globbed, 0, len(s.tokens))
		}
		next := segment{tokens: make([]string, 0, len(s.tokens)+len(firsts)), chain: s.chain}
		if g != nil {
			next.globbed = make([]bool, 0, len(s.tokens)+len(firsts))
		}
		for i, tok := range s.tokens {
			next.tokens = append(next.tokens, tok)
			if g != nil {
				next.globbed = append(next.globbed, g[i])
			}
			if after[i] {
				next.tokens = append(next.tokens, noVerifyFlag)
				if g != nil {
					next.globbed = append(next.globbed, false)
				}
			}
		}
		return next, true
	}
	return segment{}, false
}
