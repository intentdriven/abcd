---
id: spc-2609211955339422
slug: setup-wizard-explains-installs
intent: itd-63
origin: researcher-authored
production_mode: hand-written
---
# setup-wizard-explains-installs

## Summary

The design record for itd-63, from the product thinker's interview of
2026-09-21: the explain-then-install mode every verb calls when it finds a
tool missing, with a curated registry of the tools abcd knows.

## Scope

1. **The registry**: `internal/core/tools/registry.go`, one entry per tool
   (name, what it is, what abcd uses it for, optional or required per
   capability, the native default, the install step per platform, the
   verify command); shipped in the binary, extended by a capture when a gap
   is met (criteria 1, 3).
2. **The mode**: `tools.Explain(name, capability)` renders the explanation;
   `tools.Install(name)` runs the step after the caller's confirmation and
   returns the verify result; the CLI asks on the terminal, the plugin page
   through the host's question tool (criteria 1, 2, 5).
3. **Callers**: the scanner adapter's missing-gitleaks path (adr-22), the
   guard's and the launch's tool checks, `ahoy`'s gaps; each replaces its bare
   command with the mode and continues on its native default on a no
   (criteria 2, 4).
4. **Loud staging**: a no is printed as "continuing on <native default>",
   never silent (criterion 2).

## Out of scope

- A standalone setup verb; the mode is called, not run alone.
- Generated descriptions; a gap is captured, not composed.

## Approach

One package under core with no transport knowledge; the callers pass a
confirm function the surface supplies. Existing tool checks are found by
grepping for the install hints they print today and rerouted one by one,
each with a test that the explanation appears and the default holds on a no.

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 explanation from the registry | scope 1, 2 |
| 2 install on yes, default on no, reported | scope 2, 4 |
| 3 unknown tool: generic text, gap captured | scope 1 |
| 4 the safety gate routes through it | scope 3 |
| 5 no host dependency | scope 2 |

## Close note

Delivered on 2026-09-26 against the tree as it stands, which differs from the
record this spec was written from in two places:

- **The safety gate's missing-scanner path** (criterion 4) belongs to itd-62,
  which is still a draft: no gate on the default branch always blocks on a
  missing scanner. The one fail-closed missing-scanner path there is the
  history store's, for a repository that armed the gitleaks adapter
  (`internal/adapter/gitleaks`, opt-in since the 2026-07-24 ruling that made
  the native scanner the default). Its refusal now carries the registry's
  explanation (`tools.Missing`), and `ahoy install` offers the install, as a
  required tool, for exactly that repository.
- **The guard's and the launch's tool checks** (scope 3) do not exist: the
  guard runs no external tool, and the launch scans are native (itd-65). There
  was nothing to reroute, and nothing was invented.

The other callers are rerouted: `ahoy`'s dependency gap and install step, and
the missing-`gh` refusal of `ahoy remote` and `site setup`. The trufflehog gap
was removed rather than routed (iss-2609261447331434): nothing runs trufflehog.
The unknown-tool gap (criterion 3) is captured, not composed: the explanation
names it as abcd's own and carries the `abcd capture` line that records it,
and a test fails on any tool ahoy names that the registry lacks.
