---
schema_version: 1
id: "iss-2608280825450623"
slug: "no-documented-protocol-tells-an-agent-to-run-abcd-docs-lint"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "style-guide-review"
found_at: ".abcd/rules.json"
resolution: "This repository's DOCUMENTATION domain override in .abcd/rules.json gains the pre-present protocol line: drafted docs artefacts run abcd docs lint before presentation; conversational replies stay unlinted (the artefact-gated verdict from the 2026-08-28 audience research). The bundled default domain is unchanged."
impact: additive
---

no documented protocol tells an agent to run abcd docs lint over a drafted docs artefact before presenting it: style feedback arrives at commit time instead of draft time