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

## Open Questions

- **Which layer may select the back end**: the machine-wide file, the
  repository's file, or both, and which wins.
- **How the domains reach CARL**: abcd writes CARL's configuration, or CARL
  reads abcd's rules files.
- **What `abcd rules` renders** when CARL is selected.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
