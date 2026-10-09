---
schema_version: 1
id: "iss-2610091942156774"
slug: "guard-allows-rm-on-unguarded-variable-path"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "2026-10-09 security-drain coordination, product thinker report"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Product thinker's ruling 2026-10-09, options 1 and 2: (1) add a guard registry entry that blocks `rm` whose operand is an unguarded variable path (`\"$VAR\"/...`, `$VAR/...`, `\"${VAR}\"/...`) and allows `\"${VAR:?}\"/...` and literal paths, with that rewrite as its successor, so the agent rewrites the command before the host prompts; prove in a live host session that the hook's refusal lands before the host's prompt; (2) teach the same rule through the SHELL domain (generated from the entry) and in abcd's agent definition files, which sub-agents read. Build after the v0.13.4 cut; never a silent rewrite of the command."
---

An agent's `rm -f "$VAR"/*` triggers the host's own dangerous-rm permission prompt ("possibly-empty variable path ... rewrite it as "${VAR:?}"/* or use a literal path"), even with permissions skipped. The person cannot tell whether the variable is empty and cannot make the agent take the suggested form, so they must approve blind. abcd's guard allows the shape today, and sub-agents never receive the injected SHELL rules, so nothing steers an agent away from it. Reported by the product thinker on 2026-10-09 from two prompts raised by a review sub-agent.
