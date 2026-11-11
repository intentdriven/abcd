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
resolution: "docs/how-to/install.md now names the bootstrap attempt made after a question is answered beside the ones made before a command and before a compaction, under the host's ten-minute limit. The detector (detector.py in the lane directory) was watched failing then passing but is not committed: widening the hold to a test file exited 2 because the reservation push could not read GitHub credentials inside the sandbox, and lifting the sandbox is a person's call."
impact: fix
resolved_by:
  commit: "8421e794d63fdfe98b67e2865160c9eaa81ea768"
---

docs/how-to/install.md:82-85 gives the limits of the bootstrap attempts made when a message is sent and before a command or a compaction, and leaves out the attempt made after a question is answered: the PostToolUse hook on the question tool (hooks/hooks.json:39-50, hooks/post-tool-use.sh) also calls bootstrap.sh when the plugin root is empty, under the host's ten-minute default, since it declares no timeout.

## Grounds

- pursued: the install guide's ten-minute-limit sentence names every hook that hooks/hooks.json runs with no declared timeout and whose script calls bootstrap.sh; a hook in the manifest that attempts the bootstrap outside session start and whose moment or limit the paragraph does not state would show it wrong.
