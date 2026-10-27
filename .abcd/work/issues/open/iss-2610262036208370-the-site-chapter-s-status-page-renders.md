---
schema_version: 1
id: "iss-2610262036208370"
slug: "the-site-chapter-s-status-page-renders"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 641 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/22-site.md:111"
remedy: "none (filed automatically)"
related_intents: [itd-2609151838312703]
finding_ref: cfk-9848ee40938cdd86
---

The site chapter's 'status page' renders the board's Now/Next/Later block; itd-2609151838312703 renames the board so that the word status is free for abcd status, the view of announcements and reservations.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/22-site.md:111` — "The status page is the record health page, `/record/health/`."
- End B: `.abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:120` — "The status board's rename also frees the word "status" for this command."

22-site calls the record health page the status page and opens it with the block the bare abcd board carries. itd-2609151838312703 gives the word status to the people's view of announcements, reservations and release cuts, and renames the status board to the dashboard to free the word. After that rename, the site's status page shows the dashboard's block while status names a different view.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 641.
