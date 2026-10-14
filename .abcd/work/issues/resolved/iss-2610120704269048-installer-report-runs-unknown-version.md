---
schema_version: 1
id: "iss-2610120704269048"
slug: "installer-report-runs-unknown-version"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "security installer lane, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "site-src/install.sh.tmpl"
remedy: "Call --version in report(), and add a site test that runs the rendered installer against a stub binary that only knows --version, watched failing first"
resolution: "the served installer's report() runs the installed binary with --version, the one way abcd prints its version, so a successful install ends in the version report rather than an unknown-command error; the installer sandbox's stub binary now knows only --version, as abcd does"
impact: fix
---

The install script the site serves (site-src/install.sh.tmpl) ends with report(), which runs "$bin_dir/abcd" version, a command abcd does not have (it exits 2; the version is printed by --version), so an install that succeeded ends in an error. Found by the installer lane on 2026-10-12.
