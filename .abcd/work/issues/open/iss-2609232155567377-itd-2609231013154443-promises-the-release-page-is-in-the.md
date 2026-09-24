---
schema_version: 1
id: "iss-2609232155567377"
slug: "itd-2609231013154443-promises-the-release-page-is-in-the"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/release/page.go"
---

itd-2609231013154443 promises the release page is in the words each intent's own press release already uses, quotes included, and its scope names the persona quotes of the intents it tells, carried word for word; but the release-page ingest verifies only the quotes a payload carries. A headline may tell an intent whose press release carries a persona quote and carry none, and the cut writes the page. Found by the fidelity audit of ac-7 (receipt rcp-af55e181c483): agents/release-changelog-composer.md asks the composer for the quote of each told intent, and spc-2609231435545473 files the gap under Risks as quotes being optional to the binary because no criterion asks for them. Either the page bijection requires one quote per told intent whose source carries one, or the intent's press release stops promising quotes included.
