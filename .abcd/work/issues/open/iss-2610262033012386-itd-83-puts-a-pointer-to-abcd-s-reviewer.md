---
schema_version: 1
id: "iss-2610262033012386"
slug: "itd-83-puts-a-pointer-to-abcd-s-reviewer"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 349 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-83-review-bar-fires-itself.md:66"
remedy: "none (filed automatically)"
related_intents: [itd-83]
finding_ref: cfk-9fb2bd9c1fb75fe0
---

itd-83 puts a pointer to abcd's reviewer agents in the AGENTS.md conventions section that prepare-this-repo owns; the ahoy page says that block never mentions abcd, and the prepare page is retired.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-83-review-bar-fires-itself.md:66` — "a one-line pointer in the marked conventions section of `AGENTS.md` that `prepare-this-repo` already owns"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:385` — "The working-conventions block written into `AGENTS.md` never mentions abcd, the page, or any private repository"

itd-83's fallback for a harness that does not run the hook is a one-line pointer naming abcd's reviewer agents, placed in the marked conventions section of AGENTS.md owned by prepare-this-repo. The ahoy page says the working-conventions block written into AGENTS.md never mentions abcd. It also says the preparation was folded into the install and the retired page's name is gone. A pointer that names abcd's agents cannot sit in a block that never names abcd, and the owner itd-83 relies on no longer exists. The other marked block, the marker block, does name abcd, but at the default docs target it is written into no conventions file. So 'install abcd and the review bar is simply on' fails for a harness without hooks either way.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 349.
