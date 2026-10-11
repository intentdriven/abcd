---
schema_version: 1
id: "iss-2608231024027294"
slug: "site-disclosure-marker-in-accessible"
severity: "nitpick"
category: "bug"
source: "agent-observation"
found_during: "agent-verification"
found_at: "site-src/site.css"
resolution: "Fixed in the 2026-08-23 manual-test triage pass; verified by rebuild, the seven site-check gates, the overflow audit, and a screenshot of the affected route compared against the report that raised it."
impact: fix
---

The disclosure marker glyph is part of the accessible name on every folded panel: .panel.fold summary h3::before renders a triangle that screen readers announce as text, so the panel is read out as 'triangle References and sources 28'. aria-hidden cannot apply to a pseudo-element; the remedies are a background-image marker or content:\"triangle\" / \"\" alt-text syntax. Pre-existing pattern elsewhere in the stylesheet, newly multiplied by the folded panels.