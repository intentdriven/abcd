package guard

import (
	"path"
	"strings"
)

// The exec-string family — a command STRING run by a shell that the carrying
// program is not (adr-42 decision 6).
//
// `su -c CMD`, `runuser -c CMD`, `script -c CMD FILE` and `flock FILE -c CMD`
// all hand a string to a shell, so the payload is grammar the tokenizer already
// reads. But none of them is a shell, so they fit neither isShellFamily nor the
// wrapper walk, and they were the last three of the ten silent allows iss-272
// recorded. Tier 2 cannot reach them either: the payload is one opaque token, so
// no speculative start can see inside it.
//
// This is a TABLE and not a generalisation of shellCPayload, which is the same
// decision spelled two ways. shellCPayload matches short option clusters;
// generalising it would resolve `su -c` for free and return shellNone for
// `su --command`, `su --command=`, `su --session-command`, `runuser --command`,
// `script --command` and `flock --command` — six new SILENT allows, not even the
// shellUnresolved fail-safe. Widening a helper until it covers a second grammar
// is how the first three of these holes were dug.
//
// Two semantics differ from `sh -c` and are the reason the code is separate:
//
//   - the payload is the token IMMEDIATELY following the flag (getopt
//     required_argument), not the POSIX first non-option operand shellOperand
//     implements;
//   - the flag may sit AFTER a mandatory operand — `flock /tmp/lock -c CMD`,
//     `su bob -c CMD` — so the scan cannot stop at the first non-flag token the
//     way an option walk usually does.
//
// What the payload is parsed AS is a smaller claim than it looks. `flock -c`
// runs /bin/sh; `script -c` and `runuser -c` use $SHELL or the target user's
// shell; `su -c` carries no guarantee at all — it runs the TARGET user's login
// shell, overridable with -s, so a csh or fish login shell is not POSIX grammar.
// That costs FALSE NEGATIVES only: a mis-parse yields a non-match, never a false
// block. It is recorded in the guard's scope statement rather than papered over
// with a claim that the family parses uniformly.

// execStringVerbs maps each verb to the flags that carry its command string.
// Every spelling is listed explicitly, because the missing long spelling is the
// defect this table exists to avoid.
var execStringVerbs = map[string][]string{
	"su":      {"-c", "--command", "--session-command"},
	"runuser": {"-c", "--command", "--session-command"},
	"script":  {"-c", "--command"},
	"flock":   {"-c", "--command"},
}

// execStringCommandOperand names, per verb, how many leading non-flag operands
// the verb owns before the tokens start belonging to the COMMAND it launches.
// Past that point a `-c` is the launched command's flag, not this verb's.
//
// Only flock has one, and getting it wrong is not academic:
// `flock /tmp/lock /bin/echo -c "git push --force origin main"` hands `-c` and
// the string to /bin/echo as ordinary arguments — verified, it prints them — so
// reading that string as a payload is a FALSE BLOCK on a command that pushes
// nothing. Worse, classifying the segment as payload-carrying also switched the
// Tier 2 fail-safe off for it, which made a `flock` prefix a one-token off switch
// for anything behind it.
//
// The others genuinely permute and are left unbounded on purpose:
// `su root /bin/echo -c 'echo SEVEN'` prints SEVEN, so su parsed the `-c` after
// its operand and ran the string. su's trailing operands are positional
// parameters handed to that shell, not a command line of their own.
var execStringCommandOperand = map[string]int{
	// `flock [options] FILE -c CMD` or `flock [options] FILE CMD [args...]`.
	// After FILE, the next non-flag token opens the launched command.
	"flock": 1,
}

// execStringOtherValueFlags names each verb's OWN flags that consume the
// following token, so the scan steps over a value instead of reading it as the
// payload: `su -s /bin/sh -c <hazard>` carries the shell in -s, not the command.
//
// Getting this wrong is a false negative, never a false block: an unlisted value
// flag means the scan reads its value as an ordinary token and moves on.
var execStringOtherValueFlags = map[string][]string{
	"su":      {"-s", "--shell", "-g", "--group", "-G", "--supp-group", "-w", "--whitelist-environment"},
	"runuser": {"-s", "--shell", "-g", "--group", "-G", "--supp-group", "-u", "--user", "-w", "--whitelist-environment"},
	"script":  {"-o", "--output-limit", "-I", "--log-in", "-O", "--log-out", "-B", "--log-io", "-T", "--log-timing", "-m", "--logging-format"},
	"flock":   {"-w", "--timeout", "--wait", "-E", "--conflict-exit-code"},
}

// execStringPayload walks a segment's leading wrapper chain and returns the
// command string carried by the first exec-string verb it reaches.
//
// The walk mirrors splitStringValue's: assignments and reserved words are
// stepped, and a wrapper is stepped WITH its own arguments, so `sudo su -c
// <hazard>` and `nice runuser -c <hazard>` are reached rather than lost at the
// first token.
//
// resolved is false when a payload flag is present but its value is not — a
// `-c` at the end of the line — which the caller turns into a loud warn rather
// than a silent allow.
func execStringPayload(tokens []string) (verb, value string, resolved, found bool) {
	i := 0
	for i < len(tokens) {
		tok := tokens[i]
		if steppedBeforeCommand(tok) {
			i++
			continue
		}
		// Folded to lower case: `SU -c` / `FLOCK … -c` resolve to the real binary
		// on a case-insensitive filesystem, so a case-varied exec-string verb must
		// be read exactly as its lowercase spelling is (gh-315).
		name := strings.ToLower(path.Base(tok))
		if flags, ok := execStringVerbs[name]; ok {
			if v, resolved, found := scanExecString(tokens[i+1:], flags,
				execStringOtherValueFlags[name], execStringCommandOperand[name]); found {
				return name, v, resolved, true
			}
			// This verb carries no command string: `su - bob` is a login shell, not
			// an execute-a-string. If it is also a wrapper (runuser, flock), the walk
			// continues through it; otherwise the segment is an ordinary command.
			if !wrappers[name] {
				return "", "", false, false
			}
		}
		if wrappers[name] {
			i = skipWrapperArgs(tokens, i+1, name)
			continue
		}
		break
	}
	return "", "", false, false
}

// scanExecString looks through ONE verb's tokens for its payload flag.
//
// It does not stop at the FIRST non-flag token, because these grammars put the
// flag after an operand (`flock FILE -c CMD`, `su USER -c CMD`) and getopt
// permutes. It stops in two places instead:
//
//   - at `--`, after which the tokens belong to the command being launched —
//     `runuser -u bob -- git push --force` is a git, not a payload, and the
//     wrapper walk already owns it;
//   - past commandOperandAfter operands, when the verb declares one, because the
//     launched command's own flags are none of this scan's business.
func scanExecString(tokens, payloadFlags, valueFlags []string, commandOperandAfter int) (value string, resolved, found bool) {
	operands := 0
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok == "--" {
			return "", false, false
		}
		if !strings.HasPrefix(tok, "-") || tok == "-" {
			// An operand: the user, the lock file, the typescript — or, past this
			// verb's own operands, the command it launches, whose flags are none of
			// this scan's business.
			operands++
			if commandOperandAfter > 0 && operands > commandOperandAfter {
				return "", false, false
			}
			continue
		}

		// Glued long form: --command=<value>.
		if eq := strings.IndexByte(tok, '='); eq > 0 {
			if containsString(payloadFlags, tok[:eq]) {
				return tok[eq+1:], true, true
			}
			continue // a glued value for some other flag
		}

		if containsString(payloadFlags, tok) {
			if i+1 >= len(tokens) {
				// Present but unresolvable. Fail loud: the guard knows a command
				// string was meant and cannot read it.
				return "", false, true
			}
			return tokens[i+1], true, true
		}

		// A short cluster carrying the payload flag last: `su -lc <value>`, which
		// getopt reads as -l -c <value>.
		if v, ok := clusteredPayload(tok, payloadFlags, valueFlags); ok {
			if v != "" {
				return v, true, true // glued value: -c<value>
			}
			if i+1 >= len(tokens) {
				return "", false, true
			}
			return tokens[i+1], true, true
		}

		if containsString(valueFlags, tok) {
			i++ // its value is not the payload
		}
	}
	return "", false, false
}

// clusteredPayload reads a short option cluster for a single-letter payload flag.
// It returns any value glued after that letter, and whether the cluster carried
// it at all. A cluster is only read when every letter before the payload letter
// is a plausible boolean short option — anything else is a flag this table does
// not model, and guessing at it would invent a payload.
//
// A value-taking short flag (valueFlags) before the payload letter is the case
// that guessing gets WRONG: getopt hands it the rest of the token as its value,
// so `script -Tc out.txt -c <hazard>` is `-T` with value `c`, not `-T -c`, and
// the real `-c` is later on the line. Reading `-Tc` as a payload cluster there
// resolves a bogus value and — because the segment then looks payload-carrying —
// switches the Tier-2 fail-safe off, letting the real hazard through both tiers.
// So a value-flag letter aborts the cluster read; the scan falls through to the
// operand and finds the genuine later payload flag.
func clusteredPayload(tok string, payloadFlags, valueFlags []string) (value string, ok bool) {
	if len(tok) < 3 || strings.HasPrefix(tok, "--") {
		return "", false
	}
	for _, flag := range payloadFlags {
		if len(flag) != 2 || !strings.HasPrefix(flag, "-") {
			continue // long spellings never cluster
		}
		letter := flag[1]
		idx := strings.IndexByte(tok[1:], letter)
		if idx < 0 {
			continue
		}
		for _, c := range tok[1 : idx+1] {
			if !isShortOptionLetter(c) {
				return "", false
			}
			if containsString(valueFlags, "-"+string(c)) {
				return "", false // a value flag swallows the rest as its argument
			}
		}
		return tok[idx+2:], true
	}
	return "", false
}

// isShortOptionLetter reports whether a rune can be a short option in a cluster.
func isShortOptionLetter(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// execStringWarnSignal is the loud-warn verdict for a payload flag whose value
// the guard cannot locate — the fail-safe that keeps `su -c` at the end of a line
// from reading as clearance.
func execStringWarnSignal(verb string) payloadSignal {
	return payloadSignal{
		verdict: VerdictWarn,
		family:  verb + " -c",
		reason: "This `" + verb + "` invocation carries a command string the guard cannot locate, " +
			"so the command it would run cannot be checked.",
		successor: "Put the command string immediately after the flag that carries it, so the guard can read it.",
	}
}
