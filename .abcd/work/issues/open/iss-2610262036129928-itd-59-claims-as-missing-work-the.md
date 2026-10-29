---
schema_version: 1
id: "iss-2610262036129928"
slug: "itd-59-claims-as-missing-work-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 631 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md:372"
remedy: "Re-scope itd-59 to the unattended passes not yet routed through capture: the runner already stores each delegated agent's transcript through internal/core/history (internal/core/runner/transcript.go); state what the draft adds beyond that, or retire it."
related_intents: [itd-59]
finding_ref: cfk-9f67ebabcc3035d6
---

itd-59 claims, as missing work, the capture of autonomous-run transcripts into the corpus; the history chapter says abcd build's delegated agent transcripts already reach the store through capture.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/11-history.md:372` — "a delegated agent's transcript that `abcd build` and the written"
- End B: `.abcd/development/intents/drafts/itd-59-autonomous-worker-transcript-capture.md:36` — "every autonomous-run pass's instructions + reasoning lands in the native transcript corpus"

11-history lists, among the write paths into the corpus, a delegated agent's transcript that abcd build stores through the same capture. itd-59's stated gap is that every autonomous-run pass's instructions and reasoning should land in the native transcript corpus, on the premise that the unattended worker leaves only an ephemeral log. The same ground is claimed twice, once as shipped and once as unbuilt, and the draft does not say what it adds to the build capture that already exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 631.
