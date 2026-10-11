---
schema_version: 1
id: "iss-2610040758569116"
slug: "abcd-decide-writes-an-adr-as-proposed"
severity: "minor"
category: "ux"
source: "managed-repo"
found_during: "downstream brief-authoring lab report, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/decide/decide.go"
remedy: "Add an accept sub-verb to decide that sets status: accepted on a proposed ADR and stamps who accepted it and when, refusing a record that is not proposed; an agent may run it, disclosed by the commit's Assisted-by trailer, with no guard block (product thinker ruling 2026-10-09: 'Agents too, disclosed'). The reviewed code is parked on local branch parked/decide-accept-iss-2610040758569116 (bf7930a27) for reuse; test: accept on a proposed fixture ADR writes the status and the stamp, and a second accept is refused."
---

abcd decide writes an ADR as proposed, and its help says the status stays proposed until the author sets accepted, but no verb sets it: accepting a decision means hand-editing status:, and nothing records who accepted it or when. A downstream project accepting its first decisions did exactly that.
