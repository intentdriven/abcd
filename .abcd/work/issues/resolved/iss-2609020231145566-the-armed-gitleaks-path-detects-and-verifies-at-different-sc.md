---
schema_version: 1
id: "iss-2609020231145566"
slug: "the-armed-gitleaks-path-detects-and-verifies-at-different-sc"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/history.go"
resolution: "toFindings now locates each non-blank LINE of a reported value independently across the whole text and emits a finding at every occurrence, so a line that also occurs outside the value — a quoted end-of-key marker, a repeated header, a repeated body line — is sealed at that site too and history.unsealedAugmented's presence-anywhere check has nothing left to refuse on: detection and verification share one scope. A value counts as located only when every non-blank line of it is found, so a text holding part of the reported value still falls through to Match and then to ErrFindingNotLocated (TestAugmentUnlocatableReportFailsClosed unchanged). The stray line is SEALED rather than left as public boilerplate, and a repeated body line makes the capture store with both occurrences masked rather than refuse: refusing is what left the unredacted staged copy waiting for a drain that could never succeed, and the stored record holds no occurrence of the reported value either way. The cost is over-redaction of a boilerplate marker line, counted in the record's redacted_secrets bucket. Proved by TestCaptureSealsEveryRecurrenceOfAnAugmentedFragment (history), whose four variants — header repeated, footer repeated, body repeated, clean — all store with every occurrence masked; the three recurrence variants were watched refusing with RedactionResidualError before the change."
impact: fix
---

The armed gitleaks path detects and verifies at different scopes, so a benign recurrence of one fragment refuses every capture for good. toFindings locates the WHOLE reported value in the transcript and emits one finding per line that value crosses, but history.unsealedAugmented verifies each FRAGMENT by presence anywhere in the redacted text: a transcript that also quotes a standalone PEM end-of-key marker line, outside any key, keeps that line unsealed (detection never emitted a finding for it) while verification finds it, so Capture returns RedactionResidualError on every drain, the session is never stored, and its unredacted .raw copy stays staged indefinitely. Evidence: internal/adapter/gitleaks/gitleaks.go toFindings (the whole-needle strings.Index loop and its per-fragment split) and internal/core/history/history.go unsealedAugmented (the strings.Contains-anywhere check). The fix must give detection and verification one scope, so that any byte sequence verification looks for is a sequence detection located and sealed, while keeping the fail-closed contract TestAugmentUnlocatableReportFailsClosed pins.

## Grounds

- pursued: widening detection to the line scope rather than narrowing verification to the spans detection produced, because the narrow shape would store a repeated line of a real key verbatim while still reporting the capture clean, and because a span-position compare drifts under the identity placeholders Redact rewrites after the seal — which is why unsealedAugmented checks bytes and not positions. Shown wrong if over-redacting a boilerplate marker line ever costs more than it protects.
