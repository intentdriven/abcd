---
schema_version: 1
id: "iss-2610240827243654"
slug: "the-documented-cli-install-one-liner"
severity: "major"
category: "ux"
source: "user-observation"
found_during: "/abcd:capture"
origin: researcher-authored
production_mode: hand-written
found_at: "README.md"
remedy: "Make the documented one-liner a short fetch of the install.sh abcdev.app already serves (which does the same checksum and gh attestation steps), move the remaining one-liner-only behaviour (the path-entry record) into that script, and keep the long form only as the by-hand audit path in docs/how-to/install.md."
resolution: "README and the install guide document the short command that fetches install.sh from the latest release, hardened to https only"
impact: fix
---

The documented CLI install one-liner (README and docs/how-to/install.md) is far too complicated to be a one-liner: a single sh -c string of roughly 3,000 characters that unsets proxy variables, refuses a symlinked ~/.abcd.noindex, maps uname output to a release asset, downloads it and checksums.txt, verifies the checksum, probes gh attestation verify for four flags, resolves the latest tag and its commit through the GitHub API, verifies build provenance, prints a long fallback warning, installs to ~/.local/bin and writes ~/.abcd.noindex/path-entry. Nobody can read, audit or safely retype it, and a reader cannot tell what they are agreeing to run.
