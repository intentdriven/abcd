---
schema_version: 1
id: "iss-391"
slug: "research-notes-readme-routes-phase"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: ".abcd/development/research/notes/README.md"
resolution: "route phase-scoped research to notes/ and fix the Related siblings/hrefs"
impact: fix
---

research notes README routes phase-scoped research to research/phase/N/ and names research/adr/ as a sibling — both layouts adr-30 rejected and the gated research-children index denies — and its Related display texts sit one level short of their hrefs
## Evidence

- `.abcd/development/research/notes/README.md:24` routes "Phase-scoped research" to `research/phase/<N>/`; `:48` names `phase/` and `adr/` as sibling research directories. `research/` holds exactly `notes/` and `prompting/`; adr-30 (`decisions/adrs/0030-record-information-architecture.md:66-67`) names both phantom layouts as the rejected alternative, and `development/README.md:18-19`'s blocker-gated `research-children` index asserts the real children one directory up.
- Both sites evade `links_resolve`: `:24` is a backticked code span; `:48`'s href is `..`, which resolves. The Related list's display texts (`../decisions/adrs/`, `../intents/`, `../research/`) sit one level short of their hrefs (`../../decisions/adrs`, `../../intents`, `..`), which are the correct ones from `research/notes/`.
- Refuter verdict: CONFIRMED (minor) — iss-352 enumerated three other sites, iss-353 edited this very file but only its naming section (the phantom routing survived that edit), iss-333 covered the brief's copy. The line-27 `.abcd/logbook/` route was REFUTED as prior art (owned by open iss-56, deliberately unadjudicated) and is untouched by this round's fix.
