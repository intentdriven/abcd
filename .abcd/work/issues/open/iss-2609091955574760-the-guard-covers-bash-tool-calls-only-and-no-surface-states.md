---
schema_version: 1
id: "iss-2609091955574760"
slug: "the-guard-covers-bash-tool-calls-only-and-no-surface-states"
severity: "major"
category: "security"
source: "review-followup"
found_during: "v0.8.0 release gate crosscheck round 2"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/hooks.json"
deferred_after: "v0.7.1"
deferral_reason: "A documentation gap about a real scope limit, not a defect in the guard, which behaves as designed. It is recorded rather than fixed mid-tag because the right remedy is a judgement rather than an edit: whether to state the limit and leave it, or widen the matcher so the guard adjudicates more than Bash. Naming the limit in prose is cheap; deciding the scope is not, and the second belongs to the maintainer. The waiver lapses at v0.8.0."
---

The `PreToolUse` entry in `hooks/hooks.json` carries `"matcher": "Bash"`, so
`abcd guard hook` adjudicates Bash tool calls and nothing else. A hazardous
command that reaches the host through any other tool is never checked.

That is a real limit on the protection the guard offers, and **no surface states
it**. Verified by search across the whole tree: the string does not appear in
`.abcd/development/brief/`, in `docs/`, or in `commands/guard.md`. The other four
hook entries carry no matcher at all and so see every event of their kind, which
makes the guard's narrower reach easy to miss when reading the file.

## Why the omission misleads

`04-surfaces/17-guard.md` describes the guard as reaching sessions "through a
pre-tool-use hook", "wired from `hooks/hooks.json` rather than typed", and then
spends a section on fail-open-loud: the states that can independently be false
are enumerated as hook installed, binary reachable, registry armed. A reader
takes from this that the three named states are the ways coverage can be absent.
A fourth is not named, is always true, and is not a degradation at all: the tool
was not Bash.

The user-facing side is worse, because absence there is silent. `docs/` says
nothing about the matcher, so a reader who has installed the hooks and armed the
registry has no way to learn what is outside the fence.

## The two shapes of remedy, which are not the same decision

- **State the limit.** One clause in `04-surfaces/17-guard.md` beside the
  fail-open-loud states, and one sentence wherever `docs/` describes what the
  guard protects. Cheap, and honest about today.
- **Widen the matcher.** Whether the guard should adjudicate more than Bash is a
  product question with real cost: every additional tool class needs its own
  hazard vocabulary, and a guard that refuses a tool it cannot reason about is
  worse than one that says plainly what it covers.

The first is owed regardless. The second is the maintainer's call and is why this
is recorded rather than patched.

## Acceptance

- **Given** a reader of `04-surfaces/17-guard.md`, **when** they reach the
  fail-open-loud section, **then** the Bash-only scope is stated there as a
  standing limit rather than a degradation.
- **Given** a user reading the guard's user-facing documentation, **when** they
  ask what the guard protects, **then** the answer names Bash tool calls.
