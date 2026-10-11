---
schema_version: 1
id: "iss-340"
slug: "abcd-history-show-writes-the-stored"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "internal/surface/cli/cli.go"
resolution: "history show/list now pass the transcript body through termsafe.SanitizeBlock and metadata fields through termsafe.Sanitize (test TestHistoryShowSanitisesTranscriptBody)"
impact: fix
---

abcd history show writes the stored transcript body to the terminal with no termsafe sanitisation (cli.go human render branch) — a transcript that ingested hostile content (fetched pages, target-repo files) replays ESC/CSI/C1/bidi sequences raw; the termsafe sweep missed the history surface

## Evidence

- `internal/surface/cli/cli.go:2763` -- `fmt.Fprint(w, string(body))` (human render branch; JSON branch is fine).
- `internal/core/history/history.go:139-162` -- capture runs only `scanner.Redact` + a home-path backstop; nothing masks control characters. `history.go:217-243` returns bytes verbatim.
- `internal/termsafe/termsafe.go:40-68` -- `SanitizeBlock` (C0/DEL/C1/bidi/zero-width, newline-preserving); `internal/surface/cli/identity.go:169` is the hardened precedent for the same shape.

## Refuter verdict -- CONFIRMED (substantive)

The transcript body is documented untrusted input (`brief/04-surfaces/11-history.md` redaction boundary; `history capture <file>|-` ingests arbitrary files/stdin). No control-char stripping on the capture path; no raw-replay contract. Fix: `termsafe.SanitizeBlock(string(body))` on the human branch. Sibling fields on the READ path (`SessionID`/`SourceKind` via `parseRecord`, re-validated by neither charset nor enum) also wrapped in `termsafe.Sanitize`. Pinning test: an escape sequence, a two-byte C1, a bidi override, a bare CR; assert line count preserved.
