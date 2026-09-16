---
schema_version: 1
id: "iss-2609100531051385"
slug: "a-verb-s-requirements-are-revealed-one-refusal-at-a-time-so"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
---

A verb's requirements are revealed one refusal at a time, so satisfying it takes as many invocations as it has required inputs. Resolving one issue took three calls. The first refusal reports only that the verb accepts two positional arguments and received one, and names none of the flags it also requires. Supplying the second positional then produces a second refusal, for the missing grounds. Reproduced here against the same binary: the argument-count refusal mentions no flag at all, and the grounds refusal arrives alone even when the other required flag is also absent, so a caller learns the requirements in series rather than at once. Each refusal in isolation is well written, and the grounds refusal in particular explains why it wants what it wants and confirms that nothing was written. The defect is the sequence. A caller who knows nothing pays one round trip per requirement, and an autonomous caller pays it every time because it has no memory of the last session's discoveries. A single refusal listing every unmet requirement would cost one round trip regardless of how much the caller already knew. This is the same family as the finding that required flags are learned from the refusal rather than the help, and the two want different fixes: that one wants a worked example in the help, this one wants the refusals aggregated.
