---
schema_version: 1
id: "iss-2609292359485570"
slug: "the-shared-run-s-bounds-use-step-for-a-third-thing-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/bounds.go"
remedy: "Waits on a product ruling (BU1 covered only the build loop): rename the check vocabulary to 'point' (Point, the verdict's JSON 'point', 'may take the release point'), or keep 'step' and state the third sense in the glossary's step entry under a Senses table as core/loop.md does; the rename is breaking (a JSON field and help wording) so it rides the next breaking release. Grounds: the glossary convention that a word used in more than one sense carries a Senses table (brief/glossary/core/README.md)."
resolution: "Ruled CM1 (2026-09-29): implement check calls what a session takes a stage; the verdict's and the logged refusal's field is stage, its text, help, command page and brief say stage, and the upgrade guide names both renamed fields."
impact: breaking
resolved_by:
  commit: "615604f36e234cd6b72ce9e023b05ca4527d772d"
---

The shared run's bounds use 'step' for a third thing: abcd implement check <lane|release|review|audit|land> asks whether a session 'may take a step', its verdict carries the point under a JSON 'step' field (internal/core/implement/bounds.go, Step/StepLane/StepRelease/StepReview/StepAudit/StepLand), and commands/implement.md and brief 27-implement.md speak of 'the release step' and 'a step that is not a claim'. BU1 (2026-09-29) settled that in the build loop 'step' names a spec's piece and a lane's are stages; the check vocabulary, a point in a shared run where the second session's bounds are checked, was not in that ruling and still says step.

## Grounds

- pursued: step names a spec's piece and nothing else anywhere in the run's vocabulary; shown wrong if implement check's JSON, log line, text or help still says step for what a session takes
