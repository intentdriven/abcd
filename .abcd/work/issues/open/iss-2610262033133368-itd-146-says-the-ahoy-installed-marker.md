---
schema_version: 1
id: "iss-2610262033133368"
slug: "itd-146-says-the-ahoy-installed-marker"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 375 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:1071"
remedy: "none (filed automatically)"
related_intents: [itd-146]
finding_ref: cfk-54ba939931a6307c
---

itd-146 says the ahoy-installed marker block tells every managed repository to run `abcd rules`; the ahoy chapter says a default install writes that block into no conventions file

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:1071` — "install writes it into none of the repository's committed conventions files"
- End B: `.abcd/development/intents/shipped/itd-146-abcd-s-help-renders-in-labelled-command.md:77` — "`--help` contradict the `ahoy`-installed marker block, which tells every"

itd-146 keeps `rules` visible in help because hiding it would contradict the ahoy-installed marker block, which it says tells every managed repository to run `abcd rules`. The ahoy chapter says the docs target defaults to `skip`, so a default install writes the block into none of the repository's committed conventions files, and only a project that names `agents_md` gets it. The block does not reach every managed repository, so the reason itd-146 gives rests on a premise the chapter denies.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 375.
