---
schema_version: 1
id: "iss-2609012037425074"
slug: "ghsa-4fmm-95pf-32c6-cwe-150-on-an-empty-store-ask-returns-re"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/ask.go"
resolution: "Ask sanitises the question once through termsafe.Sanitize and that one value feeds RenderNoMatches, the synthesizer and AskResult.Question on both return paths; RenderNoMatches sanitises on its own as well, mirroring RenderCitedMatches, so a direct caller is covered. Retrieval still tokenises the raw question so ranking is unchanged. Proven by TestAskEmptyStoreSanitisesQuestion (ESC, C1 CSI, U+202E and U+200B planted in the question; none survive in the answer or the JSON field, the legitimate words do) and TestRenderNoMatchesSanitisesDirectly. The sibling error surface (cli.go scrubPaths, no termsafe) is repo-wide and captured as iss-2609012037438844, not fixed here."
impact: fix
---

GHSA-4fmm-95pf-32c6 (CWE-150): on an empty store, Ask returns RenderNoMatches(req.Question) with the question raw, while RenderCitedMatches sanitises it, and both return paths put req.Question raw into AskResult.Question (the --json field), so ESC, C1, bidi-override and zero-width runes in the argv reach stdout (reproduced at v0.7.0: two raw ESC bytes and a raw U+202E on stdout). The fix must sanitise the question once in Ask through termsafe.Sanitize and feed that one value to both renders and the JSON field, pinned by a test that plants ESC and U+202E and proves neither survives in the answer or the question field.

## Grounds

- pursued: one sanitiser at the point the question is first echoed, so the two renders and the JSON field cannot disagree again; a second fix to one render would have re-opened the drift this closes
