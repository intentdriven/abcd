---
schema_version: 1
id: "iss-2610110939559446"
slug: "every-push-already-carries-a-local-make"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "CI-cycles research, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "post the local preflight receipt as a commit status (the gh-signoff pattern) only to skip duplicate pull-request-level work, never in the merge queue, and only once a receipt can be attested rather than asserted"
---

Every push already carries a local make preflight receipt that ran the same gates CI runs, but CI cannot see it, so the pull-request run repeats the work. The receipt is the pusher's word, not an attestation, and with many agents pushing that is too weak to replace the queue run; it could only spare duplicate work before the queue.
