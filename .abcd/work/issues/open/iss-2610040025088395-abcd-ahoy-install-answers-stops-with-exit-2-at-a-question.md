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
remedy: "Have setup settle every answer before its first write: core collects the questions a run will ask (a plan from Detect, the approvals and the flags) and the answers file is checked against it before Install runs, so a missing answer refuses before anything is written; grounds: the spec's B3 refusal case and the existing write-gate pattern in install(), which refuses before the adoption question when a refusal is known up front."
---

abcd ahoy install --answers stops with exit 2 at a question the answers file does not answer, and writes no answers record, but it does not write nothing as spc-2610030911534855 says: setup asks its value questions and offers between its steps, so a stop at a later question (a value question, the status line, the routing offers, the house-style or artefact kind question) leaves the steps before it done. Only the adoption question and the approvals, which are all asked before the first write, stop with the repository untouched.
