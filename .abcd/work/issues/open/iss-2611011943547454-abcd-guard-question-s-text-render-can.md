---
schema_version: 1
id: "iss-2611011943547454"
slug: "abcd-guard-question-s-text-render-can"
severity: "nitpick"
category: "inconsistency"
source: "impl-review"
found_during: "delta review of build run-2610312053537139 lane 2, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "the text render of abcd guard question"
remedy: "Have the outcome line in the verb's text render count the structure findings it reports, saying 'checked; its only findings are structural' when every finding is a non-refusing structure finding, and test it with the chipped no-id --ask draft under an unread mode store."
---

abcd guard question's text render can contradict its own findings list. Take an --ask draft with a valid chip but no id: it carries one non-refusing structure finding, while the input composed for the hook is clean. If the mode store or the away record cannot be read as well, the text prints 'this question was checked and has no finding, but …' and, a few lines later, 'findings: 1, 0 of them refusing'. The JSON and the exit code (2) are correct, so this is the text render's wording alone. The line comes from step 2 of spc-2610301950037469 (PR #1333), in internal/surface/cli/guard_question_verb.go. It was found by that lane's delta review on 2026-11-01.
