---
id: spc-16
slug: abcd-teaches-repo-agents-the-shell
intent: itd-103
---
# abcd-teaches-repo-agents-the-shell-commands-they-must-never

## Summary

spc-16 delivers the shell-hazard registry for itd-103: one bundled registry of
dangerous command patterns, surfaced through two planes. The teaching plane
injects the matched safety rules through the existing rules loader before
shell-heavy work; the guard plane is a deterministic core verb
(`abcd guard check`) that a harness hook calls at execution time to refuse a
matching command and reply with its safe successor. The design decisions below
were settled by the 2026-07-27 grill (recorded in the intent's Grill
Settlements); this spec records them together with the mechanism — it does not
reopen them.

## Settled constraints (from the grill)

- **Fail-open-loud.** If the guard hook cannot execute the abcd binary, the
  session is never blocked and never silently unguarded: the hook emits an
  unmissable in-session warning, and `abcd ahoy` status reports guard health.
- **Two tiers, mirroring the docs-lint family.** `blocker` refuses the command
  with the safe successor as the block message; `warn` lets the command pass
  with the warning injected. Entry shape follows the lint `banned_tokens`
  family: id, pattern, tier, successor, plain-language why.
- **No in-session override.** The only escape is a committed, reviewable
  per-repo config override.
- **Shell-token-aware matching in command position only.** A raw-regex guard
  would have blocked this repository's own incident-capture command (a ledger
  capture whose quoted text mentions `rm -rf *`) — the known-good corpus is
  not optional.

## Mechanism

### Registry

Bundled defaults are embedded in the binary, like the rules-loader default
domains. Each entry declares:

- `id` — stable slug (e.g. `rm-rf-after-cd-chain`).
- `pattern` — a command-position match expressed over shell tokens, not raw
  text (see Matching).
- `tier` — `blocker` or `warn`.
- `successor` — the safe form the agent is told to run instead.
- `why` — one plain-language sentence a non-expert facilitator understands.
- `fixtures` — known-bad and known-good command lines proving the entry's
  behaviour (see Admission gate).

### Config home (spec-time decision)

Per-repo overrides live in a dedicated committed file, `.abcd/guard.json`
(`schema_version`, `disabled`, `entries` keyed by id for per-field override or
addition) — not in a new `.abcd/rules.json` domain. Two reasons: the guard
entry schema is structured (tier/successor/fixtures) where rules-domain
entries are prose strings, and the rules loader's kill switch must not
silently disable a safety guard — the two features keep independent switches.
Disabling the guard is itself a committed, reviewable act: `"disabled": true`
in `.abcd/guard.json`.

### Two planes, one registry

- **Teaching plane (all hosts).** The rules loader gains a guard-backed safety
  domain: prompts that recall-match shell-heavy work get the matched registry
  entries rendered as rules (pattern, why, safe successor) injected before the
  agent acts. Hosts without hook support get this plane at minimum.
- **Guard plane (hosts with hooks).** `abcd guard check` is a core verb behind
  the transport-agnostic boundary: core takes a candidate command string and
  returns a decision (allow / warn+message / block+successor+why); the CLI
  front door formats it, and the host-hook adapter parses the host's hook
  payload into that call and maps the decision onto the host's block/allow
  protocol. Hook wiring is installed by ahoy for hosts that support it.
- Both planes are wired at delivery: the verb is reachable from the CLI and
  the plugin markdown surface (`commands/abcd/guard.md`), and the hook
  executes it in a live session — no dead scaffolding.

### Matching

Matching is shell-token-aware and applies in command position only:

- The candidate line is tokenised with shell quoting honoured; a hazard
  pattern inside a quoted string argument never fires (AC 3 — the
  incident-capture command is known-good fixture #1).
- Command position is tracked across compound separators (`&&`, `;`, `||`,
  pipes), so `cd scratch && rm -rf *` matches the cd-chain structure the
  registry entry describes.
- String payloads inside `eval` or `sh -c '...'` are a **documented v1 gap**,
  stated in the verb's reference doc and the registry README — not a silent
  one.

### Fail-open-loud and health

The installed hook is a thin shim: if the abcd binary is missing, fails to
execute, or exits abnormally, the shim exits with the host's allow status
while emitting an unmissable warning into the session transcript. `abcd ahoy`
status gains a guard-health line (hook installed, binary reachable, registry
loadable) so a broken guard is visible outside the session too — never a
silent no-op, never a bricked session.

### Admission gate (bundled defaults)

A registry entry is admitted to the bundled defaults only when:

- it ships known-bad and known-good fixtures, with known-good at least 40% of
  the entry's corpus;
- it clears the declared true-negative-rate floor: **100% on its known-good
  corpus** — a single known-good fixture that fires blocks admission. The
  floor is enforced by a test over the embedded corpus, so an entry cannot
  ship without its proof.

The registry grows from reality: `abcd capture` is the single move a
facilitator needs, and recurring captures are promoted into the bundled
defaults through this gate.

## Acceptance-criteria mapping

- AC 1 (fail open loud, health in ahoy) → Fail-open-loud and health.
- AC 2 (blocker refuses with successor, warn passes, committed override only)
  → Registry tiers + Config home.
- AC 3 (quoted-string non-firing, command position, cd-chain) → Matching.
- AC 4 (fixture corpus, ≥40% known-good, TNR floor, documented eval gap) →
  Admission gate + Matching.

## Out of scope

- Non-shell tool calls (a later guard-family question, per the intent).
- Parsing string payloads inside `eval` / `shell -c` (documented v1 gap).
- Any scheduled or CI-side execution of the guard.
