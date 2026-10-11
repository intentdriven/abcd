---
id: adr-2610030720195401
slug: abcd-keeps-its-own-folders-out-of
status: accepted
date: 2026-10-03
supersedes: null
superseded_by: null
related_intents: [itd-2610030720038073]
related_rfcs: []
related_adrs: [adr-2609091248200336]
---

# ADR-2610030720195401: abcd keeps its own folders out of desktop indexing only by changing those folders, never the computer's search settings

## Context

abcd's home under the person's home folder holds its stores (lane worktrees, run logs, transcripts, the lab, the sources library, and caches), which the desktop indexer scans as ordinary user files even though the dot-prefixed folder is already left out of search results. On 2026-10-02 eight new worktrees set off an indexing burst: the run log records the machine's load at 110 to 148 with no test running and names the indexer's processes (itd-2610030720038073). Keeping abcd's folders out of the indexer's work is wanted by default.

The computer's search settings, its search privacy list among them, are the person's: a setting there reaches every program on the computer, not only abcd, and abcd could write it programmatically only through a root-only path. The principle `the-users-directory-is-theirs`, whose ruling is adr-2609091248200336, governs where a tool creates files and directories; this rule extends the same stance from the person's folders to the person's settings.

## Decision

abcd keeps a folder out of desktop indexing only by changing that folder itself, inside the space abcd already owns (a folder name the indexer honours: the `.noindex` suffix, which the intent's decision 2 chose for the home). It never edits the computer's search settings or privacy list, never asks for administrator rights to do so, and never touches a folder outside its own space. A person who wants a system-wide exclusion sets it themselves. Routed by the itd-84 decomposition; the product thinker confirmed the routing on 2026-10-03, as the decomposition-calibration research note records. It stayed `proposed` until the intent's dated receipt (criterion D6) was taken, and was accepted with it on 2026-10-04.

## Alternatives Considered

- Change only abcd's own folders (chosen): needs no extra rights, is undone by renaming the folder back, and stays inside abcd's declared space.
- Add abcd's home to the system search privacy list: edits a setting that is the person's and reaches every program, which abcd could write only through a root-only path.
- Turn indexing off for the whole volume: far beyond what is needed, and the person's search stops working.
- Do nothing and document a manual step: leaves the load burst as the default, the outcome the intent exists to remove.

## Consequences

The method is limited to a name the indexer honours per folder. No interface reports whether a folder is scanned, so whether an OS version honours the name is a dated receipt per version; a version that stops honouring it is a finding put to the person, never an escalation to system settings. The rule is brief invariant 21, landed with this record's acceptance.
