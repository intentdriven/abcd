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
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: whether the lane ceiling counts reviewers separately (pacing input to itd-2609201925079472)). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
resolution: "Answered by the product thinker's rulings on the sub-agent ceiling, which replace the pilot's flat two: .abcd/work/DECISIONS.md, 2026-09-23 (five sub-agents alive at once, in any mix of roles; then six from session 3) and 2026-09-24 (four, in any mix of roles, continuous; recorded in 0eb64ef24). 'In any mix of roles' is the ruling the record asked for: reviewers count against the same ceiling as implementers, and the ceiling is raised rather than split by role. The run's lane rules carry the current figure (five agents alive, ruled 2026-09-29)."
impact: internal
resolved_by:
  commit: "0eb64ef24"
---

Under the two-sub-agent ceiling, reviews run one at a time in the gaps and each fix round is a fresh agent, so the pattern implement, review, fix, security review, fix consumes the whole ceiling for one lane: in the pilot's first two-hour session lane D never opened (40 minutes of pure ceiling wait) and lane B's review waited 27 minutes; in the second session D waited 30 minutes before a stop. Measured cost across the run: 178 agent-minutes in 120 wall-minutes (session 1), 149 in 122 (session 2). Wanted, as a pacing input for itd-2609201925079472: the ceiling counts reviewers separately from implementers, or the window length is set from the lane shape (implement plus two reviews plus two fix rounds is about 90 agent-minutes per lane), rather than a flat two.

## Grounds

- pursued: a ceiling counted across all roles and raised above two stops a single lane's implement-review-fix cycle from consuming it; shown wrong if a later run again records a lane that never opens for ceiling wait alone
