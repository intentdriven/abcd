---
schema_version: 1
id: "iss-2608271804497507"
slug: "plugin-record-trees-are-held-only-by-readme-prose"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: "agents/README.md"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Add a record-lint root over agents/, commands/ and hooks/, or rule them prose-governed?"
resolution: "Already gated at e792a2314. The agents/ invariants the record names (itd-5 frontmatter per prompt, an injection-canary fixture for every untrusted-input agent, a per-agent changelog entry) are enforced by record-lint's agent_contract rule (internal/core/lint/agentcontract.go, a825ec869, 2026-08-28, one day after this record; shipped in v0.6.8), enabled as a blocker in .abcd/record-lint.json. commands/ is held to the brief surface registry by surface_coverage and index_drift, hooks/hooks.json by TestHooksManifestNamesLiveSubverbs (internal/surface/cli/hook_plane_test.go), and .claude-plugin/plugin.json's version by the launch lockstep tests. The record's first remedy, a gate with rules for those invariants, is the one that was built."
impact: internal
shipped_in: v0.6.8
resolved_by:
  commit: "a825ec869"
---

the plugin record trees are held only by README prose: agents/ is a second record family with its own constitution (itd-5 frontmatter per prompt, an injection-canary fixture for every untrusted-input agent, a per-agent changelog) and commands/, hooks/, .claude-plugin/ carry similar structure, but no lint root or gate checks any of it — currently conformant, enforced by nothing. Decide the gate: either a record-lint root over the plugin trees with rules for the frontmatter/fixture/changelog invariants, or an explicit ruling that these trees stay prose-governed.

## Grounds

- pursued: the plugin trees are no longer held by README prose alone, because each invariant the record lists has a rule or a test that fails on it; an agent prompt without its trust fields, canary or changelog entry that record-lint still passes would show this wrong.
