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
remedy: "Give the walled sessions one fixed home at ~/.abcd.noindex/profiles/ (the product thinker's choice, 2026-11-03), named for what it is: a dedicated configuration directory is a separate profile on the same OS account, not a sandbox or an isolated environment, since a stage's shell can still read whatever the account can. It holds one directory per agent subscription, so one person can keep several subscriptions of several tools (claude, codex, grok) side by side, each with its own login, settings and stage agents limited to file and shell tools. abcd gives each one a random name when it creates it, so a name carries no provider or account, and keeps which tool and subscription each name holds in a registry the person can read. An ahoy command creates one, walks the person through its one login and never reads a token; ahoy reports each one's readiness; drain --drive and any walled start choose among them by role and by the 95% window of dcn-2611031615094092. Each tool's wall is proved on its own before that tool runs a stage (connectors, memory, settings, hooks, billing), and a tool that cannot carry abcd's guard does not run one. Real isolation follows as its own step: a separate OS user, a container or the host's own OS sandbox per directory, since a shell in one can otherwise read another's login, the person's own configuration and their credentials. A test asserts each."
---

abcd has no fixed place or command for the walled configuration directory: spc-2610230813348813 calls for a dedicated CLAUDE_CONFIG_DIR the person logs into once but names no location, and on 2026-11-03 lab lab-261103171720-c0bdccf chose ~/.abcd.noindex/claude-walled/ and a hand-written launch script pointed at the 0.16.0 plugin path; no abcd code knows that directory, so a session or verb started in a checkout cannot find, create or report the wall, and each account repeats the setup by hand. The same setup showed that the plugin ships no implementer agent limited to file and shell tools, and that a host reads a new agent file only at session start.
