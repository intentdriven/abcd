---
schema_version: 1
id: "iss-2611281043117150"
slug: "stale-note-for-a-defined-flag"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/staleusage.go"
remedy: "Give the stale-usage note only for a flag the refusing command does not define, its own flags and the inherited ones, so a malformed argument naming a real flag gets the plain refusal; test capture list '--open x' on a build outside the plugin root, watched fail first."
resolution: "The stale-usage note is now given only for a flag the refusing command does not define, its own or inherited, so an argument holding a defined flag and a value, such as --open x, gets the plain unknown-flag refusal; the retired-flag and help --agent answers read the same line for flags no command defines and are unchanged."
impact: fix
resolved_by:
  commit: "65739ce94"
---

The stale-usage note claims a build lacks a flag it defines when the flag reaches abcd joined to a value by a space in one argument: capture list given the single argument '--open x' is refused as an unknown flag, and the note then says the plugin surface documents the --open flag of abcd capture list, which is not in this build. The unknown-flag line is cut at the first space, and the bare name is looked up on the pages without asking whether the refusing command defines it. reserve given '--footprint a --footprint b' as one argument, from a shell that did not split words, met the same claim.
