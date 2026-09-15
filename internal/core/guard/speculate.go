package guard

import (
	"fmt"
	"strings"
)

// Tier 2 — the position-agnostic fail-safe (adr-42).
//
// Tier 1 is the registry match at command position: precise, and the only tier
// that blocks. It rests on the `wrappers` table naming every program that
// launches another one, and that table cannot be complete — any binary that
// execs its arguments is a wrapper, and a repository adds one with a line in a
// Makefile. Three defects of that shape have been filed against three different
// enumerations in this package (gh-297 the interpreter set, gh-299 git's global
// value flags, iss-272 the wrapper set).
//
// So the enumeration stops being load-bearing. When no entry matched a segment,
// the matcher re-runs from each later command-position token; if a hazard turns
// up in one of those suffixes, the verdict is a loud WARN. Never a block: an
// unknown command is not proof that it execs its arguments, and `rg git push
// --force docs/` is a search, not a force-push.
//
// The wrapper class becomes loud instead of silent WITHOUT enumerating anything.
// Adding a wrapper name is still worth doing — it upgrades a warn to a precise
// block (part B) — but it is no longer the thing standing between a hazard and a
// confident allow.

const (
	// speculativeEntryID is the reserved id for a Tier 2 verdict. It is
	// deliberately NOT syntheticEntryID: adr-42 decision 2 requires a reader to be
	// able to tell a speculative warn from a literal one, because the reason Tier 2
	// only warns is that it is guessing about the launcher.
	speculativeEntryID = "unrecognised-launcher"

	familySpeculative = "unrecognised launcher"

	// maxSpeculativeStarts bounds the work PER SEGMENT. Speculation is N starts ×
	// E entries, which is the per-entry splice-and-restart shape expandPayloads'
	// doc comment forbids by name: prototyped without a cap it ran 14.9s at 6,000
	// tokens, extrapolating to hours of CPU inside a PreToolUse hook at the 1 MiB
	// stdin cap. Past the cap the guard has stopped looking, so it says so (see
	// speculativeCapSignal) rather than falling quiet — stopping quietly would be
	// the silent allow this file exists to remove.
	//
	// Per segment, not per command line: a per-line budget spent early would fire
	// the unconditional cap warn on every remaining segment of a long line, which
	// is a warn-rate decision, and a warn rate that trains users to ignore warns is
	// a STOP.
	maxSpeculativeStarts = 64

	// maxSpeculativeWindow bounds the tokens ONE start looks at. Capping starts
	// alone does not bound the cost: every start pays commandOf, which walks the
	// whole leading wrapper chain, so 64 starts on a 1 MiB line of `env env env …`
	// measured 14.2s — still linear in line length, and still a hang inside a
	// PreToolUse hook. 64 starts x 512 tokens x the entry count is instant.
	//
	// The window is a coverage trade, and it is a small one: every registry pattern
	// reads its command, the operand at position 0, and flags — all of which sit
	// near the command in any hazard the Pattern language can express. A pattern
	// whose flag is 512 tokens downstream of its own command name is not a shape
	// the registry has. When a window truncates, the segment warns anyway (see
	// speculativeCapSignal), so the coverage limit is never a silent one.
	maxSpeculativeWindow = 512

	// maxSpeculativeExpansionBytes bounds the payload text one segment's
	// speculation will re-read, and it is the third bound because the first two
	// were not enough.
	//
	// maxSpeculativeWindow caps the NUMBER of tokens a start looks at; it says
	// nothing about their SIZE, and an execute-a-string payload is a single token
	// of unbounded length. Wrappers are deliberately not skipped as starts, so
	// `myrunner env env … (x64) sh -c '<1 MiB>'` gives 64 starts that all resolve
	// to the same `sh`, and each one re-tokenized the whole payload: 23.6s
	// measured at the stdin cap, against 31ms before the change.
	//
	// adr-42 decision 3 said this in advance — "the benchmark must exercise the
	// payload path, or it pins only the cheaper half" — and the first bound test
	// pinned the cheaper half.
	//
	// The budget is per CHECK, not per segment, which is the second thing measuring
	// corrected: a per-segment budget left 1,200 short segments each paying it in
	// full, and 8.4s at 620 KB. Exhausting it warns through the same fail-loud path
	// as the other two bounds, so a truncated search is never a quiet one.
	maxSpeculativeExpansionBytes = 64 << 10

	// maxSpeculativeStartsPerCheck bounds the starts across the WHOLE command line,
	// because maxSpeculativeStarts is per segment and the number of segments is not
	// bounded at all. 1,200 short segments, each entitled to its own 64 starts,
	// measured 1.8s at 620 KB with the byte budget already in place — the same
	// mistake as the token-count window, one level up: every per-item bound needs a
	// whole-input bound behind it.
	maxSpeculativeStartsPerCheck = 1024
)

// speculationBudget is the work one Check may spend on Tier 2. Both quantities
// are whole-line, and running either down makes the remaining segments warn
// rather than fall quiet.
type speculationBudget struct {
	starts int
	bytes  int
}

// reservedEntryIDs are ids the guard speaks in its own voice. A registry entry
// claiming one would be indexed out of Registry.Entries by a synthetic winner
// (yielding a blank message), and would let a repo dress an ordinary entry up as
// the guard's own verdict.
var reservedEntryIDs = []string{syntheticEntryID, speculativeEntryID, braceEntryID, heredocEntryID, gitConfigEntryID}

// speculate runs Tier 2 over every segment Tier 1 left unmatched, returning at
// most one signal per segment (the first hit wins; there is nothing to gain from
// enumerating every way a line is suspicious).
//
// matched must be parallel to segs and true where any entry fired. The gate is
// "no entry matched THIS SEGMENT" — never "no entry matched the command line",
// which would hand an author a one-token off switch: `git clean -fd ; nice git
// push --force` matches git-clean, and a per-line gate would then leave the
// wrapped force-push silent.
func (r Registry) speculate(segs []segment, matched []bool, ids []string) []payloadSignal {
	var out []payloadSignal
	// One budget for the whole command line: a segment cannot buy itself more work
	// by being one of many.
	budget := speculationBudget{
		starts: maxSpeculativeStartsPerCheck,
		bytes:  maxSpeculativeExpansionBytes,
	}
	for i, s := range segs {
		if matched[i] {
			continue
		}
		// A segment whose payload the guard READ is not an unrecognised launcher,
		// even when nothing in it matched. `busybox sh -c "<hazard>"` classifies as
		// the shell family, its payload becomes segments of its own, and those carry
		// the verdict — speculating on the parent as well would report the same
		// hazard twice, once as a precise entry and once as a guess about it.
		//
		// A payload the guard reached only by GUESSING at a globbed command
		// name is the exception: there the reading is "this pattern can expand
		// to sh", not "this is sh", so it is taken in addition to the warn and
		// not instead of it. Dropping the warn there turned a hazard behind an
		// unknown launcher into a silent allow, which is exactly what adr-42
		// decision 2 says is never dropped.
		if _, _, _, _, carriesPayload := classifySegment(s); carriesPayload && !shellNameGuessed(s) {
			continue
		}
		if sig, ok := r.speculateSegment(segs[:i], s, ids, &budget); ok {
			out = append(out, sig)
		}
	}
	return out
}

// speculateSegment re-matches one segment from each of its later command
// positions, short-circuiting on the first hit.
func (r Registry) speculateSegment(before []segment, s segment, ids []string, budget *speculationBudget) (payloadSignal, bool) {
	starts, capped := speculativeStarts(s.tokens)
	if len(starts) > budget.starts {
		starts, capped = starts[:budget.starts], true
	}
	budget.starts -= len(starts)
	truncated := false
	// zsh's noglob switches expansion off for the whole command line behind it,
	// and a window that starts after the wrapper no longer holds it — so the
	// glob record is withheld from every window of such a segment, or Tier 2
	// would re-arm the compare Tier 1 correctly stood down.
	_, noglob := commandIndex(s)
	for _, start := range starts {
		tokens := s.tokens[start:]
		if len(tokens) > maxSpeculativeWindow {
			tokens = tokens[:maxSpeculativeWindow]
			truncated = true
		}
		// The glob record travels with the window: a globbed flag behind an
		// unrecognised launcher is still a pattern bash expands.
		cand := segment{tokens: tokens, chain: s.chain}
		if !noglob {
			cand.globbed = s.globSlice(start, start+len(tokens))
		}

		// Expand the suffix's own payloads, so `busybox sh -c "<hazard>"` is
		// reached: stepping busybox leaves `sh -c …` in command position, and the
		// interpreter path takes it from there. This also imports expandPayloads'
		// own signals, which is why they are demoted below rather than honoured.
		//
		// Charged against the byte budget, because this is the expensive half: the
		// expansion re-tokenizes every payload it finds, and the same payload is
		// reachable from many starts.
		expanded, sigs := []segment{cand}, []payloadSignal(nil)
		if size := segmentBytes(cand); size <= budget.bytes {
			budget.bytes -= size
			expanded, sigs = expandPayloads([]segment{cand})
		} else {
			truncated = true
		}

		for _, id := range ids {
			e := r.Entries[id]
			for j, es := range expanded {
				if !matchSegment(e.Pattern, es) {
					continue
				}
				if e.Pattern.AfterCD != nil && *e.Pattern.AfterCD &&
					!precededByCD(before, es.chain) && !precededByCD(expanded[:j], es.chain) {
					continue
				}
				return speculativeSignal(e), true
			}
		}
		if len(sigs) > 0 {
			return demoteToSpeculativeWarn(sigs[0]), true
		}
	}
	// Nothing found, but the search was bounded: say so. A fail-safe that stops
	// looking quietly is the silent allow this file exists to remove.
	if capped || truncated {
		return speculativeCapSignal(), true
	}
	return payloadSignal{}, false
}

// speculativeStarts returns the token indexes worth re-matching from, in one O(N)
// pass, and reports whether the list was truncated by the cap.
//
// Two token shapes are skipped, and each is skipped LOSSLESSLY — neither can
// produce a verdict that a later start does not also produce:
//
//   - a flag (`-n`, `--force`): commandOf breaks on the first token that is not
//     an assignment, a reserved word, or a wrapper, so a suffix starting at a flag
//     resolves to the flag itself, which no entry can name (Validate refuses a
//     Command containing a space or a slash, and every bundled Command is a plain
//     program name), and classifySegment finds no payload family there either;
//   - an assignment (`FOO=bar`) and a reserved word (`then`, `do`): commandOf
//     steps them and lands on a later token, which is itself a start.
//
// A known wrapper is NOT skipped, though commandOf would step it. The wrapper is
// exactly where a payload lives: `myrunner env -S '<hazard>'` needs `env` in
// command position for expandPayloads to read the -S value at all, and starting
// at the token after it hands the expansion a bare quoted string with no family
// around it. Skipping wrappers looks lossless for MATCHING and is not lossless
// for EXPANSION — and expansion is half of why a start earns its place.
//
// Index 0 is never a start — that is Tier 1's position, and it did not match.
func speculativeStarts(tokens []string) ([]int, bool) {
	var starts []int
	for i := 1; i < len(tokens); i++ {
		if !eligibleStart(tokens[i]) {
			continue
		}
		starts = append(starts, i)
		if len(starts) == maxSpeculativeStarts {
			// Report the cap only when an ELIGIBLE start was actually dropped. "Some
			// tokens remain" is the wrong test: the tokens this loop skips are skipped
			// losslessly, so trailing flags would otherwise fire the unconditional
			// "too long to inspect" warn on a line where nothing was missed — and warn
			// rate is the declared STOP, so a spurious cap warn is the direction that
			// costs.
			for j := i + 1; j < len(tokens); j++ {
				if eligibleStart(tokens[j]) {
					return starts, true
				}
			}
			return starts, false
		}
	}
	return starts, false
}

// eligibleStart reports whether a token is worth re-matching from.
func eligibleStart(tok string) bool {
	if tok == "" || tok == "-" {
		return false
	}
	return !strings.HasPrefix(tok, "-") && !isAssignment(tok) && !reserved[tok]
}

// segmentBytes is the segment's total token size, the quantity the expansion
// budget is spent in. Token COUNT is the wrong unit here: one token can carry a
// megabyte of payload.
func segmentBytes(s segment) int {
	n := 0
	for _, tok := range s.tokens {
		n += len(tok)
	}
	return n
}

// speculativeSignal is the Tier 2 warn for a hazard found behind an unrecognised
// launcher. It carries the matched entry's why and successor, so the warn teaches
// the same safe form the block would have.
func speculativeSignal(e Entry) payloadSignal {
	return payloadSignal{
		id:      speculativeEntryID,
		verdict: VerdictWarn,
		family:  familySpeculative,
		reason: fmt.Sprintf(
			"This command line starts with a program the guard does not recognise, and a hazard the registry names (%s) appears after it in command position. "+
				"The guard cannot tell whether that program runs the rest of the line, so this is a warning rather than a refusal. %s",
			e.ID, e.Why),
		successor: e.Successor,
	}
}

// demoteToSpeculativeWarn folds a signal raised inside a speculative suffix down
// to a warn. expandPayloads raises blockers of its own (an env -S value it cannot
// prove plain, a nesting depth it cannot follow), and a speculative suffix reaches
// them — `env -S 'git push --force $X'` blocks on its own, and `myrunner env -S
// 'git push --force $X'` would inherit that block through a launcher nobody has
// established runs anything.
//
// Honouring the signal at its own tier would block on two layers of uncertainty
// at once. Dropping it would invent a second silent suppression path inside the
// fail-safe. Demotion keeps both invariants; the id changes so the report says
// which tier spoke.
func demoteToSpeculativeWarn(sig payloadSignal) payloadSignal {
	return payloadSignal{
		id:      speculativeEntryID,
		verdict: VerdictWarn,
		family:  familySpeculative,
		reason: "This command line starts with a program the guard does not recognise, and what follows it could not be read: " +
			strings.TrimSuffix(sig.reason, " ") +
			" The guard cannot tell whether that program runs the rest of the line, so this is a warning rather than a refusal.",
		successor: sig.successor,
	}
}

// speculativeCapSignal is the fail-loud verdict for a segment with more command
// positions than the guard will inspect. It is not a hazard report — it is the
// guard saying it stopped looking, which is the one thing a fail-safe must never
// do quietly.
func speculativeCapSignal() payloadSignal {
	return payloadSignal{
		id:      speculativeEntryID,
		verdict: VerdictWarn,
		family:  familySpeculative,
		reason: fmt.Sprintf(
			"This command line is too long for the guard to inspect fully behind a program it does not recognise: the search is bounded at %d command positions "+
				"and %d tokens from each, because checking every one of them is how an earlier attempt turned a hazard check into a denial of service.",
			maxSpeculativeStarts, maxSpeculativeWindow),
		successor: "Split it into separate commands, or run the part that matters on its own, so the guard can check what actually runs.",
	}
}
