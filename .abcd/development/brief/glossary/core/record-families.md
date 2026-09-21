---
term: record-families
bounded_context: core
definition: The one page that maps abcd's record families (intent, spec, step, bundle, issue, release, status) and how they relate: what each groups, what groups it, its lifecycle and the verb that moves it.
aliases: []
forbidden_synonyms: []
status: stable
introduced_in: adr-2609212115255771
starts_when: null
ends_when: null
not_to_be_confused_with: null
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# record-families

The families, and the two axes they sit on.

| Family | What it is | Groups | Grouped by | Lifecycle | Moved by |
|---|---|---|---|---|---|
| **intent** | one user-facing capability, press release first | its specs | a bundle (delivery) | drafts → planned → shipped (superseded; disciplines) | `intent plan`, `spec close`, `intent reclassify` |
| **spec** | the design record for one piece of scheduled work | its steps | its intent (one or more specs per intent) | open → closed | `intent plan` mints, `spec close` |
| **step** | one landable piece of a spec | nothing | its spec | listed, landed, or carried into the remainder | `abcd build` |
| **bundle** | intents that ship as one change | intents | nothing | named at plan, ships with its spec | `intent plan --bundle` |
| **issue** | a captured finding with a remedy; no spec by design | nothing | nothing (edges: `blocked_by`) | open → resolved / wontfix | `capture`, `capture resolve`, `drain` |
| **release** | the derived cut: version from impact, changelog from records | what shipped since the last tag | nothing | cut, tagged | `launch ship` |
| **status** | Now / Next / Later, rendered from the shelves, the gate and the build's state | nothing (a view) | nothing | none: computed | nothing |

**Two axes.** The *lifecycle* (what has been decided about a record) lives in the folders and is what the gates read. The *position* (how soon) is rendered from it as Now / Next / Later and is never stored. Sequencing is dependencies (`blocked_by`, `builds_on`) plus the shelves; nothing sits above the intent for sequence.

**Retired**: [phase](phase.md) and [milestone](milestone.md) (2026-09-21, adr-2609212115255771), and the word roadmap; a **batch** is the autonomous run's internal order, derived from dependencies and the pick, and is not a term of the record.

## When to use

When unsure which word to use, or whether a new grouping deserves a name: a name that has no row here is a name nobody has justified.
