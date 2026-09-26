---
schema_version: 1
id: "iss-2609251522588539"
slug: "agents-md-s-foreign-uid-roots-paragraph-says-a-refused"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.10.0"
deferral_reason: "The posture fix (refuse the read at the working directory as well as the walk) is on the unmerged branch feat/ruled-security-forks (da3efea6) and spans the rules loader, the guard and the banlist; fix round 1 of the tier2 lane corrected the wording to the truth instead of landing a partial second form of it. Recorded in .abcd/work/DECISIONS.md, 2026-09-25."
resolution: "Every statement of the foreign-uid refusal now says what holds: the refusal bounds the walk, not the working directory, so a .abcd at the working directory is still read. 0434d475 corrected AGENTS.md and the marker block ahoy writes; ec5ca74f corrected the remaining three statements of the same claim, the refusal note itself (which now says the working directory's .abcd IS read where there is one), the configuration chapter and the install how-to. The posture change that would make the read at the working directory refuse too stays deferred as recorded in .abcd/work/DECISIONS.md on 2026-09-25; this record was the wording."
impact: fix
resolved_by:
  commit: "ec5ca74f"
---

AGENTS.md's Foreign-uid roots paragraph says a refused foreign-owned root sends the session to its own working directory on the bundled defaults, but rules.Resolve returns Resolution{Root: cwd} on that refusal (internal/core/rules/root.go:138), so when the working directory is the refused root, or lies inside it and carries its own .abcd/, that directory's .abcd/rules.json, .abcd/guard.json and .abcd/config/oracle-routing.json are read. The ownership gate bounds the upward walk, not the read at cwd. layered.RootsFor's comment states this correctly. The posture fix (an empty root with PerRepo false, so no per-repo .abcd is read in any geometry) is on the unmerged branch feat/ruled-security-forks (da3efea6), not on main.

## Grounds

- pursued: no surface tells a session its working directory's .abcd went unread while the loaders read it; a refusal note, AGENTS.md, the chapter or the how-to promising the bundled defaults at a refused root that carries a .abcd would show it wrong
