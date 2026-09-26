package guard

import (
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

	// maxBangAliasDepth bounds how many `!`-alias bodies deep the pre-pass
	// re-enters itself. A bang body is a fresh command string that may declare
	// an alias of its own, so its segments come back through the pre-pass
	// (iss-2609020348038749); past this depth the guard has lost the thread, and
	// an alias rewrite it would have had to follow is a fail-closed block, the
	// execute-a-string family's depth posture (maxPayloadDepth).
	maxBangAliasDepth = maxPayloadDepth
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
// through shellInspect, the resulting segments run through expandPayloads in
// their own right, since a body may nest further, and then back through this
// pre-pass, since the body may be a git command declaring an alias of its own.
// That re-entry carries a depth budget (maxBangAliasDepth) and a repeat guard:
// a body already inspected on this line is not inspected again.
func (r Registry) expandGitAliases(segs []segment) ([]segment, []payloadSignal) {
	return r.expandGitAliasesAt(segs, r.gitValueFlags(), 0, map[string]bool{})
}

// expandGitAliasesAt is the pre-pass at one bang-body depth. seen is shared by
// every depth of one Check, so a repeated body costs nothing twice.
func (r Registry) expandGitAliasesAt(segs []segment, valueFlags []string, depth int, seen map[string]bool) ([]segment, []payloadSignal) {
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
		// Every place git can sit is read (sitesNamed), a name a substitution
		// prints included. A globbed name reaches this pre-pass for the same
		// reason it reaches matchSegment's command compare: bash expands the
		// word before exec, so `g?t -c alias.p='push --force' p` is git whenever
		// a file called `git` is there. Without the glob reading the two
		// compares disagreed, and the disagreement fell on the allow side — the
		// entry matched the rewrite that was never built.
		unreadRaised := false
		for _, site := range sitesNamed(s, "git") {
			ci := site.idx
			args := s.tokens[ci+1:]
			decls, unread := gitConfigDeclarations(s.tokens[:ci], args, valueFlags)
			if unread && !unreadRaised {
				unreadRaised = true
				signals = append(signals, gitConfigUnreadWarnSignal())
			}
			if len(decls) == 0 {
				continue
			}
			globs := s.globSlice(ci+1, len(s.tokens))
			for _, rw := range rewriteGitAliases(args, globs, decls, valueFlags) {
				if rw.shell != "" {
					if seen[rw.shell] {
						continue
					}
					seen[rw.shell] = true
					sig, psegs, inspectable := shellInspect(rw.shell)
					if !inspectable {
						// Warned, and what the body does spell is still read.
						signals = append(signals, sig)
						if len(psegs) == 0 {
							continue
						}
					}
					// The body may itself carry an execute-a-string layer; expanding
					// it here gives that its own depth budget, which is right — the
					// body is a fresh command string, not a deeper wrapping of this
					// one.
					psegs, psigs := expandPayloads(psegs)
					signals = append(signals, psigs...)
					// Each body gets its OWN disjoint chain range, the way
					// expandPayloads gives each payload one: a body is a separate
					// command string, so a `cd` in one must not read as preceding
					// an `rm` in the next. chainMax is the running maximum, so the
					// range this body takes is never handed out again.
					for i := range psegs {
						psegs[i].chain += chainMax + 1
					}
					// The body re-enters the pre-pass one level deeper. At the
					// budget it is still checked as written, and an alias rewrite
					// in it — one the guard would have had to follow — is refused
					// instead.
					if depth+1 > maxBangAliasDepth {
						if r.anyAliasRewrite(psegs, valueFlags) {
							signals = append(signals, bangAliasDepthBlockSignal())
						}
					} else {
						var bsigs []payloadSignal
						psegs, bsigs = r.expandGitAliasesAt(psegs, valueFlags, depth+1, seen)
						signals = append(signals, bsigs...)
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
					tokens: append(append([]string(nil), s.tokens[:ci+1]...), rw.args...),
					chain:  s.chain,
				}
				// The glob record travels onto the rewrite from BOTH ends: from
				// the alias body's own words, and from the leading tokens, which
				// is where a glob-spelled `g?t` sits. Carrying only the body's
				// record left the rewritten segment unglobbed, so matchSegment
				// compared `g?t` with `git` literally and the rewrite this
				// pre-pass had just built matched nothing.
				if lead := globAtRange(s.globbed, 0, ci+1); rw.globs != nil || anyGlob(lead) {
					next.globbed = append(lead, rw.globs...)
				}
				out = append(out, next)
			}
		}
	}

	return append(out, bang...), signals
}

// anyAliasRewrite reports whether some segment is a git command whose operand 0
// names an alias the segment itself declares — a rewrite the pre-pass would
// follow if it were allowed to.
func (r Registry) anyAliasRewrite(segs []segment, valueFlags []string) bool {
	for _, s := range segs {
		for _, site := range sitesNamed(s, "git") {
			args := s.tokens[site.idx+1:]
			decls, _ := gitConfigDeclarations(s.tokens[:site.idx], args, valueFlags)
			if len(decls) == 0 {
				continue
			}
			if len(rewriteGitAliases(args, s.globSlice(site.idx+1, len(s.tokens)), decls, valueFlags)) > 0 {
				return true
			}
		}
	}
	return false
}

// bangAliasDepthBlockSignal is the fail-closed verdict for an alias declared in
// a `!`-alias body nested past maxBangAliasDepth: the command git would run is
// one the guard stopped following.
func bangAliasDepthBlockSignal() payloadSignal {
	return payloadSignal{
		id:      gitConfigEntryID,
		verdict: VerdictBlock,
		family:  familyGitConfig,
		reason: "This git command nests `!` aliases deeper than the guard follows, and the innermost declares an alias of its own, " +
			"so the command git would finally run is one the guard has not checked.",
		successor: "Spell the git command out, or flatten the aliases into one `git -c alias.x='<body>' x`, " +
			"so the guard checks the command that actually runs.",
	}
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
// points at configuration whose body the guard cannot read. It is readGitConfig
// narrowed to what the alias pre-pass reads.
func gitConfigDeclarations(prefix, args, valueFlags []string) (map[string]string, bool) {
	c := readGitConfig(prefix, args, valueFlags)
	if len(c.aliases) == 0 {
		return nil, c.unread
	}
	return c.aliases, c.unread
}

// gitConfigRead is what one git segment's own text sets in configuration: the
// alias bodies it declares, whether it points at configuration whose body the
// guard cannot read, and whether it points core.hooksPath anywhere.
type gitConfigRead struct {
	aliases   map[string]string
	unread    bool
	hooksPath bool
}

// readGitConfig reads the configuration one git segment sets in its own text.
//
// prefix is the tokens before command position (the environment assignments and
// wrapper words commandOf steps); args is everything after it. Only the
// arguments BEFORE operand 0 are read for `-c`/`--config-env`, because that is
// where git's own parser reads them — `git log -c` past the subcommand is a
// combined-diff flag, not a config setting.
func readGitConfig(prefix, args, valueFlags []string) gitConfigRead {
	c := gitConfigRead{aliases: map[string]string{}}

	env := map[string]string{}
	for _, tok := range prefix {
		if !isAssignment(tok) {
			continue
		}
		eq := strings.IndexByte(tok, '=')
		env[tok[:eq]] = tok[eq+1:]
	}
	if _, ok := env["GIT_CONFIG_GLOBAL"]; ok {
		c.unread = true
	}
	if _, ok := env["GIT_CONFIG_SYSTEM"]; ok {
		c.unread = true
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
			c.add(key, v)
		}
	}
	if params, ok := env["GIT_CONFIG_PARAMETERS"]; ok && isUnknown(params) {
		c.add(params, "")
	} else if ok {
		for k, v := range parseConfigParameters(params) {
			c.add(k, v)
		}
	}

	// Every word before operand 0 is read, in whichever reading puts operand 0
	// furthest along (firstOperandLimit); an unknown word that can be `-c` or
	// `--config-env` is read as one, and its next word as the setting.
	limit := firstOperandLimit(args, valueFlags)
	for i := 0; i < limit; i++ {
		arg := args[i]
		switch {
		case arg == "-c" || (isUnknown(arg) && flagCouldBe(arg, "-c")):
			if i+1 < len(args) {
				k, v, ok := strings.Cut(args[i+1], "=")
				switch {
				case ok:
					c.add(k, v)
				case isUnknown(k):
					// No `=` it spells, but its output may hold one.
					c.add(k, "")
				}
			}
		case arg == "--config-env", strings.HasPrefix(arg, "--config-env="),
			isUnknown(arg) && flagCouldBe(arg, "--config-env"):
			spec, glued := strings.CutPrefix(arg, "--config-env=")
			if !glued {
				if i+1 >= len(args) {
					continue
				}
				spec = args[i+1]
			}
			k, name, ok := strings.Cut(spec, "=")
			if !ok || isUnknown(k) {
				if isUnknown(k) {
					c.add(k, "")
				}
				continue
			}
			v, set := env[name]
			if !set {
				// The body is in a variable the command line does not set, so
				// it comes from the ambient environment: visible directive,
				// unreadable value. For the hooks path the directive is all
				// that matters — any value moves the hooks.
				switch key := strings.ToLower(strings.TrimSpace(k)); {
				case strings.HasPrefix(key, aliasPrefix):
					c.unread = true
				case key == hooksPathKey:
					c.hooksPath = true
				}
				continue
			}
			c.add(k, v)
		}
	}
	return c
}

// add files one `key=value` config setting: an alias declaration is stored
// under its folded name, a key that pulls configuration in from a file marks
// the segment unreadable, and the hooks path is noted whatever its value.
func (c *gitConfigRead) add(key, value string) {
	// A key a substitution prints (`-c $(cat cfg)`) is any key (unknown.go):
	// the hooks path, and an alias the guard cannot read.
	if isUnknown(key) {
		c.hooksPath, c.unread = true, true
		return
	}
	k := strings.ToLower(strings.TrimSpace(key))
	switch {
	case strings.HasPrefix(k, aliasPrefix):
		c.aliases[strings.TrimPrefix(k, aliasPrefix)] = value
		// A body a substitution prints is one the guard cannot read.
		if isUnknown(value) {
			c.unread = true
		}
	case k == "include.path", strings.HasPrefix(k, "includeif."):
		c.unread = true
	case k == hooksPathKey:
		c.hooksPath = true
	}
}

// parseConfigParameters decodes GIT_CONFIG_PARAMETERS, the variable git uses to
// pass `-c` settings to its own subprocesses and which it reads back on every
// invocation. Both shipped spellings are handled: the original `'key=value'` and
// the `'key'='value'` form git has written since 2.31. Values are single-quoted,
// and a quote inside one is escaped by closing, escaping and reopening:
//
//	'\''
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
// unquoted contents and the index just past the closing quote. A literal quote
// inside one is git's close-escape-reopen spelling (see parseConfigParameters).
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

// aliasRewrite is one command git can run once operand 0's alias is expanded:
// the arguments and their glob record, or, for a `!` alias, the shell command
// git hands to a shell.
type aliasRewrite struct {
	args  []string
	globs []bool
	shell string
}

// maxAliasRewrites bounds the rewrites one git command yields. Each reading of
// where operand 0 sits can name a different alias; past the bound the rest are
// not built, and a rewrite the guard would have had to follow is the bang
// depth's refusal's shape, one level up (the readings are an attacker's, not
// an everyday command's).
const maxAliasRewrites = 16

// rewriteGitAliases returns every command git can run once operand 0's alias is
// expanded — the flags that preceded it, the body's words, then the arguments
// that followed — following at most maxAliasHops of nesting, and every reading
// of which word is operand 0 (operandReadings). A `!` alias is not a
// subcommand: git runs the rest of the line through a shell, so the rewrite is
// that shell command. None is returned when operand 0 names no alias in any
// reading.
//
// The body is split on whitespace. git splits it with its own quote-aware
// splitter, so a body carrying a quoted space (`alias.c='commit -m "a b"'`)
// splits into more words here than git would produce — a floor, and one that
// affects the words of a body an author already controls, not whether the
// rewrite happens.
func rewriteGitAliases(args []string, globs []bool, decls map[string]string, valueFlags []string) []aliasRewrite {
	var out []aliasRewrite
	var walk func(cur []string, curGlob []bool, hop int, seen map[string]bool)
	walk = func(cur []string, curGlob []bool, hop int, seen map[string]bool) {
		if hop >= maxAliasHops {
			return
		}
		for _, i := range firstOperands(cur, valueFlags) {
			if len(out) >= maxAliasRewrites {
				return
			}
			name := strings.ToLower(cur[i])
			body, isAlias := decls[name]
			if !isAlias || seen[name] {
				continue
			}
			if strings.HasPrefix(body, "!") {
				// git runs the rest of the line as the shell command's own
				// arguments, so they belong in the payload the guard reads.
				out = append(out, aliasRewrite{shell: strings.TrimSpace(strings.Join(append([]string{strings.TrimPrefix(body, "!")}, cur[i+1:]...), " "))})
				continue
			}
			fields := strings.Fields(body)
			if len(fields) == 0 {
				continue
			}
			next := make([]string, 0, len(cur)+len(fields)-1)
			next = append(next, cur[:i]...)
			next = append(next, fields...)
			next = append(next, cur[i+1:]...)
			// The body's words come from a config value, which git does not
			// expand, so they are never patterns; the surrounding tokens keep the
			// record they arrived with.
			var nextGlob []bool
			if curGlob != nil {
				nextGlob = make([]bool, 0, len(next))
				nextGlob = append(nextGlob, globAtRange(curGlob, 0, i)...)
				nextGlob = append(nextGlob, make([]bool, len(fields))...)
				nextGlob = append(nextGlob, globAtRange(curGlob, i+1, len(cur))...)
			}
			out = append(out, aliasRewrite{args: next, globs: nextGlob})
			nextSeen := map[string]bool{name: true}
			for k := range seen {
				nextSeen[k] = true
			}
			walk(next, nextGlob, hop+1, nextSeen)
		}
	}
	walk(args, globs, 0, map[string]bool{})
	return out
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
