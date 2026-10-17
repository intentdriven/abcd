---
schema_version: 1
id: "iss-2610151422073415"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/19-identity.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/19-identity.md:80 to match the shipped surface: internal/core/repolint/rule_positioning.go:54 reports an unloadable registry through targetRefusal (internal/core/repolint/repolint.go:179), which is SeverityError ('`abcd lint identity` refuses the same registry, so this is an error'), not a warn-tier…"
---

The brief at .abcd/development/brief/04-surfaces/19-identity.md:80 says: An unloadable registry is reported as a warn-tier finding rather than as drift, so the symptom reads as a broken check rather than as a missing tagline. The shipped surface differs: internal/core/repolint/rule_positioning.go:54 reports an unloadable registry through targetRefusal (internal/core/repolint/repolint.go:179), which is SeverityError ('`abcd lint identity` refuses the same registry, so this is an error'), not a warn-tier finding; and `abcd lint identity` refuses such a registry outright. (v0.14.2 release crosscheck, false-claim, raised by opus in a14.)
