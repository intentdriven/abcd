---
schema_version: 1
id: "iss-2611032326573729"
slug: "abcd-has-no-fixed-place-or-command-for"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "preparing the subscription test in a walled session, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "spc-2610230813348813"
remedy: "Give the wall one fixed place under abcd's home, named for what it is (the product thinker prefers ~/.abcd.noindex/sandboxed/ or similar to the lab's claude-walled, which names a host), and an ahoy command that creates it (settings with abcd's guard hook, the stage agents limited to file and shell tools, no plugins or memory), walks the person through their one login and never reads the token; ahoy reports whether it is ready, and drain --drive and any walled start read that one place; a test asserts each."
---

abcd has no fixed place or command for the walled configuration directory: spc-2610230813348813 calls for a dedicated CLAUDE_CONFIG_DIR the person logs into once but names no location, and on 2026-11-03 lab lab-261103171720-c0bdccf chose ~/.abcd.noindex/claude-walled/ and a hand-written launch script pointed at the 0.16.0 plugin path; no abcd code knows that directory, so a session or verb started in a checkout cannot find, create or report the wall, and each account repeats the setup by hand. The same setup showed that the plugin ships no implementer agent limited to file and shell tools, and that a host reads a new agent file only at session start.
