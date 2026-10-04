---
schema_version: 1
id: "iss-2610040025088395"
slug: "abcd-ahoy-install-answers-stops-with-exit-2-at-a-question"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25 (lane e8term3, plain-Terminal step 3)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/setup_interview.go"
remedy: "Waits on a question plan in core ahoy (the setup questions' owner, abcd-c9): core collects, before the install's first write, the questions whose asking the run itself decides (the status line's elements, the routing and status-line offers, the lineage question, the house-style question, the artefact kind) from Detect, the approvals and the flags, and the answers file is checked against that plan before Install runs, as ahoy.WalkConfigValueQuestions already does for the config values; grounds: the spec's B3 refusal case, and the walk this branch added, which follows the install's own value collection (collectMissingValues) and a test holds to the questions the install asks."
---
abcd ahoy install --answers stops with exit 2 at a question the answers file does not answer, and writes no answers record, but it does not always write nothing, as spc-2610030911534855's B3 says it must.

Narrowed on 2026-10-04 (lane e8term3fix): setup now checks the answers file before its first write. An entry whose question has a fixed set of answers (a config value, the house-style question, a status-line element, every approval) must give one of them, and every config value the run would ask (the visibility, the docs target, the oracle backend, deep scanning) must be answered by its flag or by the file, walked by ahoy.WalkConfigValueQuestions in the order and as far as the install asks them. Either refusal comes before Install runs and says that nothing was written. The adoption and the approvals were already asked before the first write.

What remains: the questions whose asking the run itself decides are still checked only when they are put, between the install's steps: the status line's elements, the routing and status-line offers, the lineage question, the house-style question and the artefact kind (the drain rule, the git identity and a conventions file's retirement are never answered from a file). A stop at one of them leaves the steps before it done; its message now says so ("what the install changed before this question stays"), and the next run asks only what is still open. Closing it needs the question plan in core ahoy that the remedy names.
