---
schema_version: 1
id: "iss-2610020727177336"
slug: "the-implement-loop-s-pull-request-body-carries-no"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Waits on the technical facilitator's ruling on which disclosure the loop's pull-request body makes (PC1 names commits only): the recommended form ends the body with Assisted-by: abcd:<version> (abcd composed the text) followed by the Assisted-by lines the landing's records commit derives from the lane's receipts (the change carries their work, and a squash merge may adopt the body), refusing the step as the records commit does when a receipt reports no model; test the exact trailer lines in prBody and run the body through scripts/check-attribution.sh body in a test."
resolution: "The implement loop's pull-request body ends with Assisted-by: abcd:<version> (abcd composed its text) and then one Assisted-by: line per distinct model the lane's receipts reported, per ruling R4; the step refuses before the forge is called when a receipt reports no model or names abcd, as the records commit does."
impact: fix
resolved_by:
  commit: "c183a0a3d"
---

The implement loop's pull-request body carries no Assisted-by trailer, so the required attribution check refuses every pull request the loop opens: internal/core/implement/loop/land.go prBody ends with only the Delivers: and Resolves: trailers, and .github/workflows/attribution.yml runs scripts/check-attribution.sh body on every pull_request event not authored by a bot, which fails a body with no Assisted-by line. The body is text abcd computes from the run's records (its first sentence says so), but the change it describes carries the implementer's model work, and a squash merge can take the body as the commit message. Latent until the loop's first live landing in a repository running the gate.

## Grounds

- pursued: we expect every body the loop opens a pull request with to pass scripts/check-attribution.sh body and end with the label followed by the receipts' models; a loop-opened body the gate refuses, or one ending without the label or a model line, would show it wrong.
