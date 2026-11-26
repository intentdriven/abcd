---
schema_version: 1
id: "iss-2611252051227314"
slug: "update-page-omits-the-bootstrap-request"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/update.md"
remedy: "Correct the page and the invariant with whichever way iss-2611251828533623 goes: name the bootstrap's request in both, or remove the request and leave them."
---

The update page says abcd never fetches implicitly and lists what reaches the release origin without the plugin bootstrap, which asks for the latest release when a plugin root has no binary; the brief's invariant 7 says no version-discovery request exists anywhere in abcd. Run on 2026-11-25 with a stub in place of the downloader, the bootstrap asked for the latest release. Whether the bootstrap should ask is iss-2611251828533623; this is the two statements.
