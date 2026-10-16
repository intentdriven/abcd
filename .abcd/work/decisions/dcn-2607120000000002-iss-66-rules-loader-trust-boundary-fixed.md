---
id: dcn-2607120000000002
slug: iss-66-rules-loader-trust-boundary-fixed
date: 2026-07-12
---
2026-07-12 — iss-66 rules-loader trust boundary. Fixed the two mechanical items:
the Load Lstat→ReadFile TOCTOU on `.abcd/rules.json` (now open-once O_NOFOLLOW +
fstat, C19) and the session-state dir moved off the world-writable shared /tmp to
the per-user cache dir (P14). **P15 document-accepted, NOT changed:** a per-repo
`.abcd/rules.json` can set a default domain dormant and flip the global kill
switch (Merge is intentionally per-field + sticky-kill-switch). Rationale: rules
are an *opt-in, opinionated-but-overridable* config layer; `.abcd/rules.json` is a
committed file (editing it needs repo write access, like any committed guardrail),
and the real enforcement of dangerous actions is harness-level (git-guardrails
hooks, the iss-62 identity gate, pre-commit), not the injected advisory prose.
Silencing a domain removes prose, not a hard gate. **Deferred design alternative
(surfaced, not taken):** introduce a protected "guardrail" domain class that a
per-repo override cannot set dormant and that the kill switch cannot silence —
this adds a new protected-domain concept to the rules contract, a maintainer
decision, not an autonomous change.
