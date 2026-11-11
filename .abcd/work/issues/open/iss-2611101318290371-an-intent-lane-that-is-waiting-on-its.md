---
schema_version: 1
id: "iss-2611101318290371"
slug: "an-intent-lane-that-is-waiting-on-its"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/handback.go"
remedy: "Let --discard take an intent lane at its implement stage that has no commits beyond its base, removing its worktree and branch and leaving its step unlanded; a test discards a fresh lane at implement and finds it closed with nothing else changed."
---

An intent lane that is waiting on its implementer cannot be released when no work has been done: implement step --discard takes only a held lane or one handed back as stuck or closed, so a person who finds the lane should not run has to write a hand-back receipt by hand. On 7 Nov at 15:41 and 16:00 a lane built on a spec that turned out not to be READY could be released only by handing it back as a design finding.
