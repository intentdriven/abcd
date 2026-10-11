---
id: adr-2610110925465593
slug: decisions-are-one-flat-file-each-and-the
status: accepted
date: 2026-10-11
supersedes: adr-2609151138420062
superseded_by: null
related_intents: [itd-2609151138388536]
related_rfcs: []
related_adrs: [adr-45]
---

# ADR-2610110925465593: Decisions are one flat file each and the single log is retired

## Context

[adr-2609151138420062](2609151138420062-the-decisions-log-is-a-folder-of-minted.md)
ruled on 2026-09-15 that the decisions log becomes a folder of minted records,
one file per decision, with an index derived from the folder and never appended
to. It kept `.abcd/work/DECISIONS.md` as a symlink to that index "for now", so
that existing links, readers and citations went on resolving. It also gave every
repository abcd manages the same shape at adoption.

The conflict that ADR was about recurred on 2026-10-11. Two pull requests each
appended an entry to the tail of `.abcd/work/DECISIONS.md`, and whichever landed
second conflicted in the merge queue. The `merge=union` attribute on the path
does not help there: the forge computes mergeability and the queue's merge
without local merge drivers (iss-2609240646538011). Planning the intent that
enacts the shape (itd-2609151138388536), the product thinker settled the open
questions on the same day, and two of the rulings change what the earlier ADR
decided.

A single appended decisions file is not a common standard: the widely used
conventions for decision records are folders of one file per decision (MADR's
`docs/decisions/NNNN-title.md`, adr-tools' `doc/adr/`). The old path has no
outside readers to keep resolving, only this repository's own links.

## Decision

**A decision is a record, minted through the adr-45 seam, stored one per file in
one flat folder.** The folder is `.abcd/work/decisions/`, and each file is
`dec-<id>-<slug>.md`, the same flat layout every other record family uses. The
id is the timestamp, so a plain listing sorts by date; there are no year, month
or day subfolders, which would spend path length (iss-2610100626320367) on
information the id already carries. Recording a decision creates a file and
appends to nothing.

**`.abcd/work/DECISIONS.md` is retired, not kept as a symlink or a signpost.**
Every link, reader, citation and convention in the repository that names it is
rewritten in the change that lands the shape. A citation of a dated entry points
at that entry's own file. Nothing in the repository names the old path as live
afterwards.

**The history is read through a listing, never through a committed index.** A
command and the site render every decision in date order with its id, derived
from the folder. No committed file is written by each new decision, because that
file would bring the conflict back.

**A repository abcd manages but did not create keeps its existing log.** Its
`DECISIONS.md` is the team's file, and abcd never rewrites it
([adr-2609091248200336](2609091248200336-a-tool-never-creates-directories-in-user.md)'s
caution about acting on a person's files unasked). New decisions go to the
folder from the change that adopts the shape onward. The old file stays where it
is, as frozen history.

**The decisions-append gate is retired by the shape**, as the earlier ADR ruled.
DA001–DA004 police position and preservation inside one appended file, and a
folder of minted files has no such file. The union merge attribute for
`.abcd/work/DECISIONS.md` goes in the same change. The changelog's attribute is
a separate conflict class and is not this decision's business.

## Alternatives Considered

- **Keep `DECISIONS.md` as a symlink to a derived index** (the earlier ADR's
  interim form). Rejected: the index must exist in the tree for the link to
  resolve, and an index every decision regenerates collides as the log did. A
  symlink also arrives as plain text on a Windows checkout without symlink
  support.
- **Keep `DECISIONS.md` as a short signpost page** naming the folder, the
  listing command and the site page. Considered in the interview, and set aside
  by the product thinker for full retirement: the signpost keeps a second name
  for the history alive, and every link to it lands one step short.
- **Date subfolders, `decisions/<year>/<month>/<day>/<title>.md`.** Rejected:
  every other record family is flat with the timestamp in the name, the
  subfolders add about eleven characters to every path the slug cap was set to
  keep short, and the listing can group by period without the tree doing it.
- **Migrate a managed repository's log** (automatically, or after asking at
  install). Rejected for now: abcd does not rewrite a file a team wrote. The
  frozen-history route leaves nothing of theirs rewritten, at the cost of history
  sitting in two places.

## Consequences

- Two branches that each record a decision add two files and conflict on
  nothing, with no merge attribute or driver configured.
- The change that lands this rewrites every mention of `.abcd/work/DECISIONS.md`
  in the repository: 443 mentions in 263 files on 2026-10-11, about fifty of them
  in code. A citation of a dated entry needs a map from each old entry to its new
  file. Where several entries share a date and the citation does not say which,
  the migration cannot choose alone, so it lists those cases for review rather
  than guessing.
- abcd's own existing entries are migrated into the folder with their dates and
  text verbatim, in their original order. The migration and the link rewrite are
  a one-off script run once in this repository (the product thinker's ruling of
  2026-10-11), never a verb or code that ships in abcd: they are needed once and
  only here, and a managed repository is never migrated.
- A managed repository that adopted abcd before this change has its history in
  two places: the frozen file, and the folder from then on.
- adr-2609151138420062 is superseded. Its folder-of-minted-records rule and its
  ban on an appended index stand, restated here; its symlink and its
  same-shape-at-adoption migration do not.
