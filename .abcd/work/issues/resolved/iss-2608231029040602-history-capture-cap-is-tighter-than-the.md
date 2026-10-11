---
schema_version: 1
id: "iss-2608231029040602"
slug: "history-capture-cap-is-tighter-than-the"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "transcript backlog recovery 2026-08-23"
found_at: "internal/surface/cli/cli.go"
resolution: "history capture now reads through maxTranscriptBytes rather than the 8 MiB JSON-operand cap, via readSourceCapped, so the recovery verb accepts what the hooks accept. The caps themselves are unchanged and still refuse an over-cap file whole. Found in the field: it refused an ordinary 11.8MB session during the backlog recovery, which the fix then unblocked."
impact: fix
resolved_by:
  commit: "2e5a87d"
---

`abcd history capture` read its operand through `maxOperandJSONBytes` (8 MiB)
while the SessionEnd path read through `maxTranscriptBytes` (64 MiB). Every
transcript between the two was therefore capturable automatically and
unrecoverable by hand — the recovery verb bounded eight times tighter than the
thing it exists to recover from. Stdin carried the same cap, so there was no
workaround.

Found while recovering this repo's backlog on 2026-08-23: ten of eleven missed
sessions ingested, and `9c89b576` (11.80 MB) refused with "exceeds the
8388608-byte cap". That is an ordinary session, not a pathology — this repo's
transcripts reach 11.8 MB routinely — so the bound was refusing real work.

The two constants are not competing policies. `maxOperandJSONBytes` is
documented as the house cap for an untrusted JSON operand, sized to match the
registry and graveyard payloads, and it reached the transcript path only because
`readSource` is shared. `maxTranscriptBytes` is the deliberate transcript bound.
This was a wrong-constant bug, so the fix names the cap at the call site rather
than widening a shared default.

The caps themselves are load-bearing and stay. Both transports read whole into
memory before the scanner walks them, so an unbounded read is a memory hazard on
a hook path that must never stall a session; and both refuse an over-cap file
WHOLE rather than truncating, because a severed prefix would be stored under a
sha256 idempotency key computed over the prefix (spc-4's refuse-whole
invariant).