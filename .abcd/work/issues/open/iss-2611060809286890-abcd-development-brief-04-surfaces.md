---
schema_version: 1
id: "iss-2611060809286890"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/21-update.md"
remedy: "Correct the passage to say what the code does."
---

`.abcd/development/brief/04-surfaces/21-update.md:26`: The update chapter names the launch preview and the cut as the other commands reaching the release origin, and session set-up and install reach it too. Other verbs reach the release origin. `abcd session blueprint` and `abcd session setup` for the cloud fetch checksums.txt from the release page (internal/core/session/blueprint.go:82,495; internal/surface/cli/session.go:76 update.NewReleaseAssets(sessionReleaseOrigin); `session blueprint --help`: 'The checksum is read from the release page when the files are written (the one network read this verb makes'; `session setup --help`: 'for the cloud it reads the release's checksums'). `abcd ahoy install` downloads the release binary when abcd is not on PATH (internal/core/ahoy/abcd_download.go; internal/surface/cli/update.go:42 update.NewGitHubUpdater().FetchVerified).
