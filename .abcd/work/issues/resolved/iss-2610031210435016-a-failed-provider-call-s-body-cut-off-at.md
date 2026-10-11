---
schema_version: 1
id: "iss-2610031210435016"
slug: "a-failed-provider-call-s-body-cut-off-at"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25 (lane streamFix, security re-check follow-up)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/openaiapi/client.go"
remedy: "Read the failed call's body to maxErrorBodyBytes+1 and quote it only when it was read whole: a body cut off by the size bound, by the error-body wait or by any read error is reported on the status alone with a note that it was cut off, because a cut can end inside a key form the scrub cannot see. Grounds: the scrub replaces whole key forms only (replaceForms), so only a body read whole can be scrubbed completely; the reproducing test is the proof."
resolution: "A failed call's body is quoted only when it arrived whole; a body cut off by the error-body wait, the 16 KiB size bound or a read error is reported on the status alone and named as cut off, so no cut can carry a key prefix the scrub cannot see."
impact: fix
resolved_by:
  commit: "06831e9a7"
---

A failed provider call's body cut off at its 16 KiB size bound can end inside an echoed key, and the error then quotes a prefix of the key: internal/adapter/openaiapi/client.go reads a non-200 body through io.LimitReader(maxErrorBodyBytes), and providerSaid trims whitespace before bounding the echo to 200 bytes, so a 401 body of whitespace padding followed by the key yields the key's first bytes in the refusal, which the scrub (whole key forms only) does not match. Reproduced at the streaming lane's tip by TestAnErrorBodyCutOffIsNotQuoted/cut_by_the_size; the same read on main has the same cut.

## Grounds

- pursued: no refusal of a failed call carries any prefix of the key however its body is cut, as TestAnErrorBodyCutOffIsNotQuoted proves for the wait and the size cut; a provider error body that arrives whole but is still unscrubbable, or a cut path other than these three, would show it wrong.
