---
schema_version: 1
id: "iss-2610241414105439"
slug: "a-dependabot-pull-request-1105-a-go-deps"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "drain session abcd-0d, 2026-10-24 autonomous test run"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/dependency-reauthor.yml"
remedy: "When the re-author workflow cannot run (an unset owner or App secret, or any failure), it comments on the pull request naming the missing piece and the command a person or session runs to re-author the bump by hand, and the drain or a review sweep lists such pull requests as waiting on that step."
---

A dependabot pull request (#1105, a go-deps bump touching only go.mod and go.sum) has sat for four days failing the attribution gate, because its re-author job failed. Nothing tells a person or an autonomous session why it failed or what to do: the pull request just shows two red checks. During the 2026-10-24 run it needed a session to diagnose it and redo the bump by hand.
