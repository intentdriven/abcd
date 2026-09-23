---
schema_version: 1
id: "iss-193"
slug: "attribution-identity-enforcement-belongs-in-abcd-for-every-m"
severity: "major"
category: "process"
source: "user-observation"
found_during: "manual-capture"
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M2: fold into itd-131 / spc-34, widened to pin the identity at setup and set user.useConfigOnly; the amendment of spc-34 is owed and must reconcile itd-131's disk-only rule (adr-38) with resolving the canonical GitHub identity at install, and iss-85 / iss-2608210738367948 with it)."
---

A managed repo commits under whatever identity git happens to resolve, and when
`user.email` is unset git **invents one** — the OS account's full name plus
`$USER@$(hostname)`, silently, on the first commit. This repo's own history
carried 25 such commits before the 2026-08-06 rewrite, all authored and
committed under a fabricated `<user>@<host>.local` pair no one ever typed. The
same gap let autonomous harness runs commit under a tool identity: nothing in a
managed repo declares who its commits belong to, so every machine and every
harness answers the question differently.

**Scope, settled by the maintainer (2026-08-06):** abcd enforces exactly one
thing — that a managed repo commits under the user's own GitHub identity, and
that git can never fabricate a substitute. Nothing more. Rejecting AI
co-author trailers, allowlisting authors, and gating pushes on trailer
presence are **out of scope here**: those belong to the forge's own rules
(a server-side ruleset) and to iss-119, not to abcd.

What that means concretely:

- **`ahoy install` pins the identity.** It resolves the user's canonical GitHub
  name and email (the account's no-reply address is the safe default — it is
  what the forge already attributes to them, and it leaks no personal mailbox)
  and writes both into the managed repo's local git config. Repo-local, not
  global: abcd configures the repos it manages and leaves the rest of the
  machine alone.
- **Fabrication is disabled, not merely overridden.** The repo also sets
  `user.useConfigOnly`, so a checkout whose identity is missing or cleared
  fails the commit with an error instead of quietly deriving one from the host.
  A refusal is recoverable; a fabricated commit is already in the history.
- **The state is observable.** A managed repo whose identity is unset, or set
  to something the install did not pin, is reported — the same fail-loud
  posture the guard and audit rules take, so the gap surfaces before a commit
  rather than after a rewrite.

The privacy half matters as much as the attribution half: an auto-derived
identity embeds the machine's hostname in every commit it touches
(`alice@alice-laptop.local` in the persona shape), which is the same
never-commit-hostnames class the v0.5.0 security half closes in scanning and
redaction. Prevention at the commit boundary is where that leak is cheapest to
stop — the scanner can only find it once it is already written.

Likely intent-scale: a user-facing capability across every managed repo, so it
wants an intent and a spec rather than a bare fix. Reconcile with, rather than
duplicate, iss-84 (managed pre-commit gates — the hook seam this would use),
iss-85 (managed attribution config — the nearest neighbour; check whether this
supersedes it or lands inside it) and iss-119 (`Assisted-by` declared but
unenforced — the trailer half, deliberately left out of this scope).
