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
---

The generated SHELL rules domain recalls on the prose verb kill: the registry's kill-by-search entry contributes the bare head kill to the domain's recall terms, so an ordinary prompt such as 'kill the feature flag, we are not shipping it' injects the whole SHELL block (about 2k tokens, once per session per signature). rm, pkill, killall and git push are shell-only words; kill is an everyday English verb, the class the domain's own fixed aliases already keep out (push, commit, reset). A rules-side exclusion list would be the second source ruling J10 forbids.
