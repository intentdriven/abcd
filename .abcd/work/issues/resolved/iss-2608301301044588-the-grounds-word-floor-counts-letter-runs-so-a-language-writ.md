---
schema_version: 1
id: "iss-2608301301044588"
slug: "the-grounds-word-floor-counts-letter-runs-so-a-language-writ"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-179-round-3-ruthless"
found_at: "internal/core/grounds/grounds.go"
resolution: "the substance floor's unit is a word where the script separates words and a LETTER where it does not: textUnits counts one unit per letter-run in a spaced script and one per letter of a named scriptio-continua script (Han, Hiragana, Katakana, Thai, Lao, Khmer, Myanmar, Tibetan, Javanese), so a Chinese or Japanese conjecture — and mark-light text in the other seven, which prose in those scripts reached only by the accident of its Mn marks breaking the runs — clears the floor the writer and the reader both inherit from ValidateText. The letter test is load-bearing: a Unicode script table carries its script's digits, punctuation and marks too, so twenty Thai digits or twenty Tibetan tsheg still count nothing, as do zero-width padding and ideographic commas. The refusal asks for letters only when every counted unit is one, so a text carrying a word alongside an ideograph is still told to add a word."
impact: fix
resolved_by:
  intent: "itd-179"
---

the grounds word floor counts letter runs so a language written without inter-word spaces is refused and its bullet is silently dropped by the reader

Found by the round-3 adversarial ruthless review of build/itd-179.
REGRESSION INTRODUCED BY fc557f10 -- the text passed before the floor existed.

`wordRe = regexp.MustCompile("[\\p{L}]+")` counts letter RUNS, and a language
written without inter-word spaces has exactly one. Measured by the reviewer:
Chinese 26 runes -> 1 run; Japanese 30 runes -> 1 run. Both are refused with
"carries 1 word(s), below the 3-word floor" despite being substantive
conjectures well clear of MinTextLen. Thai survives by accident, because its
combining marks are Mn rather than L and so break the runs.

Two things make this worse than an i18n nitpick:

1. **The refusal is satisfiable by punctuation.** Adding two ideographic commas
   yields 3 runs and the same text passes. For that script the floor degenerates
   into a comma count -- which is the "answerable with nothing" failure the
   commit set out to close, displaced by script rather than removed.

2. **The reader half compounds it into a gate that lies.** `intent.ParseGrounds`
   applies the same `ValidateText`, so such a bullet is silently dropped, and
   ready.go:306 then reports "no recorded grounds -- the conjecture behind
   pursuing itd-N is unrecorded" about a record that VISIBLY CARRIES ONE. That
   is the gate/reader divergence class this branch has spent three rounds
   closing, re-entering through the floor.

No committed record is affected: all 10 populated records are Latin script and
clear the floor (the corpus's shortest entry is 32 words). So this blocks
nothing today and is captured rather than chased.

Remedy (smallest, class-level): express the floor over a property that survives
scriptio continua -- count `\p{L}` RUNES, applying the run-count condition only
when the text contains no ideographic or kana runes. Failing that, state the
scope honestly in the doc comment AND in the refusal message, so an operator is
not told to add words to a language that has no word breaks.

Do NOT raise or lower 3: for Latin script the value is well judged.

CORROBORATED INDEPENDENTLY by the round-3 security review, which reached it by
a different route (a 36-case injection harness plus route enumeration) and
graded it a correctness/availability defect rather than a security one --
declining to launder it into a block, and putting the fix-or-record call to the
orchestrator.

Its added detail, which raises the severity of the consequence:

- The argument is MANDATORY since round 1. So a Chinese- or Japanese-writing
  operator cannot `promote` or `resolve` ANY issue, and cannot satisfy the
  `intent ready` gate. There is no flag and no waiver.
- The failure is ASYMMETRIC: `capture wontfix <id> "<CJK reason>"` still
  succeeds, because the reason-derived path skips ValidateText. So the same
  operator can decline work but cannot pursue it.
- The remedy the gate PRINTS on refusal will itself refuse the same text.

Verified not to be a wedge on this corpus: all 10 populated records are Latin
script and the measured figures are identical before and after the floor.

## Grounds

- pursued: we expect a floor expressed over the text's letters rather than over its inter-word spaces to admit every script without re-opening the padding cases; appending one ideograph to a single long word, or one kanji to twenty zero-width spaces, would still clear it if that were wrong
