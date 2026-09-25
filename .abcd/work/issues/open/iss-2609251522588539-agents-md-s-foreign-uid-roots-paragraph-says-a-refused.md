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
---

AGENTS.md's Foreign-uid roots paragraph says a refused foreign-owned root sends the session to its own working directory on the bundled defaults, but rules.Resolve returns Resolution{Root: cwd} on that refusal (internal/core/rules/root.go:138), so when the working directory is the refused root, or lies inside it and carries its own .abcd/, that directory's .abcd/rules.json, .abcd/guard.json and .abcd/config/oracle-routing.json are read. The ownership gate bounds the upward walk, not the read at cwd. layered.RootsFor's comment states this correctly. The posture fix (an empty root with PerRepo false, so no per-repo .abcd is read in any geometry) is on the unmerged branch feat/ruled-security-forks (da3efea6), not on main.
