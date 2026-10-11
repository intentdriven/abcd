---
schema_version: 1
id: "iss-2609291925387011"
slug: "the-live-check-of-abcd-site-setup"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2609061543533170-abcd-sets-up-a-managed-repository-s.md"
related_intents: [itd-2609061543533170]
deferred_after: "v0.11.1"
deferral_reason: "Owed to a person under the product thinker's ruling of 2026-09-29 on paid services: the live paid check is recorded as owed and done later by a person, not by an agent, because it spends the person's own paid credential. Deferred out loud by autonomous run A (lane recRulings, 2026-09-29) until that person runs it; capture defer refuses a minor record, so these two fields were set by hand (the iss-2609281654467661 precedent)."
---

The live check of abcd site setup against a real hosting provider is owed to a person. itd-2609061543533170 shipped on tests against a simulated provider, and the product thinker ruled on 2026-09-29 that each paid service (live site setup, the API adapter, the decision adapter) may close on tests against a simulated service, with the live paid check recorded as owed and done later by a person. Owed: in a managed repository, with a real hosting credential configured, run abcd site setup and confirm that the host is created and routed through the provider adapter, that the reported live address serves the rendered site, and that nothing beyond the one host is created or billed; record the result on this record and resolve it.
