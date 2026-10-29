---
schema_version: 1
id: "iss-2610262038254260"
slug: "status-names-the-now-next-later-record"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 794 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/record-families.md:28"
remedy: "Rename the 'status' family row in record-families.md once the board rename lands (iss-2610150726321422 holds the product thinker's 2026-10-15 ruling that the status board becomes the dashboard, freeing 'status' for abcd status); until then qualify the row as 'the status board view'."
finding_ref: cfk-c18ad121f15699d8
---

"status" names the Now / Next / Later record family and also the `/abcd:status` view of announcements on the shared remote.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/record-families.md:28` — "Now / Next / Later, rendered from the shelves, the gate and the build's state"
- End B: `.abcd/development/brief/04-surfaces/39-status.md:3` — "`/abcd:status` is the people's view of what everyone else is doing on the"

record-families defines the status family as the Now / Next / Later view computed from the shelves, the gate and the build's state. 39-status uses status for something else: the announcements sessions post on `abcd-status/*` branches, reservations and releases being cut, shown by `/abcd:status` and the prompt hook's `## STATUS` block. One word now names two unrelated views.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 794.
