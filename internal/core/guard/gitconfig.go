package guard

import (
	"path"
	"sort"
	"strings"
)

// git rewrites its own subcommand from configuration handed to it in the command
// line, and the matcher's operand walk was built to step exactly those tokens
// over without reading them.
//
// `git -c alias.p='push --force' p origin main` runs a force push. `-c` and
// `--config-env` sit in every git entry's value_flags (gh-299, so the value is
// not mistaken for the subcommand), and `GIT_CONFIG_*` arrives as an environment
// assignment prefix, which commandOf steps for the same reason. The value was
// therefore consumed and discarded, the attacker-chosen alias NAME reached
// operand 0, and the subcommand compare missed (GHSA-m2r8-fx7r-rq34).
//
// Everything needed to decide this is IN THE STRING the matcher already holds,
// and it already tokenises the exact tokens: this pre-pass reads the values it
// used to step, and where operand 0 names an alias declared in the same segment
// it appends the command git would actually run, so Tier 1 matches it with the
// entries that already exist. No new Pattern field, no new enumeration — two
// flag names and four environment names.
//
// What the string CANNOT decide is where adr-42's "permanently invisible"
// residual starts: `GIT_CONFIG_GLOBAL`, `GIT_CONFIG_SYSTEM`, `-c include.path=`
// and `includeIf` deliver the alias body in a FILE, and `--config-env` can name
// a variable the command line does not set. The directive is visible even when
// the body is not, so each raises a loud warn under the guard's own reserved id
// rather than a silent allow.

const (
	// gitConfigEntryID is the reserved id a git command pointing at
	// configuration the guard cannot read is reported under. Like the other
	// reserved ids it names a verdict the Pattern language cannot express, so no
	// registry entry may claim it and it must never index Registry.Entries.
	gitConfigEntryID = "git-config-rewrite-unread"

	familyGitConfig = "git config"

	// maxAliasHops bounds the alias chain the pre-pass follows. An alias body may
	// name another alias, and git resolves the chain itself; the guard follows a
	// few hops and stops, because the work is per segment on the PreToolUse path
	// and a cycle must not be able to spend it (a repeated name also breaks the
	// loop, so the cap is a second floor rather than the only one).
	maxAliasHops = 4
)

// aliasPrefix is the config section a subcommand rewrite can come from. git
// config section and variable names are case-insensitive, so declarations are
// folded before they are stored and operand 0 is folded before it is looked up:
// `-c ALIAS.P=…` rewrites `git p` on a real git, and a case-sensitive compare
// here would be a silent allow for the cost of one shift key.
const aliasPrefix = "alias."

// expandGitAliases appends, for every segment whose command is git and whose
// operand 0 names an alias the same segment declares, the command git would
// actually run — inserted directly after its source segment, keeping its chain,
// so an `after_cd` entry reads the rewrite exactly where the original stood.
//
// It runs AFTER expandPayloads, so a git command inside an `sh -c` payload is
// reached too. A `!`-prefixed alias body is not a subcommand at all — git hands
// it to a shell — so it is read as an execute-a-string payload: tokenised
// through shellInspect, and the resulting segments run through expandPayloads in
// their own right, since a body may nest further.
func (r Registry) expandGitAliases(segs []segment) ([]segment, []payloadSignal) {
	valueFlags := r.gitValueFlags()
	out := make([]segment, 0, len(segs))
	var signals []payloadSignal
	var bang []segment

	// chainMax is the running maximum chain number across everything handed out
	// so far — the input segments, then each bang body as it is inspected. A
	// bang body is a separate command string, so it is offset into a disjoint
	// range for the same reason expandPayloads offsets a payload's: an
	// `after_cd` entry must not read across the boundary. One shared offset for
	// every body was not enough, because each body's own tokenize numbers its
	// chains from 0, so two bodies on one line collided on chain 0.
	chainMax := 0
	for _, s := range segs {
		if s.chain > chainMax {
			chainMax = s.chain
		}
	}

	for _, s := range segs {
		out = append(out, s)
		ci, noglob := commandIndex(s)
		if ci < 0 {
			continue
		}
		// A globbed name reaches this pre-pass for the same reason it reaches
		// matchSegment's command compare: bash expands the word before exec, so
		// `g?t -c alias.p='push --force' p` is git whenever a file called `git`
		// is there. Without the glob reading the two compares disagreed, and the
		// disagreement fell on the allow side — the entry matched the rewrite
		// that was never built.
		base := path.Base(s.tokens[ci])
		if !strings.EqualFold(base, "git") &&
			!(!noglob && s.globAt(ci) && globMatches(strings.ToLower(base), "git")) {
			continue
		}
		args := s.tokens[ci+1:]
		decls, unread := gitConfigDeclarations(s.tokens[:ci], args, valueFlags)
		if unread {
			signals = append(signals, gitConfigUnreadWarnSignal())
		}
		if len(decls) == 0 {
			continue
		}
		globs := s.globSlice(ci+1, len(s.tokens))
		rewritten, globbed, shell, ok := rewriteGitAlias(args, globs, decls, valueFlags)
		if !ok {
			continue
		}
		if shell != "" {
			sig, psegs, inspectable := shellInspect(shell)
			if !inspectable {
				signals = append(signals, sig)
				continue
			}
			// The body may itself carry an execute-a-string layer; expanding it
			// here gives that its own depth budget, which is right — the body is
			// a fresh command string, not a deeper wrapping of this one.
			psegs, psigs := expandPayloads(psegs)
			signals = append(signals, psigs...)
			// Each body gets its OWN disjoint chain range, the way
			// expandPayloads gives each payload one: a body is a separate
			// command string, so a `cd` in one must not read as preceding an
			// `rm` in the next. chainMax is the running maximum, so the range
			// this body takes is never handed out again.
			for i := range psegs {
				psegs[i].chain += chainMax + 1
			}
			for _, ps := range psegs {
				if ps.chain > chainMax {
					chainMax = ps.chain
				}
			}
			bang = append(bang, psegs...)
			continue
		}
		next := segment{
			tokens: append(append([]string(nil), s.tokens[:ci+1]...), rewritten...),
			chain:  s.chain,
		}
		// The glob record travels onto the rewrite from BOTH ends: from the
		// alias body's own words, and from the leading tokens, which is where a
		// glob-spelled `g?t` sits. Carrying only the body's record left the
		// rewritten segment unglobbed, so matchSegment compared `g?t` with
		// `git` literally and the rewrite this pre-pass had just built matched
		// nothing.
		if lead := globAtRange(s.globbed, 0, ci+1); globbed != nil || anyGlob(lead) {
			next.globbed = append(lead, globbed...)
		}
		out = append(out, next)
	}

	return append(out, bang...), signals
}

// gitValueFlags is the union of the value_flags the registry's git entries
// declare, plus the two config-carrying flags this pre-pass reads. Taking it
// from the registry rather than from a second hand-kept list is what stops the
// pre-pass and matchSegment disagreeing about where operand 0 is — a
// disagreement that would show up as a rewrite of the wrong token.
func (r Registry) gitValueFlags() []string {
	out := []string{"-c", "--config-env"}
	for _, e := range r.Entries {
		if !strings.EqualFold(e.Pattern.Command, "git") {
			continue
		}
		for _, f := range e.Pattern.ValueFlags {
			if !containsString(out, f) {
				out = append(out, f)
			}
		}
	}
	sort.Strings(out)
	return out
}

// gitConfigDeclarations collects the alias bodies one git segment declares in its
// own text, keyed by the folded alias name, and reports whether the segment also
// points at configuration whose body the guard cannot read.
//
// prefix is the tokens before command position (the environment assignments and
// wrapper words commandOf steps); args is everything after it. Only the
// arguments BEFORE operand 0 are read for `-c`/`--config-env`, because that is
// where git's own parser reads them — `git log -c` past the subcommand is a
// combined-diff flag, not a config setting.
func gitConfigDeclarations(prefix, args, valueFlags []string) (map[string]string, bool) {
	decls := map[string]string{}
	unread := false

	env := map[string]string{}
	for _, tok := range prefix {
		if !isAssignment(tok) {
			continue
		}
		eq := strings.IndexByte(tok, '=')
		env[tok[:eq]] = tok[eq+1:]
	}
	if _, ok := env["GIT_CONFIG_GLOBAL"]; ok {
		unread = true
	}
	if _, ok := env["GIT_CONFIG_SYSTEM"]; ok {
		unread = true
	}
	// The GIT_CONFIG_COUNT/KEY_n/VALUE_n triple. The keys present are read
	// rather than the count trusted: a count that undersells what is set would
	// otherwise hide a declaration from the guard while git still applies it.
	for name, key := range env {
		n, ok := strings.CutPrefix(name, "GIT_CONFIG_KEY_")
		if !ok {
			continue
		}
		if v, ok := env["GIT_CONFIG_VALUE_"+n]; ok {
			addConfigPair(decls, &unread, key, v)
		}
	}
	if params, ok := env["GIT_CONFIG_PARAMETERS"]; ok {
		for k, v := range parseConfigParameters(params) {
			addConfigPair(decls, &unread, k, v)
		}
	}

	limit := len(args)
	if idx := operandIndexes(args, valueFlags); len(idx) > 0 {
		limit = idx[0]
	}
	for i := 0; i < limit; i++ {
		arg := args[i]
		switch {
		case arg == "-c":
			if i+1 < len(args) {
				k, v, ok := strings.Cut(args[i+1], "=")
				if ok {
					addConfigPair(decls, &unread, k, v)
				}
			}
		case arg == "--config-env", strings.HasPrefix(arg, "--config-env="):
			spec := strings.TrimPrefix(arg, "--config-env=")
			if spec == "--config-env" {
				if i+1 >= len(args) {
					continue
				}
				spec = args[i+1]
			}
			k, name, ok := strings.Cut(spec, "=")
			if !ok {
				continue
			}
			v, set := env[name]
			if !set {
				// The body is in a variable the command line does not set, so
				// it comes from the ambient environment: visible directive,
				// unreadable value.
				if strings.HasPrefix(strings.ToLower(k), aliasPrefix) {
					unread = true
				}
				continue
			}
			addConfigPair(decls, &unread, k, v)
		}
	}
	if len(decls) == 0 {
		return nil, unread
	}
	return decls, unread
}

// addConfigPair files one `key=value` config setting: an alias declaration is
// stored under its folded name, and a key that pulls configuration in from a
// file marks the segment unreadable.
func addConfigPair(decls map[string]string, unread *bool, key, value string) {
	k := strings.ToLower(strings.TrimSpace(key))
	switch {
	case strings.HasPrefix(k, aliasPrefix):
		decls[strings.TrimPrefix(k, aliasPrefix)] = value
	case k == "include.path", strings.HasPrefix(k, "includeif."):
		*unread = true
	}
}

// parseConfigParameters decodes GIT_CONFIG_PARAMETERS, the variable git uses to
// pass `-c` settings to its own subprocesses and which it reads back on every
// invocation. Both shipped spellings are handled: the original `'key=value'` and
// the `'key'='value'` form git has written since 2.31. Values are single-quoted
// with `'\”` escaping their own quote.
func parseConfigParameters(v string) map[string]string {
	out := map[string]string{}
	i := 0
	for i < len(v) {
		if v[i] == ' ' || v[i] == '\t' {
			i++
			continue
		}
		if v[i] != '\'' {
			// Not a shape git writes; skip to the next separator rather than
			// guess at it.
			for i < len(v) && v[i] != ' ' && v[i] != '\t' {
				i++
			}
			continue
		}
		first, next, ok := readSingleQuoted(v, i)
		if !ok {
			return out
		}
		i = next
		if i < len(v) && v[i] == '=' {
			second, after, ok := readSingleQuoted(v, i+1)
			if !ok {
				return out
			}
			out[first] = second
			i = after
			continue
		}
		if k, val, ok := strings.Cut(first, "="); ok {
			out[k] = val
		}
	}
	return out
}

// readSingleQuoted reads the single-quoted run starting at v[i], returning its
// unquoted contents and the index just past the closing quote. `'\”` is git's
// escape for a literal quote inside one.
func readSingleQuoted(v string, i int) (string, int, bool) {
	if i >= len(v) || v[i] != '\'' {
		return "", i, false
	}
	i++
	var b strings.Builder
	for i < len(v) {
		if v[i] != '\'' {
			b.WriteByte(v[i])
			i++
			continue
		}
		if strings.HasPrefix(v[i:], `'\''`) {
			b.WriteByte('\'')
			i += 4
			continue
		}
		return b.String(), i + 1, true
	}
	return "", i, false
}

// rewriteGitAlias returns the arguments git would run once operand 0's alias is
// expanded — the flags that preceded it, the body's words, then the arguments
// that followed — following at most maxAliasHops of nesting. shell is non-empty
// when the body is a `!` alias, which git runs through a shell rather than as a
// subcommand; ok is false when operand 0 names no alias and nothing is rewritten.
//
// The body is split on whitespace. git splits it with its own quote-aware
// splitter, so a body carrying a quoted space (`alias.c='commit -m "a b"'`)
// splits into more words here than git would produce — a floor, and one that
// affects the words of a body an author already controls, not whether the
// rewrite happens.
func rewriteGitAlias(args []string, globs []bool, decls map[string]string, valueFlags []string) (rewritten []string, globbed []bool, shell string, ok bool) {
	cur := args
	curGlob := globs
	seen := map[string]bool{}
	for hop := 0; hop < maxAliasHops; hop++ {
		idx := operandIndexes(cur, valueFlags)
		if len(idx) == 0 {
			break
		}
		i := idx[0]
		name := strings.ToLower(cur[i])
		body, isAlias := decls[name]
		if !isAlias || seen[name] {
			break
		}
		seen[name] = true
		if strings.HasPrefix(body, "!") {
			// git runs the rest of the line as the shell command's own
			// arguments, so they belong in the payload the guard reads.
			return nil, nil, strings.TrimSpace(strings.Join(append([]string{strings.TrimPrefix(body, "!")}, cur[i+1:]...), " ")), true
		}
		fields := strings.Fields(body)
		if len(fields) == 0 {
			break
		}
		next := make([]string, 0, len(cur)+len(fields)-1)
		next = append(next, cur[:i]...)
		next = append(next, fields...)
		next = append(next, cur[i+1:]...)
		// The body's words come from a config value, which git does not expand,
		// so they are never patterns; the surrounding tokens keep the record
		// they arrived with.
		var nextGlob []bool
		if curGlob != nil {
			nextGlob = make([]bool, 0, len(next))
			nextGlob = append(nextGlob, globAtRange(curGlob, 0, i)...)
			nextGlob = append(nextGlob, make([]bool, len(fields))...)
			nextGlob = append(nextGlob, globAtRange(curGlob, i+1, len(cur))...)
		}
		cur, curGlob = next, nextGlob
		ok = true
	}
	if !ok {
		return nil, nil, "", false
	}
	return cur, curGlob, "", true
}

// globAtRange returns globs[lo:hi] padded to that length, so a record shorter
// than the tokens it describes cannot panic the rewrite.
func globAtRange(globs []bool, lo, hi int) []bool {
	out := make([]bool, 0, hi-lo)
	for i := lo; i < hi; i++ {
		if i < len(globs) {
			out = append(out, globs[i])
			continue
		}
		out = append(out, false)
	}
	return out
}

// anyGlob reports whether a record marks any of its tokens as a pattern.
func anyGlob(globs []bool) bool {
	for _, g := range globs {
		if g {
			return true
		}
	}
	return false
}

// gitConfigUnreadWarnSignal is the loud-warn verdict for a git command that
// points at configuration the guard cannot read. It is a WARN and not a block
// because the directive is not itself evidence of a hazard — `GIT_CONFIG_GLOBAL`
// pointing at a scratch config is ordinary work — and not silence because git
// rewrites its own subcommand from what it finds there, so the command that runs
// need not be the command written.
func gitConfigUnreadWarnSignal() payloadSignal {
	return payloadSignal{
		id:      gitConfigEntryID,
		verdict: VerdictWarn,
		family:  familyGitConfig,
		reason: "This git command points at configuration the guard cannot read — a config file, an include, or an environment variable the command line does not set — " +
			"and git rewrites its own subcommand from an alias found there, so the command that runs need not be the command written here.",
		successor: "Spell the git command out, or put the alias body in the command line (`git -c alias.x='<body>' x`), " +
			"so the guard checks the command that actually runs.",
	}
}
