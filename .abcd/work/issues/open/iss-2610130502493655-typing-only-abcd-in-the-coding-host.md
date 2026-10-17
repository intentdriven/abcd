---
schema_version: 1
id: "iss-2610130502493655"
slug: "typing-only-abcd-in-the-coding-host"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "a managed repository on v0.13.4, 2026-10-13"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/abcd.md"
remedy: "Find what orders the host's slash-command menu for a typed prefix (recent use, name, or the order pages are listed); if abcd can influence it, make the status page the first hit for /abcd:; if only recent use decides it, say in commands/abcd.md and the install guide that the status board is /abcd:abcd (or bare /abcd) and that Return on the first menu hit runs whatever was used last"
---

Typing only '/abcd:' in the coding host lists /abcd:lab first, not the status page /abcd:abcd, so pressing Return runs the lab listing instead of the general status board. Seen in a managed repository on 2026-10-13, where /abcd:lab had just been run, which suggests the host orders the menu by recent use (the next hit shown was /abcd:ahoy). The person expected the bare prefix to land on the status board.
