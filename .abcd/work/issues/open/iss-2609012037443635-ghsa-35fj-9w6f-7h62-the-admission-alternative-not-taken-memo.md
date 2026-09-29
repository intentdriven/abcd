---
schema_version: 1
id: "iss-2609012037443635"
slug: "ghsa-35fj-9w6f-7h62-the-admission-alternative-not-taken-memo"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/ingest.go"
remedy: "Waits on ruling CA2; if https-only stays, move the record to wontfix naming invariant 12 and abcd update's same posture; if a flag is added, admit an http source only under an explicit --allow-http, and have ingestRedirectPolicy (internal/core/memory/ingest.go) refuse every hop from https to http whatever the flag, the two flag variants differing only on whether an http source may redirect to http. Prove it with httptest servers: an https source redirecting to http refused with and without the flag, and an http source refused without it."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lane drainFresh of autonomous run A, 2026-09-29): Add an explicit --allow-http to memory ingest, and if so may a hop be plaintext only when the source itself is? An implementation of exactly that shape was written on 2026-09-02 (commit 300968594, branch feat/ruled-security-forks) and never merged; main still admits https only."
---

GHSA-35fj-9w6f-7h62, the admission alternative not taken: memory ingest now admits https sources only, with no escape — the posture of abcd update and of invariant 12 in the brief. The advisory proposed an additive --allow-http flag instead (https by default, explicit opt-in for a consumer that needs a plaintext source such as an internal mirror or a local test server), wired on the CLI, commands/memory.md and the reference page. Left open as the fork: a consumer that needs plaintext has no route today; adding one is additive, and needs a decision on whether the flag also relaxes the per-hop scheme pin or only the admission.

## Remedy grounds (2026-09-29)

- Why: CA2's three answers; it is unanswered and none is picked. The unmerged commit 300968594 on feat/ruled-security-forks is the starting point if the flag is ruled in.
- Sources (consulted 2026-09-29): curl restricts redirects to HTTP, HTTPS, FTP and FTPS by default since 7.65.2 and documents --proto-redir =https to forbid a downgrade (https://github.com/curl/curl/blob/master/docs/cmdline-opts/proto-redir.md); Go's http.Client exposes CheckRedirect for a per-hop policy (https://pkg.go.dev/net/http#Client); RFC 6797 (HSTS) treats a downgrade of a known-https host as an attack (https://www.rfc-editor.org/rfc/rfc6797). No answer relaxes the https-to-http refusal.
- Rejected: a flag that also relaxes the per-hop pin, which would let an https source be downgraded in transit.
