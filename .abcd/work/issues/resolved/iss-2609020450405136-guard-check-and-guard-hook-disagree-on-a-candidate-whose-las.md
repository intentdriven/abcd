---
schema_version: 1
id: "iss-2609020450405136"
slug: "guard-check-and-guard-hook-disagree-on-a-candidate-whose-las"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
resolution: "A here-document still pending when the INPUT ends now takes the same fail-closed heredoc-unterminated block as one whose delimiter line never came: tokenize resolves the pending state at its final flushSegment as well as at a newline, so the verdict belongs to the command rather than to whether its last byte is a newline. guard hook (payload untrimmed) and guard check (stdin trimmed, and --command untrimmed) therefore return the same verdict for the same command. Proved by TestGuardCheckAndHookAgreeOnAHereDocumentLeftOpen (internal/surface/cli), which drives all three channels for 'cat <<EOF' with and without its trailing newline, and by the four ambiguous rows of TestSubshellHeredocIsNotArithmetic (internal/core/guard), which now assert the fail-closed verdict the comment claimed instead of only that no error was returned — three of them were silent allows. Both watched failing on a scratch archive first."
impact: fix
---

guard check and guard hook disagree on a candidate whose last line opens a here-document. tokenize resolves a pending here-document only when it crosses a newline, so a candidate that ends at the redirection (`cat <<EOF`, no trailing newline) drops the pending document silently and reads allow, while the same command with its newline intact takes the fail-closed heredoc-unterminated block. guard check trims the trailing newline off stdin (guardCandidate, internal/surface/cli/guard.go), so the two front doors return different verdicts for the same command: the hook blocks, check clears. Evidence symbol: tokenize's final flushSegment, which returns with pending non-empty. The fix must give a here-document still pending at the END OF INPUT the same verdict as one whose delimiter line never came, so both front doors answer identically whether or not the candidate ends in a newline.

## Grounds

- pursued: the divergence is an allow-direction one on the surface a script queries, and fixing it in the core rather than by untrimming stdin makes every channel agree at once — the trim is a reasonable thing for a front door to do, and no front door should be able to change a verdict by doing it
