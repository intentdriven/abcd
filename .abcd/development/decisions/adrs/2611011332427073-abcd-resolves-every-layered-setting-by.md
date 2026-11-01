---
id: adr-2611011332427073
slug: abcd-resolves-every-layered-setting-by
status: proposed
date: 2026-11-01
supersedes: null
superseded_by: null
related_intents: [itd-2611011333366788]
related_rfcs: []
related_adrs: []
---

# ADR-2611011332427073: abcd resolves every layered setting by one precedence rule

## Context

abcd reads its settings from several layers: the defaults bundled in the binary, the person's home (`~/.abcd.noindex/`), the repository's `.abcd/`, the environment and a command-line flag. It has no single rule for which layer wins. An inventory taken on 2026-11-01 found these differences:

- **The shared resolver** (`internal/core/layered`) orders flag over repository over machine over bundled.
- **The rules loader** orders the repository over the user too, but it reads on its own.
- **Restriction-style settings** go the other way, in at least five shapes:
  - `land.push`/`land.enqueue` are ANDed across layers.
  - `facilitator.mode` is won by `human` from any layer.
  - `question.*` limits can only be lowered by the machine.
  - Fan-out ceilings clamp.
  - Deny lists and the rules kill switch are unions.
- **Several keys are machine-only**: paid-key oracle routes, `load.preflights`, `reservation.*`, the status line and the trust lists.
- **Environment variables are read outside the resolver**: `ABCD_LOAD_PREFLIGHTS`, `ABCD_ACCESSIBLE`, `ABCD_HOLDER_ID`.
- **Some readers bypass the resolver entirely**: `guard.json`, `outbound.forge` and the status line.

There is also no layer through which a person can override a repository's choice for themselves alone, without editing a committed file.

The product thinker raised this during the setup's model-tier routing question on 2026-11-01. They found the present order confusing, asked for a state-of-the-art check, and asked for one rule for all of abcd. That check read the primary documentation of git config, Cargo, pip, mise, VS Code, Claude Code settings and permissions, npm, EditorConfig, ESLint, direnv, Terraform's CLI config, kubeconfig, systemd drop-ins, the XDG base-directory specification, and Chromium's and Firefox's policy designs.

## Decision

We will resolve every layered setting by one rule, lowest precedence first:

1. abcd's bundled default;
2. the person's home file (`~/.abcd.noindex/`);
3. the repository's committed file (`.abcd/`);
4. the person's gitignored file in that repository (in the local tier);
5. the invocation: a flag, or an environment variable that names the same key.

The closer and more specific layer wins for an ordinary preference. Three classes of setting are exempt from that order, and each class is declared per key, never per layer:

- **Restrictions only tighten.** A cap, a deny list, a kill switch or an off switch from any layer holds, and no layer can loosen what another tightened. List-valued restrictions are merged, never replaced.
- **A repository never grants capability or trust.** A key that widens what abcd may do, spend or reach is read from the person's layers alone, or from a repository the person has trusted from their home.
- **Some keys belong to the person or the machine.** Identity, credentials, paid routes, local resource budgets and the person's own display are read from home and the invocation only. A repository value for such a key is ignored, with a diagnostic naming it.

The model-tier routing table follows the ordinary order. The repository's table sets the project's tiers, and the person's gitignored per-repository file overrides them for that person alone. A fan-out ceiling stays a restriction.

## Alternatives Considered

- **Personal beats repository beats default, for every key.** It is the simplest rule to state, and it was the product thinker's first framing. It was rejected because no repository choice could be relied on. Settings that define the project, such as its gates, lint and review tiers, would differ per person, which is why ESLint removed personal configuration in v8.0.0.
- **Repository beats personal everywhere, as the resolver does today.** It is reproducible, but a person has no way to run a repository's agents cheaper or on a local model without editing a committed file. It also leaves the restriction exceptions implicit, which is the confusion this ADR exists to end.
- **The chosen rule.** git (system < global < local < worktree < `-c`), Cargo, pip, mise, VS Code and Claude Code share its order. Claude Code (a deny from any scope beats an allow), git (`safe.directory` honoured only in protected configuration), VS Code (machine- and application-scoped keys are user-only) and Terraform (credentials are per-user only) show its three exemptions.
- **A managed policy tier above the invocation, as browsers and VS Code have.** It was not adopted. It pays off only at enterprise scale, and it can be added above layer 5 later without changing this order.

## Consequences

- One sentence now answers "which file wins": the closer one, unless the key is a restriction, a capability or the person's own.
- A new gitignored per-repository layer has to be added to the shared resolver.
- Every layered reader has to be moved onto the resolver and onto this order, so that each exemption is declared on its key rather than coded in its reader. Those readers include the rules loader, `guard.json`, `outbound.forge`, the status line, and the environment variables now read outside it.
- The configuration chapter of the brief has to state the rule once, with each key's class. Every other page names that chapter instead of restating it.
- `question.*` currently lets the machine only lower the repository's limit. It needs reviewing against the restriction class, as do the `max_reviewers` clamp and `oracle.judgements`' limit on which values a repository may name.
- A repository that relies on the present order, where the repository beats the home file, keeps it for every ordinary key. Only a person's new local file can override it.
