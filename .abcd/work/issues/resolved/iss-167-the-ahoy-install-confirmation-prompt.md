---
schema_version: 1
id: "iss-167"
slug: "the-ahoy-install-confirmation-prompt"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "second-repo install session"
found_at: "internal/core/ahoy"
resolution: "ahoy install prompts read stdin when it is not a terminal, so a piped y drives the interactive path"
impact: fix
---

The ahoy install confirmation prompt only accepts answers from a real TTY: piped 'y' and pseudo-TTY attempts all arrive as declines, so a host agent cannot drive the interactive path at all. For an agent-first tool every prompt needs a non-interactive answer channel (flags per category, stdin when not a TTY, or a --prompt-answers input) — TTY-only confirmation makes the agent report failure and hand the step back to the human.