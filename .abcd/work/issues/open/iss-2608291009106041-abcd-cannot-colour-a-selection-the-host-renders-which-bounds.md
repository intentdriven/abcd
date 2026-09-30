---
schema_version: 1
id: "iss-2608291009106041"
slug: "abcd-cannot-colour-a-selection-the-host-renders-which-bounds"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "role-clarification-run"
found_at: "internal/term"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed E): Record 'the harness owns the pixels, abcd owns the words' as a design constraint (a principle or a brief line)? The stance itself is now stated by adr-2609151528057131 (accepted), which holds it for every surface but the product thinker's; what stays unrecorded is the measured bound this record adds, that a host escapes colour codes in the options it renders, so role colouring reaches only output abcd renders itself"
remedy: "Waits on the pixels-and-words ruling: if recorded: add one bounds line to the brief's terminal-capability invariant (adr-49) that a host escapes colour codes in options it renders, so role colouring applies only to output abcd renders itself and a host-rendered prompt marks the addressee in words and glyph alone, pinned by a test that the addressee marker text carries no escape sequence; if not: wontfix the record citing adr-2609151528057131 as the stance already recorded."
---

abcd cannot colour a selection the host renders, which bounds where role colouring can work at all. Tested by embedding terminal colour codes in a host-rendered option preview: the host escaped them rather than interpreting them, so the reader saw the raw codes. This is the working stance stated plainly, that the harness owns the pixels and abcd owns the words, arriving as a measured constraint rather than a position. Two consequences follow. Role colouring is available only on surfaces abcd renders itself, which today means its own terminal output and later the product thinker's own surface, and on a host-rendered prompt the addressee marker has to carry the distinction through its words and its glyph alone. And a design that assumes colour is available everywhere would degrade silently on exactly the surface a product thinker is most likely to meet first.

## Remedy grounds (2026-09-29)

- The measured bound (a host escaping embedded colour codes) is a fact about hosts, so it belongs beside the colour ladder internal/term resolves rather than in a new principle.
- Rejected: an escape-sequence probe of each host at run time, which would colour output abcd does not own.
