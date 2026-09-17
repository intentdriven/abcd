---
schema_version: 1
id: "iss-2608291009106041"
slug: "abcd-cannot-colour-a-selection-the-host-renders-which-bounds"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "role-clarification-run"
found_at: "internal/term"
---

abcd cannot colour a selection the host renders, which bounds where role colouring can work at all. Tested by embedding terminal colour codes in a host-rendered option preview: the host escaped them rather than interpreting them, so the reader saw the raw codes. This is the working stance stated plainly, that the harness owns the pixels and abcd owns the words, arriving as a measured constraint rather than a position. Two consequences follow. Role colouring is available only on surfaces abcd renders itself, which today means its own terminal output and later the product thinker's own surface, and on a host-rendered prompt the addressee marker has to carry the distinction through its words and its glyph alone. And a design that assumes colour is available everywhere would degrade silently on exactly the surface a product thinker is most likely to meet first.