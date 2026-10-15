---
id: dcn-2609250000000010
slug: the-request-spec-close-emits-when-it
date: 2026-09-25
---
2026-09-25 — The request `spec close` emits when it ships an intent carries the `## Routing` section `intent audit <itd-N>` writes; an unreadable routing table does not refuse the close (lane implementer, autonomous run A, fix round 1 on spc-2609180535002478, after the tier2 review's finding 3). The route is a front-door fact, so the core close still emits the request without it and the front door re-emits the owed request with the section once the intent-auditor's route resolves, with the same one-line fallback on stderr. The close is a record move and its emit is report-only (a failed emit already leaves the intent shipped), so a table that cannot be read leaves that request without the section and one stderr warning names the fault and `abcd intent audit <itd-N>` as the re-emit; refusing the close over a file only its report-only step reads would block the record on the wrong thing. The close carries no `--route` and no `routing` member in its result: it is not a delegating verb, and `intent audit <itd-N>` is the documented way to hand the auditor its request.
