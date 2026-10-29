---
schema_version: 1
id: "iss-2610262033117994"
slug: "itd-117-puts-the-user-scope-rules-file"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 371 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:490"
remedy: "none (filed automatically)"
related_intents: [itd-117]
finding_ref: cfk-57dae97a5aedc3e2
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both were true when written; the home rename is recorded by itd-2610030720038073 and adr-2610030720195401, so a shipped intent's old spelling is history, not a live contradiction."
---

itd-117 puts the user-scope rules file at `~/.abcd/rules.json`; the ahoy chapter's user scope is `~/.abcd.noindex/`, and the file lives there

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:490` — "rules.json the machine's rule conventions, the user layer"
- End B: `.abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19` — "now live once in `~/.abcd/rules.json` and inject into every"

itd-117 says a machine's shared conventions now live once in `~/.abcd/rules.json`. The ahoy chapter lists the contents of the one user-scope directory, `~/.abcd.noindex/`, including `rules.json`: the machine's rule conventions, the layer between the bundled domains and each repo's `.abcd/rules.json`, citing itd-117. That is two names for one user-scope home. A person who writes the file where itd-117 says would write one the loader does not read.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 371.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both were true when written; the home rename is recorded by itd-2610030720038073 and adr-2610030720195401, so a shipped intent's old spelling is history, not a live contradiction.
