---
schema_version: 1
id: "iss-184"
slug: "guard-heredoc-arithmetic-shift-bypass"
severity: "critical"
category: "bug"
source: "agent-finding"
found_during: "bug-hunt loop round 1"
found_at: "internal/core/guard/tokenize.go:152"
resolution: "Fixed at root cause, in two parts after a pre-PR security review caught the first pass narrowing rather than closing the hole. (1) Classification: a heredoc delimiter word immediately followed by a bare '(' or ')' with no separator is never a real heredoc -- the body and terminator line have to come first, so nothing legitimate places a paren directly against the delimiter word. `$((expr<<ident))` produces exactly that shape, so `<<` in that position is now correctly read as the arithmetic operator at tokenize time, same as the pre-existing literal-digit case, and the dangerous command on a later line reaches command position and blocks normally -- no error, no swallow, no reliance on whether a later line happens to coincidentally match the misread delimiter (the security review's exploit: appending a bare `shift` line let the earlier, error-only fix still swallow the guarded command silently). (2) Defense in depth: skipHeredocBodies still signals when a GENUINE pending heredoc never finds its terminator line, and tokenize turns that into ErrUnparsableCommand (guard fails open LOUDLY per the documented itd-103 contract) rather than silently consuming the rest of the input -- covers truly malformed/unterminated heredocs, a distinct residual gap the same swallow covered. Detectors: internal/core/guard/tokenize_test.go TestArithmeticShiftByIdentifierIsNotAHeredoc, TestArithmeticShiftCoincidentalDelimiterStillBlocks (the adversarial payload), and TestTokenizeRejectsUnterminatedHeredoc -- all watched failing before their respective fix and passing after."
impact: fix
---

guard tokenizer heredoc misparse: an unquoted arithmetic left-shift with an identifier operand (e.g. `$((1<<shift))`) is misparsed as a here-document start by isDelimStart (tokenize.go:206, accepts any word starting with a letter/underscore) plus readHeredocDelim (tokenize.go:249, terminates the delimiter word on ')'). skipHeredocBodies (tokenize.go:264) then scans for a line equal to the bogus delimiter, finds none, and silently swallows every subsequent line of the command before it reaches command position. A dangerous command on a later line (git push --force, rm -rf, gh repo delete, etc.) is never seen by Registry.Check and the guard returns VerdictAllow with no error — a silent fail-open, not the loud could-not-answer path the shim reserves for unparsable input. Confirmed live via both CLI front doors (internal/surface/cli/guard.go: guard check and guard hook). The literal-digit form (1<<20) is unaffected; only identifier/variable operands trip it. Reproducing test: internal/core/guard/tokenize_test.go, TestArithmeticShiftByIdentifierIsNotAHeredoc.

## Correction, 2026-09-29

The resolution was true when it was written, and its part (2) is not true at
this tip. `16d50feb` ("fix: give the two bash-executable tokenizer states a
verdict") changed what a genuine heredoc with no terminator line does: it is no
longer `ErrUnparsableCommand`, because an error reaches the pre-tool-use hook as
fail-open, but a segment flag that `Check` folds into a fail-closed block. The
same commit replaced `TestTokenizeRejectsUnterminatedHeredoc`, which the
resolution names, with `TestTokenizeFlagsUnterminatedHeredoc` in
`internal/core/guard/tokenize_test.go`, which pins the block. Part (1), the
arithmetic-shift classification, stands as the resolution describes it, and
its two tests are still in the tree.
