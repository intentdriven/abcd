---
schema_version: 1
id: "iss-2610120617521209"
slug: "bootstrap-and-installer-trust-same-page"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "E5 fix lane, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/bootstrap.sh"
remedy: "Choose how the shell installers verify provenance (gh attestation verify when present, an already-verified abcd, or a refusal naming the manual check), then implement it in both scripts with tests in a temp HOME; the choice is the product thinker's where it changes what a first install needs"
resolution: "per the product thinker's ruling of 2026-10-12, hooks/bootstrap.sh and site-src/install.sh.tmpl verify the download with gh attestation verify (held to intentdriven/abcd's release workflow on github.com) when gh is present and refuse on a failed verification; with gh absent, too old for the attestation command, or not signed in, they install on the checksum alone with a warning naming why and the hand check"
impact: fix
---

The plugin's first-run bootstrap (hooks/bootstrap.sh) and the install one-liner the site serves (site-src/install.sh.tmpl) download an abcd binary and check it only against checksums.txt from the same release page, the weakness iss-2609231050273096 fixed for abcd update in Go. A shell script cannot run the in-process provenance check, so the fix needs a design choice (for example: verify with gh attestation verify when gh is present and refuse or warn without it; or verify with a previously installed, already-verified abcd). Found by the E5 fix lane on 2026-10-12.
