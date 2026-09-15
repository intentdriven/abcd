---
schema_version: 1
id: "iss-2609012039210402"
slug: "an-override-pii-rules-passes-validate-internal-core-rules-ru"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/rules.go"
resolution: "A domain whose merged rules are empty no longer renders a heading-only block. Load DROPS it and keeps the rest of the file, with one stderr diagnostic per dropped domain naming it, the file, and the deliberate route (\"state\": \"dormant\"); both front doors — the prompt-router hook and the `abcd rules` verb — print those notes out of band, never into the injected context. Validate still refuses the shape outright, because Validate is what guards the BUNDLED defaults, where a ruleless domain is a build error. The first version of this fix refused the whole file: that made an existing rules.json using {\"rules\": []} stop ALL injection on upgrade — safety-shaped domains included — over one malformed entry, which the security review called disproportionate, and it was: fail-closed is right for a file abcd ships and wrong for a file a user already has. The bundled defaults and a dormant state-only override are untouched. Proven by TestValidateRefusesDomainWithoutRules, TestLoadSkipsARulelessDomainAndKeepsTheRest and TestHookPromptRouterNamesASkippedDomain, each watched failing before its change. AGENTS.md and the ahoy marker-block template state the skip."
impact: fix
---

An override `{"PII": {"rules": []}}` passes `Validate` (internal/core/rules/rules.go) and injects a heading-only `## PII` block — suppression wearing the domain's name, which the agent reads as a domain that says nothing; a custom domain declared with recall but no rules does the same. Reproduced at v0.7.0 (a 43-byte block, exit 0). The documented way to silence a domain is `{"state": "dormant"}`; a domain whose merged rules are empty should not render at all, and the loader should say which domain it dropped and name the dormant remedy, in line with the empty-rule-body refusal (iss-2608261550497978). Sibling of GHSA-22f8-qf5r-gjgq.

## Grounds

- pursued: the heading-only block must not render, and the drop must be loud — a domain that silently stops existing is the same suppression-nobody-sees the fix is about
- rejected: refusing the whole file, the shape the empty-rule-body refusal (iss-2608261550497978) set. That refusal is on a rule BODY inside an otherwise loadable domain; this one would take every other domain down with it, on an upgrade, for a config that worked yesterday. Validate keeps the refusal where it guards abcd's own bundled defaults
