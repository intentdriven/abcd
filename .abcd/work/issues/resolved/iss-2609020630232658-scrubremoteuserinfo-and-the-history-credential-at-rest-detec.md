---
schema_version: 1
id: "iss-2609020630232658"
slug: "scrubremoteuserinfo-and-the-history-credential-at-rest-detec"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/remote_userinfo.go"
resolution: "Every userinfo colon test runs on the decoded userinfo: ahoy's scrub (and so the at-rest detector and its heal) and memory ingest's two refusal renderers; undecodable userinfo fails closed."
impact: fix
resolved_by:
  commit: "4c34bb9d"
---

scrubRemoteUserinfo and the history.credential_at_rest detector decide that a userinfo carries a password by a literal colon, but git percent-decodes userinfo, so a remote such as ssh://user%3Apw@host/owner/repo.git under a non-http scheme is neither scrubbed at the derivation site nor detected at rest: the encoded password is stored verbatim in index.json and meta.json and the heal never fires. Reachability is thin (no credential helper is known to write this form) so this is a coverage hole in the new detector rather than a demonstrated leak; the fix is to percent-decode the userinfo before the colon test, for every scheme.

## Grounds

- pursued: ssh://user%3Apw@host is scrubbed and detected at rest while a bare or double-encoded login is kept; an encoded password surviving any of the three sites would show it wrong
