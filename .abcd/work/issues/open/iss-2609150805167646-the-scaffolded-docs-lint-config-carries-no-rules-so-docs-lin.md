---
schema_version: 1
id: "iss-2609150805167646"
slug: "the-scaffolded-docs-lint-config-carries-no-rules-so-docs-lin"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "peer report from a downstream repo, 2026-09-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/banlist_scaffold.go"
---

`abcd docs lint` reports "0 finding(s), 0 blocker(s)" in a scaffolded repository
having run no rules at all. A green result that means nothing is worse than a red
one, and this is abcd's own tool breaking abcd's own stated principle.

## Measured, in a repository abcd prepared

Reported by a session working in a downstream repository, with the numbers.

Before: `abcd docs lint` returned 0 findings. Proven vacuous by appending
"Previously this was different." — a present-tense violation — to a file
**inside the configured roots**, and getting 0 findings again.

After copying abcd's own rule set and widening the roots to the repository's
real prose surfaces: **545 findings, 535 blockers.** By rule: 419
em-dash-in-list-item, 112 `harness/*`, 6 spelling, 5 `present_tense`, 2
`stray_root_docs`, 1 `links_resolve`.

Every "0 findings, 0 blockers" that repository's sessions reported to their user
this week was vacuous.

## The cause, and the distinction the report did not draw

`publicFamilySeed` (`internal/core/ahoy/banlist_scaffold.go:89-95`) is what a
repository with no docs-lint config inherits:

```json
{
  "roots": ["docs", "README.md"],
  "banned_tokens": [],
  "rules": {},
  "exempt_paths": [],
  "exempt_if_status": []
}
```

The comment above it justifies the emptiness:

> Empty is the point — abcd cannot know which names a repo may not publish, and
> seeding a ban nobody declared would fail a build over a word the maintainer
> never chose.

**That reasoning is correct, and it covers exactly one of the two empty fields.**

`banned_tokens` is the repository's own private names. abcd genuinely cannot know
them, and seeding one would fail a build over a word nobody chose. Empty is
right, and the scaffolded private stub's commented examples make the same
argument the same way.

`rules` is not that. The rules are abcd's OWN Writing-Guide rules —
`links_resolve`, `present_tense`, `stray_root_docs`, `harness_leak`, the
em-dash-in-list-item token, the spelling family. They are not anybody's secrets,
they are the machine-enforced half of a guide abcd ships and documents. One
justification has been applied to two fields, and only one of them earns it.

## Why this is the principle abcd states, broken by abcd

`loud-staging`: "a stage that no-ops or degrades must say so; never manufacture a
false green." A lint that ran zero rules and prints "0 finding(s), 0 blocker(s)"
manufactures a false green by that definition exactly.

The second-order cost is the one the report names: the Writing Guide labels those
rule families "machine-enforced". In a scaffolded repository that label is true of
the rule and false of the corpus, and a reader has no way to tell. They are not
misreading the guide; the guide is describing a gate that is not running.

## Two fixes, and they are not alternatives

1. **Say what ran.** `docs lint` must not report "0 findings" when it configured
   zero rules. "no token rules configured — nothing was checked" is the honest
   render, and it is the loud-staging remedy: the degradation announces itself.
   This is the one that generalises, because it holds however the config got
   into that state.
2. **Scaffold the rules.** Seed `rules` with the canonical families and leave
   `banned_tokens` empty, since the first is abcd's and the second is the
   repository's. A repository that wants a family off turns it off deliberately,
   which is a decision with a record rather than an absence nobody chose.

Do (1) regardless. (2) is the maintainer's call, and it carries a fit question
recorded below.

## The fit question the same report raises

Of the 545 findings, 112 were `harness/*` — the rule that refuses naming a
specific bundled tool in user-facing prose. In that repository the finding is a
false positive by construction: it is teaching material *about* those tools, so
naming them is the content. The reporting session dropped that family and
recorded why.

"Names a specific tool" is right for abcd's published surface and wrong for a
course that teaches those tools, and 112 false positives is what a wrong rule
costs the reader's trust in the other 433. So scaffolding the rules wholesale
would import a family that cannot fit every repository. Whatever (2) does should
offer the harness family as a per-repository fit decision rather than assume it.

## Also reported, and separable

`ahoy` wrote `CLAUDE.md` and `AGENTS.md` into that repository as two identical
copies, where abcd's own root carries `CLAUDE.md` as a symlink to `AGENTS.md`.
Two copies drift and a symlink cannot. The scaffolded `stray_root_docs`
allowlist then named neither, so the lint flagged both files abcd had just
written. Worth its own record if the maintainer wants it separated; it is noted
here because it came from the same scaffold pass and has the same shape — the
scaffold writing something the scaffold's own gate then refuses.

## Acceptance

- **Given** a docs-lint configuration with no rules, **when** `abcd docs lint`
  runs, **then** it says nothing was checked and does not report a finding count
  that implies it was.
- **Given** a freshly prepared repository, **when** a present-tense violation is
  written inside a configured root, **then** the lint finds it.
