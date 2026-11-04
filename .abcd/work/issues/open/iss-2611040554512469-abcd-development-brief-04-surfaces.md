---
schema_version: 1
id: "iss-2611040554512469"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/22-site.md"
remedy: "State the live-site question, the refusal under --yes and the flag in the prose."
---

.abcd/development/brief/04-surfaces/22-site.md:96-99 describes the remote stages' confirmation with no case for a site already live, and --replace-existing appears only in the flag appendix at .abcd/development/brief/04-surfaces/22-site.md:397. `site setup --help` says that flag publishes over a site the host already serves without asking, and that --yes without it refuses when such a site exists. commands/site.md:284 and docs/reference/cli/commands.md:5746 carry this; the chapter's prose does not.
