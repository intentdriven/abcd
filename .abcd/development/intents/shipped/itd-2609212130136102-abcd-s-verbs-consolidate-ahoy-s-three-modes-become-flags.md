---
id: itd-2609212130136102
slug: abcd-s-verbs-consolidate-ahoy-s-three-modes-become-flags
spec_id: spc-2609212139587510
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-146]
severity: minor
impact: breaking
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-122, itd-123, itd-124, itd-125]
related_adrs: [adr-2609212115255771]
---

# abcd's verbs consolidate: modes become flags and five checks become one lint

## Press Release

> **abcd's command list loses its modes-as-verbs and its five spellings of "check this repository", so the person's list holds about a dozen verbs.**
>
> "Twenty-four verbs, and three of them were the same check wearing different hats," said a product thinker reading `abcd --help`. "Now `lint` is the check, `ahoy` has flags instead of sub-verbs for its modes, and `--version` is where every tool keeps it. I can hold the list."

## Why This Matters

On 2026-09-21 the product thinker asked whether fifty-three verbs all make sense. The command-line guidelines the field converges on say: a sub-verb for a distinct action, a flag for a mode of the same action, and a top-level list a newcomer can hold. Three of `ahoy`'s sub-verbs are modes; `version` is a flag everywhere else; `intent new` is a dead alias; and `lint`, `docs lint`, `lint outbound`, `site check` and `identity render` are five spellings of one act. Together with the agent block (itd-146) the person's list falls to about fourteen. The cut is breaking, in the same major as the four rename intents already in the run.

## Mechanism

We expect a person to hold a list of about a dozen verbs and stop asking which of five checks to run, because the checks become one verb with targets and the modes stop looking like actions; shown wrong if the same questions recur after it ships.

## Scope Conditions

None stated.

## What's In Scope

- **Modes to flags**: `ahoy dry-run`, `ahoy identity-check`, `ahoy remote` become `ahoy --dry-run`, `--identity`, `--remote`; the old spellings answer with the new one and exit non-zero for one release.
- **`--version`** replaces `abcd version`; `version --check` becomes `update --check`; `intent new` is removed.
- **One lint with targets**: `abcd lint` (all), `lint docs`, `lint outbound`, `lint site`, `lint identity`; `docs cite` and `site build` stay, because they write.
- **The record of the move**: every moved spelling in the surface snapshot with its successor; the command pages and the brief's surface chapters say the new forms only; the release derives as breaking.
- **The count**: the person's default list (itd-146) is at most fourteen verbs after this ships.

## What's Out of Scope

- The people/agent split (itd-146).
- Renaming any record family verb (`capture`, `intent`, `spec`).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. Modes to flags, `--version`, the dead alias removed, and one lint with targets, in one breaking change (ruled 2026-09-21).

## Open Questions

_None open._

## Acceptance Criteria

- **Given** `ahoy dry-run`, `ahoy identity-check` or `ahoy remote`, **when** run after this ships, **then** each answers naming its flag form and exits non-zero, and the flag form does what the sub-verb did.
- **Given** `abcd --version`, **when** run, **then** it prints what `abcd version` printed; `abcd version` answers naming the flag; `update --check` does what `version --check` did; `intent new` is unknown.
- **Given** `abcd lint docs`, `lint outbound`, `lint site` and `lint identity`, **when** run, **then** each does what its old spelling did, `abcd lint` runs them all, and `docs cite` and `site build` are unchanged.
- **Given** the surface snapshot, **when** regenerated, **then** every moved spelling is recorded with its successor, the pages and the brief say the new forms only, and the release derives as breaking.
- **Given** `abcd --help`, **when** it renders after itd-146 and this ship, **then** the person's list counts at most fourteen verbs.

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-dd80f3fcae80 -->
Fidelity review OWED (receipt rcp-dd80f3fcae80).

## Grounds

- pursued: batch 3 already carries four breaking renames, so this lands in the same major cut at no extra cost to adopters; we expect the person's list to be held and the which-check question to stop; shown wrong if the same questions recur
