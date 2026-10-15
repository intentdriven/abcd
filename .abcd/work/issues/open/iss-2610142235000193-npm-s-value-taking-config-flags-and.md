---
schema_version: 1
id: "iss-2610142235000193"
slug: "npm-s-value-taking-config-flags-and"
severity: "nitpick"
category: "security"
source: "agent-finding"
found_during: "security review of feat/guard-port-blocks for v0.15.0"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
remedy: "Add npm's value-taking config flags (--registry, --cache, --userconfig, --prefix and the other documented ones) to the npm exec value-flag list, and step npm's global options before the subcommand in runnerAt; prove it with each spelling in front of kill-port at block."
---

npm's value-taking config flags and global options are not in the runner value-flag lists, so their value is read as the program: npm exec --registry <url> kill-port 3000, --cache <dir>, --userconfig <file>, and a global option before the subcommand (npm --prefix . exec kill-port 3000) read the value or the option as the program and only warn as an unrecognised launcher where npm exec kill-port 3000 blocks.
