---
schema_version: 1
id: "iss-2611231340312545"
slug: "abcd-embargo-close-refuses-an-advisory"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/embargo/forge.go"
remedy: "let abcd embargo close take a public-track advisory: triage drafts its closing text from the reason and the public issue the capture command filed, close shows it with its digest and closes on --approve exactly as for a closing verdict; test that close refuses a vulnerability and accepts public-track"
resolution: "abcd embargo close now takes an advisory triaged public-track: triage drafts its closing text from the reason, and a verdict recorded before that has its text drafted at close time from the stored reason; the same embargo scan, digest approval and read-back apply, and a vulnerability or untriaged advisory is still refused"
impact: additive
---

abcd embargo close refuses an advisory triaged public-track: it sends a closing text only for not-abcds and not-a-vulnerability (isClosingVerdict), and public-track triage drafts no closing text, so a person ends each public-track advisory by hand on GitHub and the store records the close only at the next intake or plan. Nine advisories triaged public-track on 2026-11-21 have no abcd route to closed.
