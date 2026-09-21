---
schema_version: 1
id: "iss-2609211105014235"
slug: "the-two-agent-ceiling-turns-reviews-into-a-queue-and-a-fourth-lane-never-opens-in-a-window"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "pilot run 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "the pilot run's pace rule: two sub-agents alive at once"
---

Under the two-sub-agent ceiling, reviews run one at a time in the gaps and each fix round is a fresh agent, so the pattern implement, review, fix, security review, fix consumes the whole ceiling for one lane: in the pilot's first two-hour session lane D never opened (40 minutes of pure ceiling wait) and lane B's review waited 27 minutes; in the second session D waited 30 minutes before a stop. Measured cost across the run: 178 agent-minutes in 120 wall-minutes (session 1), 149 in 122 (session 2). Wanted, as a pacing input for itd-2609201925079472: the ceiling counts reviewers separately from implementers, or the window length is set from the lane shape (implement plus two reviews plus two fix rounds is about 90 agent-minutes per lane), rather than a flat two.
