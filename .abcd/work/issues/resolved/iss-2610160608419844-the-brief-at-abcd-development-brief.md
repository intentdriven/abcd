---
schema_version: 1
id: "iss-2610160608419844"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/21-update.md"
remedy: "Name the attestations API and the Sigstore TUF repository as update's other network destinations at 21-update.md:9-14 or :92."
resolution: "the update brief now names GitHub's API (attestations, tag commit) and Sigstore's TUF repository as the verb's other destinations, and says the redirect rule covers them"
impact: fix
resolved_by:
  commit: "8ba003074e3c2d58d41ed08197fce207f89670eb"
---

The brief at .abcd/development/brief/04-surfaces/21-update.md:9-14 says update, with its check, is the only command that reaches the release origin, and at :92 that its transport redirects only onto the release origin's own hosts; it names no other network destination. update also reaches GitHub's attestations API and Sigstore's TUF repository (tuf-repo-cdn.sigstore.dev) to verify build provenance (commands/update.md:26-28).
