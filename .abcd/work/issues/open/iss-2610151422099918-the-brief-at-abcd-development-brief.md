---
schema_version: 1
id: "iss-2610151422099918"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/README.md:148 to match the shipped surface: `bin/abcd-darwin-arm64 --help` lists two verbs under Release: `embargo` and `launch`. commands/embargo.md declares `block: people`. The table's placement of every visible verb leaves embargo out."
---

The brief at .abcd/development/brief/04-surfaces/README.md:148, .abcd/development/brief/04-surfaces/README.md says: The help-placement table gives the people block's `release` group as `launch` alone, and `embargo` appears in no row of the table (neither a people group nor the agents-and-hosts block). The shipped surface differs: `bin/abcd-darwin-arm64 --help` lists two verbs under Release: `embargo` and `launch`. commands/embargo.md declares `block: people`. The table's placement of every visible verb leaves embargo out. (v0.14.2 release crosscheck, stale-count, raised by opus+sonnet in a31, b00.)
