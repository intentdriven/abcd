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
