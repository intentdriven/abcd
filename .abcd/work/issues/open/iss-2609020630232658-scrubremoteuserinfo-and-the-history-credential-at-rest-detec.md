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
---

scrubRemoteUserinfo and the history.credential_at_rest detector decide that a userinfo carries a password by a literal colon, but git percent-decodes userinfo, so a remote such as ssh://user%3Apw@host/owner/repo.git under a non-http scheme is neither scrubbed at the derivation site nor detected at rest: the encoded password is stored verbatim in index.json and meta.json and the heal never fires. Reachability is thin (no credential helper is known to write this form) so this is a coverage hole in the new detector rather than a demonstrated leak; the fix is to percent-decode the userinfo before the colon test, for every scheme.
