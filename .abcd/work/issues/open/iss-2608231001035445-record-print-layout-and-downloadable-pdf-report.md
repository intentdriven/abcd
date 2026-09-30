---
schema_version: 1
id: "iss-2608231001035445"
slug: "record-print-layout-and-downloadable-pdf-report"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "user-observation"
found_at: ".abcd/site.json"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker: should a print/PDF record surface become an intent, and is it built at site-build time (a renderer dependency) or printed by the reader's browser?"
remedy: "Waits on the print/PDF planning ruling: if printed by the browser: extend the print stylesheet with @page margin boxes (running head, counter(page)) and a beforeprint handler that opens every disclosure, adding no dependency, proven by a site-render test on the emitted stylesheet and script; if built at site time: the intent names the renderer, which needs the product thinker's dependency sign-off before it is fetched, and pins its output with a golden test; either way the document selects spans and writes no prose (adr-47)."
---

A dedicated print layout for the record, and a downloadable PDF report of the whole Record section (future intent seed). The 2026-08-23 print pass is a stylesheet that stops the worst faults — blank sheets, sliced cards, printed chrome — and it stops there: a closed disclosure still prints as its summary alone outside Chromium, the relationship chart's list is deliberately left collapsed, and nothing composes a document a reader would want to keep. This intent is the designed thing: a print/PDF surface for /record/** with its own page furniture (running heads, page numbers, a contents page, record ids as cross-references rather than links), and a single 'download the record' artefact a visitor can take away. Generic-side under itd-140 — inputs stay the record format, git history and CHANGELOG.md — and adr-47's single-source rule applies unchanged: the document selects spans, it never writes prose. Open questions for the interview: is the PDF built at site-build time (deterministic, attestable, adds a renderer dependency) or printed by the reader's browser from a print stylesheet (no dependency, no attestation, engine-dependent); and does it cover the record only or the whole site.

## Remedy grounds (2026-09-29)

- CSS Paged Media gives running heads and page numbers through @page margin boxes and counter(page) (https://developer.mozilla.org/en-US/docs/Web/CSS/@page, checked 2026-09-29; engine support varies, which is the attestation cost the record names).
- Browsers fire beforeprint and afterprint, which is the script hook that closes the residue the stylesheet comment names, a closed disclosure printing as its summary outside Chromium (https://developer.mozilla.org/en-US/docs/Web/CSS/Guides/Media_queries/Printing, checked 2026-09-29).
- Rejected: choosing the renderer here, since a build-time renderer is a new dependency (prefer-sota: none without sign-off).
