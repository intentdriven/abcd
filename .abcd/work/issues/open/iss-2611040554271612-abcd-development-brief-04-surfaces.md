---
schema_version: 1
id: "iss-2611040554271612"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/30-inbox.md"
remedy: "Name the read failure among the exit-1 cases, or map it to the refusal exit."
---

.abcd/development/brief/04-surfaces/30-inbox.md:148-150 defines exit 1 only as a failure after the capture is written, naming the capture. With an inbox directory the account cannot read, `abcd inbox` and `abcd inbox show <id>` exit 1 with a cannot-read-the-inbox message, also in the JSON envelope, and no capture exists or is named.
