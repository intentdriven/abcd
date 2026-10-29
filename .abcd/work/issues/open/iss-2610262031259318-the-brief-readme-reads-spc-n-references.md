---
schema_version: 1
id: "iss-2610262031259318"
slug: "the-brief-readme-reads-spc-n-references"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 128 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:163"
remedy: "Rewrite the delivery-state provenance note in brief/README.md (lines 8-15): an unqualified `spc-N` is a native-store spec, a predecessor id is marked '(predecessor store)' per the two-namespace rule in specs/README.md; also replace 'roadmap dashboard' there with the status block."
finding_ref: cfk-f0d881f70a3b6189
---

The brief README reads spc-N references throughout the brief as predecessor-store ids; 04-naming says an unqualified spc-N resolves in the live store and a predecessor id must be marked as such.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:163` — "an unqualified `spc-2` or `spc-3` resolves in the live store to a spec about something else entirely"
- End B: `.abcd/development/brief/README.md:8` — "the predecessor implementation's spec store, carried as design input"

The brief README's provenance note says spc-N references throughout the brief trace 'the predecessor implementation's spec store, carried as design input', never delivery claims. 04-naming's two-namespace rule marks a predecessor id '(predecessor store)' and warns that the id spaces overlap, so 'an unqualified `spc-2` or `spc-3` resolves in the live store to a spec about something else entirely'; 04-naming itself cites live specs such as spc-2610030911534855 unqualified. A reader applying the README's rule resolves every unqualified id in the wrong store.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 128.
