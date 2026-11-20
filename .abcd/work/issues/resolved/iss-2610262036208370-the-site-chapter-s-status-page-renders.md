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
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Should the Now/Next/Later block keep the name status (status block, status page, the status family in the record-families glossary) now that abcd status is the shared-remote view, or be renamed as itd-2609151838312703 promised? Answer 1: rename the block's surfaces so status names one thing. Answer 2: keep the block's names and drop the rename claim from the intent. Recommend a narrow answer 1: stop calling the site page the status page (it is already the record health page) and record that the block keeps its internal name."
related_intents: [itd-2609151838312703]
finding_ref: cfk-9848ee40938cdd86
resolution: "The site chapter, commands/abcd.md, ACKNOWLEDGEMENTS.md and the site package's comments and test messages now call /record/health/ the record health page instead of the status page, because status names abcd status. The site chapter also records that the Now / Next / Later block keeps its internal name, status (its ui.json labels, the pages.status switch, statusblock). TestTheHealthPageIsNotCalledTheStatusPage holds that prose to the ruling. Three brief files under .abcd/ that the drive's guard would not let this lane edit still say Status page; report.md names them."
impact: internal
resolved_by:
  commit: "eb1e018292af23fd10d3480b26b0853bfd184561"
---

The site chapter's 'status page' renders the board's Now/Next/Later block; itd-2609151838312703 renames the board so that the word status is free for abcd status, the view of announcements and reservations.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/22-site.md:111` — "The status page is the record health page, `/record/health/`."
- End B: `.abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:120` — "The status board's rename also frees the word "status" for this command."

22-site calls the record health page the status page and opens it with the block the bare abcd board carries. itd-2609151838312703 gives the word status to the people's view of announcements, reservations and release cuts, and renames the status board to the dashboard to free the word. After that rename, the site's status page shows the dashboard's block while status names a different view.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 641.

## Grounds

- pursued: the narrow answer 1 of dcn-2610271112330536. No living prose this lane could reach calls the health page the status page, and the site chapter records that the block keeps its internal name. This would be shown wrong if a reader still finds the site page called the status page in the site chapter, the plugin's board page or the acknowledgements, or if the rendered site's labels changed.
