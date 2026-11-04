---
schema_version: 1
id: "iss-2611040554208787"
slug: "06-capture-md-296-says-every-refusal-of"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/06-capture.md"
remedy: "Make these three capture promote refusals exit 2, with a test for each."
---

06-capture.md:296 says every refusal of a ledger verb's own input exits 2, naming a malformed id and a flag value naming nothing, on promoting among others, and leaves exit 1 to a fault. abcd capture promote exits 1 on capture promote bad, on capture promote <iss-N> --intent itd-9 (not in the intent store, nothing stamped), and on a second promote of a promoted issue (06-capture.md:549 lists that as a refusal). A malformed id exits 2 on resolve and defer, and an unknown iss-N exits 2 on promote.
