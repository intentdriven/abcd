---
schema_version: 1
id: "iss-2610262039356482"
slug: "itd-117-puts-the-user-rules-layer-at"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 871 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-117]
finding_ref: cfk-ee2aeea467092f02
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Decisions 6 and 7 of the rename intent introduce the stop on the old folder; the shipped intent itd-117 cannot have named them."
---

itd-117 puts the user rules layer at `~/.abcd/rules.json`; the configuration chapter puts it under `~/.abcd.noindex/` and treats any entry at `~/.abcd` as a stop

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19` — "now live once in `~/.abcd/rules.json` and inject into every"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:702` — "The folder's earlier name, `~/.abcd`, is read by `abcdhome.Check` alone, with `Lstat`, and never by a reader"

itd-117 says the machine's shared conventions live once in `~/.abcd/rules.json`. The configuration chapter says the user layer is `~/.abcd.noindex/rules.json`. It also says the folder's earlier name `~/.abcd` is never read by a reader, and while any entry stands there every command and hook writes nothing and prints the rename. Someone following the intent would create the very path that halts abcd.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 871.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Decisions 6 and 7 of the rename intent introduce the stop on the old folder; the shipped intent itd-117 cannot have named them.
