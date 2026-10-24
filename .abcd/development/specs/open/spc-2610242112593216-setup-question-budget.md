---
id: spc-2610242112593216
slug: setup-question-budget
intent: itd-2610240823055889
origin: researcher-authored
production_mode: hand-written
---
# setup-question-budget

## Summary

What spc-2610241007434505 did not deliver for itd-2610240823055889: criterion 5, the question budget. On a fresh account with the assistant's settings file present, setup asks the download, shell settings, commit checks, status line and identity questions plus one per missing tool. That is more than the four-plus-tools the criterion allows, because criterion 3 keeps the status line its own approved question and the commit-checks question was added after review (iss-2610241134315056). This spec holds until the product thinker rules on dcn-2610241228598485: count these in the budget, fold the status line into the download question, or move them onto the plan screen. It then brings the run within the ruled budget, with a test counting the questions a fresh account is asked.

## Footprint

- packages: internal/core/ahoy, internal/surface/cli
- tests: internal/core/ahoy/question_budget_test.go

## Steps

_No steps listed: the spec is built as one step. To split it, list the steps in order as `1. <title>`, each with `- packages:` and `- tests:` indented beneath it; `- landed: <pull request or commit>` marks a step that has landed._
