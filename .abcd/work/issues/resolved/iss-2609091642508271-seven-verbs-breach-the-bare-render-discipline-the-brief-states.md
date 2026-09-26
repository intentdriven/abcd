---
schema_version: 1
id: "iss-2609091642508271"
slug: "seven-verbs-breach-the-bare-render-discipline-the-brief-states"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "release-gate"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
resolution: "Every visible top-level verb either renders state on a bare call, checked by a test that runs it bare in a scratch checkout, or is recorded with its reason in bareRenderExceptions (barerender.go), which the brief's enumeration is held to; a verb added later fails until it is one or the other."
impact: internal
resolved_by:
  commit: "2bfb1fade25d9a0c938b401201869ff0d3739d66"
---

The brief states as a requirement that a bare verb renders state and writes nothing, and seven verbs do not. Six of them, disembark and docs and embark and guard and history and ideate, render only the command framework's help on a bare call, and launch exits one. The history verb breaks the rule from both ends: its list and show sub-verbs take the shape the naming constraints forbid while the bare verb renders no state at all. The record has been corrected to say what is true rather than what was intended, so the discipline is no longer stated as satisfied, and that correction is what makes this a code observation rather than a documentation defect. It matters because the bare render is the discipline that makes a verb safe to type when you do not know what it will do, and a reader who has learnt the rule from the twelve verbs that keep it will type the other seven expecting the same. Fix direction: give each bare verb a state render, or record per verb why it has none, so the exception is a decision rather than a gap. Detector: every verb the binary registers either renders state on a bare call or is listed as an exception with its reason, and a verb added later fails until it is one or the other.

## Grounds

- pursued: no verb departs from the bare-render discipline without a recorded reason; a new top-level verb that prints only usage bare and passes the test would show it wrong
