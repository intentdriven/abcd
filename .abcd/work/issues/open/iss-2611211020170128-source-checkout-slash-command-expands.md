---
schema_version: 1
id: "iss-2611211020170128"
slug: "source-checkout-slash-command-expands"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "an /abcd:ideate run in the source checkout of abcd, 2026-11-20"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/rules.json"
remedy: "Add one rule to the DOGFOODING domain: in this checkout the expanded command page is the installed release page, so before filling any input read the page for that verb under commands/ in the tree and run scripts/abcd.sh explain for it. Grounds: memory-graduates-to-record, the same local note having been written twice in one day."
---

In the source checkout of abcd, the page a slash command expands is the installed release page, and nothing tells the agent so. On 2026-11-20 two separate sessions ran /abcd:ideate here. Each was handed the 0.16.0 page, which has no `current_state` field, no `implies_building` list, no dry run and no line sending the agent to `abcd explain`, while the recorder built from the tree requires `current_state`. Each briefed its two reviewers from the old page, had the finished verdict refused, and had to brief the reviewers a second time: one session resumed its reviewers, the other discarded them and ran a new pair. Each then wrote the same lesson into its local agent memory, four minutes apart. The DOGFOODING domain in `.abcd/rules.json` fires on such a prompt and says which binary to run, but says nothing about the page text. The releases from v0.16.1 on carry the field, and v0.17.0 adds the line that sends the agent to `abcd explain`; these sessions were served the user-scope install, still at v0.16.0, which iss-2610241228359748 holds. The page half stays with the newest release installed too: in this checkout a release's page always trails the tree, so this recurs for every command whose page changed since the release that is installed. iss-2610141647314635 holds the binary half of this; this record is the page half.
