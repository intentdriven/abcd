---
schema_version: 1
id: "iss-2610111815024967"
slug: "guard-reads-bare-pushd-as-home"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security fix lane G round 3, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Treat a bare pushd (and pushd +N/-N) as leaving the directory unknown unless the line itself fixed the directory stack, never as HOME; prove with a Check test over a planted home and stack layout, watched failing first"
resolution: "A pushd with no operand (and pushd -q or -- alone) now swaps to the directory the line's own pushd left, tracked as the line's directory stack, and leaves the directory unknown where the line did not fix the stack, never HOME; a popd, a dirs with an operand, a cd (zsh's AUTO_PUSHD) or a broken chain unfixes the stack, and pushd +N and -N stay unknown (TestBarePushdIsNotHome over a planted home and stack layout)."
impact: fix
---

The guard's script reader treats `pushd` with no operand as going to HOME, but bash swaps the top two directories of the stack instead (zsh does the same unless an option sends it home), so the scripts that follow can be read in the wrong directory. Noticed by guard round 3 on 2026-10-11 while fixing the zsh cd substitution form (iss-2610111809142536); predates it.
