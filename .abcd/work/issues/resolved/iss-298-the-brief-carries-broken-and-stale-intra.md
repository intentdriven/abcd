---
schema_version: 1
id: "iss-298"
slug: "the-brief-carries-broken-and-stale-intra"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: ".abcd/development/brief"
resolution: "Repointed all 18 broken/stale brief heading anchors and section citations, and the three live-README legacy brief-section-number citations."
impact: fix
---

The brief carries broken and stale intra-record cross-references invisible to links_resolve (which strips the #fragment before resolving): six deep links name headings that were renamed or renumbered (04-launch.md, 04-universal-patterns.md, 01-ahoy.md, 05-intent.md sections 5/6/7), ten anchors mis-encode an em dash, 00-meta.md cites a deleted disembark section-3 budget rule, and three live README lines cite the pre-split monolithic brief by section number
## Evidence

Root cause: `internal/core/lint/lint.go` `checkLinks` strips the `#…?` fragment before
resolving and skips same-file `#` links, so `links_resolve` never validates anchors even
though it is a blocking rule (tracked separately as the systemic gap).

Broken/stale references (all in `.abcd/development/`):
- `brief/05-internals/06-lint.md` cites `05-intent.md § 5 #5-acceptance-gates…`; that heading
  is now `## 6.` (§ 5 is "Frontmatter fields").
- `brief/05-internals/{03-configuration,04-universal-patterns,09-provenance-substrate}.md` →
  `04-launch.md#2-payload-manifest-default-deny`; heading is now `## 2. Curated release
  artefact (default-deny)`.
- `brief/05-internals/05-prompt-quality.md` → `04-universal-patterns.md#2-mcp-preferred-…`;
  heading is now `## 2. Host-delegated by default, oracle adapters opt-in`.
- `brief/01-product/01-press-release.md` → `01-ahoy.md#1-acceptance` (heading is unnumbered
  `## Acceptance`); `intents/disciplines/itd-1-acceptance-gates.md` → `05-intent.md#7-the-intent-fidelity-reviewer-agent…`
  (renamed `intent-auditor`).
- Ten anchors mis-encode an em dash (single hyphen where the slug needs `--`): the
  `03-embark.md#7-voyage-layout — …` and `03-mental-model.md#the-naurian-gap — …` targets.
- `brief/00-meta.md` cites `02-disembark.md § 3 #3-agent-context-budget`; that section (and the
  word "budget") no longer exists in the target — the rule now lives in
  `05-internals/03-configuration.md` (`maxAgentTokens`).
- Three live READMEs cite the pre-split monolithic brief by section number
  (`roadmap/rfcs/README.md` "§ 11" and "§ 5"; `intents/README.md` "§ 1.5" — now the
  four-layer model in `01-product/03-mental-model.md`).

## Adversarial review

CONFIRMED (substantive + nitpick sites) by two independent refuters, which supplied the exact
corrected slugs and confirmed the frozen `planned/`, `plans/`, `research/` bodies are
accepted-as-historical and left untouched. Fix: repoint each anchor/citation; leave frozen
records alone.
