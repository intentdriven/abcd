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
---

The brief states as a requirement that a bare verb renders state and writes nothing, and seven verbs do not. Six of them, disembark and docs and embark and guard and history and ideate, render only the command framework's help on a bare call, and launch exits one. The history verb breaks the rule from both ends: its list and show sub-verbs take the shape the naming constraints forbid while the bare verb renders no state at all. The record has been corrected to say what is true rather than what was intended, so the discipline is no longer stated as satisfied, and that correction is what makes this a code observation rather than a documentation defect. It matters because the bare render is the discipline that makes a verb safe to type when you do not know what it will do, and a reader who has learnt the rule from the twelve verbs that keep it will type the other seven expecting the same. Fix direction: give each bare verb a state render, or record per verb why it has none, so the exception is a decision rather than a gap. Detector: every verb the binary registers either renders state on a bare call or is listed as an exception with its reason, and a verb added later fails until it is one or the other.
