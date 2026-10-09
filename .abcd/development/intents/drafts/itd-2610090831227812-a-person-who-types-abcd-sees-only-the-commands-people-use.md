---
id: itd-2610090831227812
slug: a-person-who-types-abcd-sees-only-the-commands-people-use
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# A person's /abcd: list shows only the commands people use

## Press Release

> A person who types /abcd: sees the commands people use, not the ones only agents run, such as the guard that judges a shell command or the loop that steps an autonomous run, so the list a person scans reads as their own toolbox and an agent still finds every verb it needs. The command-line help already sorts every verb into For people and For agents and hosts; the plugin's command list keeps that same classification, and `abcd --help --agent` stays the one place both lists render.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- Which host mechanism carries the split? The host offers two. Hiding a page (`user-invocable: false`) shortens the person's menu and renames nothing, but a person can no longer type that page; only the agent can run it. A subdirectory namespace (`/abcd:agent:<page>`) keeps every page typeable but does not shorten the menu, only clusters it, and renames every agent invocation across the command pages, help placements and their gates. (Adversarial review 1, finding 3, from the host's plugin documentation.)
- Does the split act at page grain only, or are agent sub-verbs moved to pages of their own? A person's page can carry agent sub-verbs: `/abcd:intent` carries `audit ingest`, `prepass` and `consistency ingest`. Page grain is the only option that changes no invocation. (Finding 2.)
- The person/agent class of every binary-backed page is already recorded in its `block:` frontmatter and gated by `TestCommandPagesDeclareTheirBlock` (itd-146; the 2026-09-25 and 2026-09-29 rulings). The audit, iss-2610090831317531, starts from that table and rules what it leaves open: the five pages with no block (`abcd`, `version`, `consult`, `ingest`, `prepare-this-repo`), the provisional placement of `drain`, the pages either side could claim (`report`, `ideate`, `dashboard`), and the keep/merge/rename/retire column no record holds. (Finding 1.)
- Hiding or namespacing a page keeps it on the plugin surface, so the boundary that every verb is reachable from both the CLI and the plugin surface (AGENTS.md, Boundaries) holds; only deleting a page would reverse it, and this intent deletes none. (Finding 4.)
- Would a later recommendation of which verb fits a situation belong to this intent, or to a separate one?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
