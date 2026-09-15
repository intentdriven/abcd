---
schema_version: 1
id: "iss-2609012039215212"
slug: "ghsa-22f8-qf5r-gjgq-cwe-451-cwe-1427-a-abcd-rules-json-overr"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/rules.go"
resolution: "Every rule domain carries its provenance, derived at merge time with no rules.json schema change. Merge records each domain the override names (rules replaced, state changed, or a custom domain; conservatively, since the effective behaviour is repo-chosen) and Match, Active and Lookup set ResolvedDomain.Source to \"repo\" for those and \"bundled\" otherwise; the label set is open so the spc-23 user layer (AC6) can add \"user\" without renaming. renderDomain emits \"## NAME (repo override)\" for a repo-sourced domain, so the marker lands in the agent context, in abcd rules and inside the dedup Signature; InjectResult carries Sources and Labels(), and the prompt-router diagnostic prints the labelled names; rules --json carries the source field; the rules verb Long text, the reference page, the ahoy marker block and AGENTS.md say so. One-time effect: the signature moves for overridden domains only, so each re-injects once on the first post-upgrade prompt. Named out of scope: the guard registry merge has the same provenance-less shape for guard.json overrides but its surface is a block/allow decision. Proven by TestMergeRecordsRepoOrigin (watched failing to compile), TestRenderMarksRepoOverride, TestInjectLabelsRepoOverrides (core), TestRulesJSONCarriesSource, TestRulesTextMarksRepoOverride and TestHookPromptRouterDiagnosticNamesOverrides (cli)."
impact: fix
---

GHSA-22f8-qf5r-gjgq (CWE-451, CWE-1427): a `.abcd/rules.json` override replaces a bundled domain's rules wholesale (`mergeDomain`, internal/core/rules/rules.go) and nothing downstream records where the words came from — `ResolvedDomain` carries no origin, `renderDomain` emits name and bullets only, the prompt-router diagnostic prints names, and `abcd rules --json` marshals the same origin-less shape — so a hostile or compromised checkout injects attacker-chosen text under `## PII` and the tool's own `# abcd rules` header, byte-indistinguishable from a bundled default in every consumer-visible surface. Reproduced at v0.7.0. The fix must derive provenance at merge time with no rules.json schema change: a `source` field on `ResolvedDomain` valued `bundled` or `repo` (leaving room for `user`, spc-23 AC6), the rendered heading `## NAME (repo override)` for a repo-sourced domain so the marker lands in the agent's context, in `abcd rules` and inside the dedup signature, the source per injected name on the hook's stderr diagnostic, and the field in `rules --json`. The marker moves `Signature` for overridden domains only: a one-time re-injection on the first post-upgrade prompt. Named as out of scope: the guard registry merge (internal/core/guard/config.go) has the same provenance-less shape for `.abcd/guard.json` overrides, but its surface is a block/allow decision, not injected prose. Siblings captured separately: `"rules": []` injecting a heading-only block, and a stop-word recall keyword firing on ordinary prose. iss-174 (an override withholds bundled upgrades) stays open and distinct. Advisory severity medium.

## Grounds

- pursued: provenance, not permission, is the fix the advisory and spc-23 both call for; the origin is only knowable at merge, so it is recorded there and read by every consumer through one field
