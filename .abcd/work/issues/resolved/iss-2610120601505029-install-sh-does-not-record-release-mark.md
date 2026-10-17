---
schema_version: 1
id: "iss-2610120601505029"
slug: "install-sh-does-not-record-release-mark"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security fix lane B2, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "site-src/install.sh.tmpl"
remedy: "Have the install script record the mark after it verifies the release, with the same atomic owner-only write the bootstrap uses; prove with a test in a temp HOME that a fresh install leaves the mark at the installed release"
resolution: "site-src/install.sh.tmpl records the installed release (as the installed binary reports it) in ~/.abcd.noindex/release-mark after the install, raise-only, with the bootstrap's mktemp-and-rename 0600 write, refusing a symlinked ~/.abcd.noindex and saying so when it cannot record"
impact: fix
---

The install one-liner the site serves (/install.sh, built from site-src/install.sh.tmpl) installs a release without recording the newest-release mark that the bootstrap and abcd update now keep (the fix for iss-2609151847117977 on branch fix/sec-pub-b2-261011), so a machine installed only that way has no mark until the bootstrap or update next runs, and a stale cached release is not refused in that window. Found by lane B2 on 2026-10-12.
