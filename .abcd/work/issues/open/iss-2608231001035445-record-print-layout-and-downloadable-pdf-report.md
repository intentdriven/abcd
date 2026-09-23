---
schema_version: 1
id: "iss-2608231001035445"
slug: "record-print-layout-and-downloadable-pdf-report"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "user-observation"
found_at: ".abcd/site.json"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: print/PDF record surface intent seed; build-time vs browser print open). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

A dedicated print layout for the record, and a downloadable PDF report of the whole Record section (future intent seed). The 2026-08-23 print pass is a stylesheet that stops the worst faults — blank sheets, sliced cards, printed chrome — and it stops there: a closed disclosure still prints as its summary alone outside Chromium, the relationship chart's list is deliberately left collapsed, and nothing composes a document a reader would want to keep. This intent is the designed thing: a print/PDF surface for /record/** with its own page furniture (running heads, page numbers, a contents page, record ids as cross-references rather than links), and a single 'download the record' artefact a visitor can take away. Generic-side under itd-140 — inputs stay the record format, git history and CHANGELOG.md — and adr-47's single-source rule applies unchanged: the document selects spans, it never writes prose. Open questions for the interview: is the PDF built at site-build time (deterministic, attestable, adds a renderer dependency) or printed by the reader's browser from a print stylesheet (no dependency, no attestation, engine-dependent); and does it cover the record only or the whole site.