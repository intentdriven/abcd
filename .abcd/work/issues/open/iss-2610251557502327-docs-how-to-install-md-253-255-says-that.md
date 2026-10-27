---
schema_version: 1
id: "iss-2610251557502327"
slug: "docs-how-to-install-md-253-255-says-that"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D1"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/how-to/install.md"
remedy: "Say the shell guard refuses every shell command while ~/.abcd stands and admits the exact rename when the plugin root holds a binary, and keep the Terminal advice for the empty plugin root a plugin update leaves."
---

docs/how-to/install.md:253-255 says that while ~/.abcd stands no tool runs at all in an agent session, because the safety check blocks every command, the rename included. That holds only while the plugin root is empty, the state a plugin update leaves, since hooks/pre-tool-use.sh exits 2 before any bootstrap. With a binary in the plugin root, abcd guard hook admits the exact rename command (internal/surface/cli/homestop.go:102-109, :143-160; internal/abcdhome/abcdhome.go:45-48), and the hook gates only the shell and question tools (hooks/hooks.json:29), so other tools still run.
