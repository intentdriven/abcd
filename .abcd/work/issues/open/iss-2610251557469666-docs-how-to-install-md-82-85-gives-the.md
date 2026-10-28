---
schema_version: 1
id: "iss-2610251557469666"
slug: "docs-how-to-install-md-82-85-gives-the"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D1"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/how-to/install.md"
remedy: "Name the attempt made after a question is answered beside the one made before a command, at docs/how-to/install.md:84-85."
---

docs/how-to/install.md:82-85 gives the limits of the bootstrap attempts made when a message is sent and before a command or a compaction, and leaves out the attempt made after a question is answered: the PostToolUse hook on the question tool (hooks/hooks.json:39-50, hooks/post-tool-use.sh) also calls bootstrap.sh when the plugin root is empty, under the host's ten-minute default, since it declares no timeout.
