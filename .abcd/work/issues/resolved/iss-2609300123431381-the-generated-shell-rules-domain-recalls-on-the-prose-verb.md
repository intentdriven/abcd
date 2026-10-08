---
schema_version: 1
id: "iss-2609300123431381"
slug: "the-generated-shell-rules-domain-recalls-on-the-prose-verb"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25 (lane integ24a, follow-up t2 of review-teachPlane)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/teach.go"
remedy: "Mark an entry's command head as an everyday verb in the guard registry itself (a registry-side marker read by RecallTerms), so the generated recall carries such a head only with an operand or flag shape and the bare word no longer recalls SHELL; the single source stays the registry, as J10 requires."
resolution: "A guard entry can carry everyday_verb, a registry-side mark RecallTerms reads: a marked head recalls SHELL only joined to a flag or operand shape its pattern declares (kill $(pgrep, kill $(pidof, xargs kill), never bare. The bundled kill-by-search entry carries it, so prose such as 'kill the feature flag' no longer recalls SHELL."
impact: fix
resolved_by:
  commit: "32ef74486c47d325fe513bc27b8e4df501d15c36"
---

The generated SHELL rules domain recalls on the prose verb kill: the registry's kill-by-search entry contributes the bare head kill to the domain's recall terms, so an ordinary prompt such as 'kill the feature flag, we are not shipping it' injects the whole SHELL block (about 2k tokens, once per session per signature). rm, pkill, killall and git push are shell-only words; kill is an everyday English verb, the class the domain's own fixed aliases already keep out (push, commit, reset). A rules-side exclusion list would be the second source ruling J10 forbids.

## Grounds

- pursued: the bundled SHELL recall no longer contains the bare word kill, so a prose prompt using kill does not recall SHELL, while kill $(pgrep ...) and pgrep ... | xargs kill still do; it would be shown wrong by TestShellDomainRecallsShellHeavyPrompts recalling SHELL on 'kill the feature flag, we are not shipping it', or by the bundled RecallTerms listing kill bare
