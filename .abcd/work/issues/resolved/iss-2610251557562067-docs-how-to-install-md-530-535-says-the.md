---
schema_version: 1
id: "iss-2610251557562067"
slug: "docs-how-to-install-md-530-535-says-the"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D1"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/how-to/install.md"
remedy: "Name the script's --bin-dir option and its PATH line beside the release-mark at docs/how-to/install.md:530-535."
resolution: "the install guide now documents the released installer itself rather than a separate one-liner, so the comparison is gone: it names --bin-dir beside the command and says the installer prints the PATH line when the install directory is not on PATH"
impact: fix
---

docs/how-to/install.md:530-535 says the script served at /install.sh does what the one-liners do and one thing more, recording the release-mark. The script also takes --bin-dir DIR (site-src/install.sh.tmpl:53-55, :311-332) and prints the PATH line when the install directory is not on PATH (:300-308), neither of which the one-liners do (docs/how-to/install.md:484-485).
