---
id: itd-2609291924469783
slug: every-lint-warning-rule-shares-one
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
related_issues: [iss-46]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Every lint warning rule shares one baseline file that only shrinks, so a new warning fails the change that introduces it

Typed links: `related_issues` [iss-46](../../../work/issues/open/iss-46-lint-scope-holes.md) (the lint scope holes record, whose one remaining hole is this baseline).

## Press Release

> **Every lint warning rule shares one baseline file that only shrinks, so a new warning fails the change that introduces it.** The warnings record-lint and docs-lint report today are frozen in one committed baseline file; a change that adds a warning the baseline does not hold fails, a change that fixes one takes it out of the baseline in the same change, and the file only ever loses entries.
>
> "We had forty warnings nobody read, and every new one hid among them," said Bob, a staff engineer on a platform team. "Now the old ones sit in one file that only gets shorter, and the one I just introduced fails my own change."

## Why This Matters

A warn rule reports and never fails, so a warning added today reads exactly
like the ones that were already there, and a rule can only be armed once its
whole backlog is cleared. iss-46 names this as the last of its five lint scope
holes: the ratchets that exist (the prose-citation baseline, the site's
unresolved-reference baseline) are each one rule's own file, and there is no
general one. The product thinker ruled on 2026-09-29 (ruling BT3) for one
shared baseline file for all warn rules, shrink-only, in which a new warning
fails; a baseline per rule and arming rules only when clean were the
alternatives not taken.

## Mechanism

We expect warnings to stop accumulating because the change that introduces
one is the change that fails, while the existing backlog stays visible in one
file a reviewer can read; shown wrong if the baseline file grows between
releases, or if warnings are silenced by moving rules out of warn rather than
by fixing them.

## What's In Scope

- **One committed baseline file** holding every current finding of every
  warn rule in record-lint and docs-lint, keyed so an unrelated edit to the
  same file does not re-key a finding.
- **The gate**: a finding the baseline does not hold fails the lint; a
  baseline entry whose finding is gone must leave the baseline in the same
  change; the file only ever loses entries.
- **Coexistence with the per-rule baselines** that exist (the prose-citation
  and site baselines): the planning interview decides whether they fold into
  the shared file or stay beside it.

## What's Out of Scope

- Blocker rules, which fail already.
- The scope matrix iss-46 also asks for (which rule covers which tree).

## Scope Conditions

- Holds for the warn rules of record-lint and docs-lint in a repository that
  commits the baseline; a repository without the file runs the rules as warn,
  unchanged.

## Acceptance Criteria

- **Given** a tree whose warn findings all appear in the baseline, **when**
  the lint runs, **then** it passes and reports the baseline's size.
- **Given** a change that introduces a warn finding the baseline does not
  hold, **when** the lint runs, **then** it fails naming the finding and the
  rule.
- **Given** a change that fixes a baselined finding but leaves its entry,
  **when** the lint runs, **then** it fails naming the stale entry.
- **Given** a change that adds an entry to the baseline, **when** the lint
  runs, **then** it fails: the file only shrinks.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [forge, CI and merge SOTA note](../../research/notes/2026-10-23-forge-ci-merge-sota.md).

- **The most recent general design keys by file, rule and count, not by
  line**: ESLint bulk suppressions (April 2025) hold
  `{ "<file>": { "<rule>": { "count": n } } }` in one committed file for all
  rules; a count exceeded reports every violation of that rule in that file;
  a count too high exits non-zero ("There are suppressions left that do not
  occur anymore") until `--prune-suppressions` runs; counts were chosen
  because there is "no reliable way to determine whether the new violations
  were introduced recently or already existed" (ESLint docs and blog). ESLint
  applies the file to error-severity rules only ("If a rule is enabled as
  `"warn"`, ESLint will not suppress the violations"), so the borrowed part is
  the file shape and the count semantics; a shrink-only ratchet over a warn
  tier has no direct precedent in these tools, and PHPStan's baseline is the
  precedent that is not tier-restricted.
- **PHPStan is the same shape** (`message`, `count`, `path`), reports an
  entry whose error is gone, warns that regenerating to admit new errors
  grows the file, and says "The life goal of a baseline file is to not
  exist." Psalm's `--update-baseline` removes fixed issues and "will not add
  new issues".
- **detekt keys by signature** (`<RuleID>:<FileName>:<Signature>`, an id
  that carries no line number, an inference from the format), and "auto
  formatting cannot be combined with the baseline" because its signatures
  would be too ambiguous: the finer key, at the cost of a signature per rule.
- **Ruff has no baseline** (inline `--add-noqa` plus `RUF100`); Betterer is
  the "only gets better" idea, with no stable release since August 2022 (5.4.0
  on npm; one alpha, 6.0.0-alpha.1, in December 2024).
- **The alternative family is new-code-only gating** (SonarQube: "do not
  recommend adding conditions for overall code"; golangci-lint's
  `new-from-merge-base`, already in the
  [lint-subset note](../../research/notes/2026-09-30-merge-blocking-lint-subset-sota.md)
  for blocking rules). No conflict: that note gated the error tier; this
  intent is the warn tier, where a committed shrinking file is the debt
  record the ratchet principle asks for, and where ESLint's precedent, built
  for the error tier, is extended rather than copied.

**Path 2**: a native shared baseline shaped to ESLint's file: one committed
JSON keyed by path, rule id and count; a new warn rule's existing findings
enter through an explicit `--suppress-rule <id>`-style admission in the
change that adds the rule, the one named exception to shrink-only; the
per-rule baselines fold in, as every precedent keeps one file for all rules;
the count key is why an unrelated edit cannot re-key a finding, and detekt's
signature key is the seam for a rule whose findings are many per file. No
dependency. The independent fit-challenge runs at plan time.

## Open Questions

- **How a new warn rule's existing findings enter the file**: shrink-only
  refuses additions, so arming a new rule over a tree that already breaks it
  needs a declared path (the rule's first freeze in the change that adds the
  rule, or no freeze at all).
- **Whether the per-rule baselines fold in** (see scope).
- (SOTA 2026-10-23) Whether a count key is fine enough for rules with many
  findings per file (prose citations), where a fix and a new finding in one
  file cancel out in the count, or such a rule takes a signature key.
- (SOTA 2026-10-23) Whether each entry carries the issue id that retires it,
  which none of the precedents do and the ratchet principle asks for.
- **The key** a finding carries in the file, so an unrelated edit does not
  read as a fixed finding plus a new one.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
