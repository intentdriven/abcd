---
schema_version: 1
id: "iss-2609100508565741"
slug: "required-flags-are-learned-from-the-refusal-not-the-help"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli (help output)"
---

A verb's required arguments are learned from its first refusal, not from its help, because the bare help carries no worked example.

Observed resolving issues during an autonomous run. `abcd capture resolve` requires `--impact` and `--grounds`; the operator discovered both by being refused, once each, and only then assembled a working invocation. The flags are listed, so the help is not wrong — it is that a list of flags does not say which combination constitutes a legal call, and the cheapest way to find out is to run the verb and read the error. For an agent this is not merely inelegant: each refusal costs a round trip, and the text being filed has to be re-sent with it.

The same shape recurs across the record verbs, whose calls carry several interdependent flags and whose refusals are the only place the dependency is stated.

Wanted: one worked example per verb in its own `--help` output — the shortest legal invocation, with the required flags filled in. It is a line of text per verb and it removes the refusal-as-documentation loop entirely.

Distinct from the sibling finding that abcd does not name its own adjacent capabilities: that one is about a verb the operator never learns exists, this one is about a verb they have found and cannot call. The remedies differ — a worked example in the verb's own help, versus a cross-pointer between verbs — so they are filed apart.
