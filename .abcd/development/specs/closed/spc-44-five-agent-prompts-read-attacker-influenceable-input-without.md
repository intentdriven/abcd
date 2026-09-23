---
id: spc-44
slug: five-agent-prompts-read-attacker-influenceable-input-without
intent: itd-151
---
# five-agent-prompts-read-attacker-influenceable-input-without

## Summary

Brings the `agents/` tree — which sits outside every lint root — under
record-lint, and adds the missing detector for the itd-5 trust-contract class:
the PQ linter the `agents/README.md` names but no code implements. It enforces,
per agent prompt, that an agent reading attacker-influenceable input carries its
itd-5 trust-contract frontmatter, ships an injection-canary fixture, and gains a
per-agent changelog entry when it is added or changed in a diff.

## Scope

In:

- A new record-lint rule (e.g. `agent_contract`) in `internal/core/lint`,
  registered in `.abcd/record-lint.json`'s `rules` object, run once outside the
  per-root loop (the `agents/` tree is not a member of `cfg.Roots`).
- Git diff plumbing into the lint engine for the changelog check (new; the lint
  package has none today).
- `agents/CHANGELOG.md` as the per-agent changelog target.
- Tests in `internal/core/lint` and fixtures under `agents/*/fixtures/`.

Out:

- No change to `cfg.Roots` (`.abcd/record-lint.json` keeps `[".abcd/development"]`);
  `agents/` is walked by the new rule directly, not added as a lint root, because
  the rule enforces a different contract than the record stores' schema.
- No change to the agents' runtime behaviour or to the itd-5 discipline record
  (`.abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md`).

## Approach

**Walk `agents/`.** record-lint discovers roots from
`internal/core/lint/config.go:25` (`Roots []string`); the engine's per-root walk
(`lint.go:209`) scans only `*.md` under those roots, and `agents/` is in neither
lint root. The new rule follows the once-outside-the-loop, repo-root-scoped
pattern of `checkStrayRootDocs` (lint.go:528) and `checkDeliveryState`
(`deliverystate.go`): it enumerates `agents/*.md` itself, parsing each file's
frontmatter with the existing `frontmatterFields` (lint.go:2524) and returning
`[]Finding` (`File/Line/RuleID/Severity/Message`) as `validateSpec`
(lint.go:2131) and `validateIntent` (lint.go:1893) do. It is gated the standard
way: `if c, ok := cfg.Rules["agent_contract"]; ok && c.Enabled { … }`.

**The three sub-checks**, per the itd-5 contract documented in
`agents/README.md` (the trust-contract fields section):

1. *Trust-contract frontmatter.* An agent whose frontmatter declares
   `reads_untrusted_input: true` must also carry the contract fields
   (`prompt_version` semver, `capability_scope.task_classes`, `designed_for`).
   Missing any → a finding naming the missing field, mirroring `validateSpec`'s
   per-field messages. The five named prompts (ruthless-reviewer,
   security-reviewer, docs-currency-reviewer, intent-auditor, sota-researcher)
   are the class this catches.
2. *Injection-canary fixture.* For an untrusted-input agent, stat
   `agents/<name>/fixtures/injection-canary.json` (the injection-canary contract
   in `agents/README.md`); absent → a finding naming the missing canary. The
   rule reaches outside `*.md` for this existence check — the existing walk never
   would.
3. *Per-agent changelog entry on diff.* When an agent file is added or changed in
   the diff under lint, `agents/CHANGELOG.md` must carry a matching entry (keyed
   on the agent and its `prompt_version`). Missing → a finding naming the missing
   entry.

**Diff-awareness (new plumbing).** The lint package takes only a config and a
root today (`lint.go:159/167`) — no git range. The changelog check needs the
changed-file set, so the rule shells out with the established precedent
`gitutil.Run(root, "diff", "--name-only", …)` used by
`internal/core/changelog/guard.go:244` and `shipped.go:281`, and correlates the
changed `agents/*.md` set against `agents/CHANGELOG.md` the way
`checkDeliveryState` correlates `CHANGELOG.md` against the intents store. When no
diff range is available (a full-tree lint outside a git range), the changelog
sub-check is a no-op — it fires only over a diff, per the AC wording.

## How it satisfies each acceptance criterion

- *record-lint walks the `agents/` tree* — the new rule enumerates `agents/*.md`
  directly. Test: a fixture agent tree is linted and asserted to be visited
  (before the change, nothing under `agents/` is scanned).
- *An untrusted-input agent lacking itd-5 frontmatter fails, naming the missing
  frontmatter* — sub-check 1. Test: an agent with `reads_untrusted_input: true`
  but no `capability_scope` produces a finding naming the field.
- *An untrusted-input agent with the frontmatter but no canary fails, naming the
  missing canary* — sub-check 2. Test: remove the fixture, assert the finding
  names `injection-canary.json`.
- *An agent added/changed in a diff without a per-agent changelog entry fails,
  naming it* — sub-check 3. Test: stage an agent change with no
  `agents/CHANGELOG.md` entry over a diff range and assert the finding.
- *An agent with all three passes with no finding* — the composed rule. Test: a
  complete fixture (frontmatter + canary + changelog entry) lints clean.

## Decisions

`agents/` is linted by a dedicated rule rather than by adding it to `cfg.Roots`:
the record stores' schema rules (`record_schema`, `spec_lifecycle`) do not apply
to agent prompts, and the itd-5 contract is a different shape, so a bespoke rule
is cleaner than overloading the root walk. Diff plumbing is added narrowly for
the changelog sub-check only, reusing the `gitutil.Run` pattern already present
in the changelog package rather than inventing a new git seam.
