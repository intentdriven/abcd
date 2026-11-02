---
id: itd-2609292109214516
slug: a-person-who-already-runs-carl-can-route
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-3]
related_issues: [iss-64]
related_adrs: [adr-22]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Rule injection is a seam: the native loader stays the default, and an opt-in CARL back end can take it over

Typed links: `related_issues` [iss-64](../../../work/issues/open/iss-64-rules-seam-carl-adapter.md) (the record this draft plans); `builds_on` [itd-3](../shipped/itd-3-modular-rules-loader.md) (the native rules loader that stays the default); `related_adrs` [adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md) (native default, an external tool as an opt-in adapter).

## Press Release

> A person who already runs CARL can route abcd's rule injection through it, and everyone else keeps abcd's own rules loader. Rule injection becomes a seam with one setting: the native loader is the default and the floor, an opt-in CARL back end takes the same domains when the person selects it, and a selected back end that is missing falls back to the native loader and says so. "CARL already carries my rules everywhere else," said Kira, a DX engineer. "Now abcd's rules go through it too, and when I switch it off nothing changes for anyone else."

## Why This Matters

abcd's rules loader (itd-3) is abcd's in-plugin adoption of CARL's recall
engine: domain-keyed keyword recall at prompt submission. A person who already
runs CARL has two engines doing the same job with two configurations. iss-64
asks for the pattern adr-22 records, a native default with an easy route to a
better external tool: rule injection as a seam, the native loader as the floor,
and an opt-in CARL back end selected by one setting. The product thinker ruled
on 2026-09-29 to plan it, with the native loader staying the default (ruling
J22).

## What's In Scope

- **The seam**: rule injection behind one interface, with the native loader as
  its first implementation and its default.
- **The setting**: one field (`rules.backend: native | carl` in iss-64's
  sketch) in the rules configuration the layered files already carry.
- **The CARL back end**: the same domains, the same explicit activation and the
  same kill switch, delivered through CARL when it is selected.
- **Loud fallback**: a selected back end that is not installed falls back to the
  native loader and says so once per session.

## What's Out of Scope

- Changing what the native loader does for anyone who does not opt in.
- CARL's memory half, which itd-39 drafts on its own.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

Seeded by the run from iss-64 and the ruling, unconfirmed: the planning
interview walks each one.

- **Given** no back end is set, **when** a prompt is submitted, **then** the
  native loader injects exactly what it injects today.
- **Given** the CARL back end is selected and installed, **when** a prompt
  matches a domain, **then** the domain's rules reach the session through CARL
  and the native loader injects nothing for it.
- **Given** the CARL back end is selected and not installed, **when** a prompt
  is submitted, **then** the native loader runs and the session is told once
  that the selected back end is missing.
- **Given** the kill switch is set, **when** a prompt is submitted, **then**
  neither back end injects anything.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [harness, interop and lifeboats SOTA note](../../research/notes/2026-10-23-harness-interop-lifeboats-sota.md).

- CARL is "Context Augmentation & Reinforcement Layer" (ChristopherKahler/carl, MIT): a Claude Code hook scans each prompt against each domain's `recall` keywords and injects matching rules; `carl.json` (`version: 1`) holds `config`, `domains` (each `state`, `always_on`, `recall`, `exclude`, `rules` of `{id,text,added}`, `decisions`) and `staging`; global config in `~/.carl/`, project in `./.carl/`, merged up the tree with project winning; injection is skipped while the signature is unchanged and forced every five prompts; an MCP server exposes `carl_v2_create_domain`, `carl_v2_replace_rules`, `carl_v2_toggle_domain` and others; install `npx carl-core`.
- CARL and abcd's loader share the design (domain-keyed keyword recall at prompt submission, explicit activation, dedup); CARL also carries decisions and a staging queue, which abcd keeps in `abcd decide` records, so the seam is rules only, as scoped.
- Two hooks on one event both run (Claude Code runs every matching hook; Codex "loads all matching hooks"): with CARL selected, abcd's hook must emit nothing for the domains CARL carries, or both inject.
- CARL is documented for Claude Code only; the native loader as floor is what keeps other hosts whole. A separate project advertises an engine that "replaces the legacy CARL system"; whether that is the same CARL is unverified, which argues for checking the selected back end's presence at every session start, as the loud fallback already requires.
- Enforcement is CARL's: "rules reach the session through CARL" is not provable from abcd's hook; the acceptance check is a canary word, as the AGENTS.md re-check proposes for conventions files.

**Path 2 — a native seam whose CARL back end renders abcd's resolved domains into CARL's documented layers (generated and marked, written only when absent or abcd's own) while abcd's hook stays silent for them; selection in the machine-wide file, the repository file able to force `native`.** Two things for the fit-challenge: layer the render as CARL does, bundled and machine-wide domains into `~/.carl/` and only the repository's domains into `./.carl/`, so a machine-wide rule never lands in a working tree where it can be committed; and whether a generated `./.carl/carl.json` is a tool-specific conventions file under itd-2610030814013772's decision 7, and a write into the person's tree under the-users-directory-is-theirs. The independent fit-challenge runs at plan time.

## Open Questions

- **Which layer may select the back end**: the machine-wide file, the
  repository's file, or both, and which wins.
- **How the domains reach CARL**: abcd writes CARL's configuration, or CARL
  reads abcd's rules files.
- **What `abcd rules` renders** when CARL is selected.
- (SOTA 2026-10-23) When `./.carl/carl.json` already exists and is the person's own, does abcd refuse, merge under a marker, or write a sibling file CARL will not read?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
