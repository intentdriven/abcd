---
id: dcn-2610020000000007
slug: ruling-cd2-of-the-person-as-technical
date: 2026-10-02
---
2026-10-02 — Ruling CD2 of the person as technical facilitator (2026-09-29, same interview and local-tier file as the DR3 entry above; recorded by lane rd2), verbatim: "repository route to a keyed provider: WARN AND SKIP that route (other commands keep working), not a hard LoadAPI refusal." Built: a repository's route to a provider that holds a key is skipped with a diagnostic and the machine's own route to that name applies in its place (`skipKeyedRoute` in `internal/core/oracle/config.go`; a repository's malformed route and malformed route name are skipped under the same ruling); decision 8 of itd-2609081951381895 and adr-2609221009491186 carry it.
