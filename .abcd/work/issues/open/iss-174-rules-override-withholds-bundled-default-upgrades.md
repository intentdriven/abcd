---
schema_version: 1
id: "iss-174"
slug: "rules-override-withholds-bundled-default-upgrades"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "iss-156 adversarial review 2026-07-30"
found_at: "internal/core/rules/rules.go"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed on override merge for security-bearing arrays: union, a replace-vs-extend marker, or a diagnostic naming withheld bundled entries). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

mergeDomain replaces a domain's recall and rules arrays wholesale, so a repo that overrides either array silently withholds every later security upgrade to the bundled defaults: internal/core/rules/rules.go copies the override array over the base one instead of unioning, with no schema bump and no notice, so a repo that pinned PII recall or PII rules before iss-156 keeps the old set and never sees the new network keywords or the never-commit-network-identifiers rule line. Per-field merge is the documented and wanted behaviour for state, but for security-bearing arrays the quiet outcome is a stale ruleset that looks current. Options to weigh: union rather than replace for recall/aliases (additive, no loss), a distinct replace-vs-extend marker in the override, or at minimum a loud diagnostic in abcd rules naming which bundled entries an override is withholding. First activated by the iss-156 PII upgrade, which is why it is captured now. Prior disposition to weigh when triaging: iss-66 (resolved, rules-loader trust boundary) document-accepted the adjacent risk that an override can weaken a default guardrail domain (its P15); this entry is the distinct case of an override silently withholding later upgrades rather than deliberately silencing a domain.