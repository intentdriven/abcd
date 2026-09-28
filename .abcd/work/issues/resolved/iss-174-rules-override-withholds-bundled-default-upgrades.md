---
schema_version: 1
id: "iss-174"
slug: "rules-override-withholds-bundled-default-upgrades"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "iss-156 adversarial review 2026-07-30"
found_at: "internal/core/rules/rules.go"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): For security-bearing arrays in a repo override: union with the bundled entries, a replace-vs-extend marker, or a diagnostic naming withheld entries?"
resolution: "The silence is closed: for the guardrail domains (COMMITTING, LOAD, PII) Load names every bundled recall keyword, alias and rule an override's list leaves out, with the file whose list is in force, on stderr from abcd rules and from the hook on every prompt. The comparison is against the list the running binary bundles, so an upgrade that adds an entry is named on the first load after it. The merge stays per field, as documented. Whether security-bearing lists should union or take a replace-versus-extend marker stays the product thinker's question (itd-117's finer-grained-merging follow-up; DECISIONS.md 2026-09-26)."
impact: fix
resolved_by:
  commit: "aea2ae7e"
---

mergeDomain replaces a domain's recall and rules arrays wholesale, so a repo that overrides either array silently withholds every later security upgrade to the bundled defaults: internal/core/rules/rules.go copies the override array over the base one instead of unioning, with no schema bump and no notice, so a repo that pinned PII recall or PII rules before iss-156 keeps the old set and never sees the new network keywords or the never-commit-network-identifiers rule line. Per-field merge is the documented and wanted behaviour for state, but for security-bearing arrays the quiet outcome is a stale ruleset that looks current. Options to weigh: union rather than replace for recall/aliases (additive, no loss), a distinct replace-vs-extend marker in the override, or at minimum a loud diagnostic in abcd rules naming which bundled entries an override is withholding. First activated by the iss-156 PII upgrade, which is why it is captured now. Prior disposition to weigh when triaging: iss-66 (resolved, rules-loader trust boundary) document-accepted the adjacent risk that an override can weaken a default guardrail domain (its P15); this entry is the distinct case of an override silently withholding later upgrades rather than deliberately silencing a domain.

## Grounds

- pursued: a repo or user override that holds back a bundled PII, COMMITTING or LOAD entry is told which entry and which file on every load; an override withholding one with no stderr note, or a note reaching the injected context or the --json document, would show it wrong
