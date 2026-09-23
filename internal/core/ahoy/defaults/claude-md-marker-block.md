<!--
  Managed by abcd (Agent-Based Configuration for Development).
  Do NOT hand-edit content inside the abcd-managed fences — `/abcd:ahoy`
  silently overwrites this block on drift (per itd-3). Per-repo rule
  customisation goes in <repo>/.abcd/rules.json instead.
-->

## abcd rule loader

This repository uses the abcd modular rules loader. On `UserPromptSubmit`, a hook
recall-matches the prompt against keyword triggers declared in the plugin-bundled
default domains and `<repo>/.abcd/rules.json`, and injects only the matched
domain rules into context — instead of force-loading the full ruleset every turn.
A prompt that matches no domain injects nothing (zero added tokens).

- Inspect rules: `abcd rules` renders the active set; `abcd rules <DOMAIN>`
  (case-insensitive) scopes to one domain.
- Per-repo overrides: edit `<repo>/.abcd/rules.json`. It is
  `{"schema_version": 1, "disabled": false, "domains": {}}` — add a domain key to
  override a default per-field (e.g. `{"ROADMAP": {"state": "dormant"}}` silences
  it while keeping its rules) or to declare a custom domain
  (`{"recall": [...], "rules": [...]}`). A domain left with no rules at all
  (`{"rules": []}`, or a custom domain declared without any) is SKIPPED with a
  diagnostic on stderr naming it — it would otherwise inject a heading-only
  block, which reads as a domain that says nothing. The rest of the file still
  loads; `{"state": "dormant"}` is the way to silence a domain deliberately.
- Provenance: a domain the override names (rules replaced, state changed, or a
  custom domain) renders as `## NAME (repo override)` wherever it appears: the
  injected block, `abcd rules`, and the hook's diagnostic; `abcd rules --json`
  carries `"source": "repo"` for it and `"source": "bundled"` for an untouched
  default.
- Kill switch: set `"disabled": true` at the top of `.abcd/rules.json`.
- Foreign-uid roots: the loader and the shell guard read `.abcd/` from the
  repository root resolved for the session, never from a directory above the
  working tree. Where git cannot answer for that tree — a checkout owned by
  another uid, a container bind mount — the root is recovered from the `.git`
  marker instead, and a root the caller does not own is REFUSED: the session
  falls back to its own working directory on the bundled defaults, and one line
  on stderr names what was refused. Re-admit such a checkout deliberately, from
  an account you control:
  `mkdir -p ~/.abcd && printf '%s\n' '<checkout>' >> ~/.abcd/trusted-roots`
  (one absolute path per line; `#` starts a comment). Only your home declares
  it — a file inside the checkout can never vouch for the checkout.
- Explicit activation: start a prompt with `*<DOMAIN>` (e.g. `*COMMITTING`,
  `*PII`) to inject that domain unconditionally — overrides a `dormant` state,
  but never the kill switch.

### Default domains

`COMMITTING`, `DOCUMENTATION`, `ROADMAP`, `ISSUES`, `INTENTS`, `LIFEBOAT`, `PII`,
`OPINIONS`, `LOAD`. Each carries recall keywords and its rules, bundled in the
abcd binary; a repo overrides them per-field via `.abcd/rules.json`. `OPINIONS`
points at the canonical conventions under `.abcd/development/principles/` rather
than copying them. `LOAD` carries the trust rule for load experiments: one owned
process group killed together, clean proven by what is running, and explicit
consent with a cap below the core count on a live development machine.

### Reset triggers

`SessionStart` and `PreCompact` clear the per-session dedup ledger, so a matched
domain re-injects on the next prompt (the event-driven refresh that recovers
after compaction). Within a session the hook does not re-inject unchanged rules.

For internals see `.abcd/development/brief/05-internals/03-configuration.md`.
