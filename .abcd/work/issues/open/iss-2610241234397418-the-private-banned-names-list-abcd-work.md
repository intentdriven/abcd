---
schema_version: 1
id: "iss-2610241234397418"
slug: "the-private-banned-names-list-abcd-work"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "abcd build itd-82 session, 2026-10-24, after the pre-commit warning that the private name guard has no entries"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/banlist"
remedy: "run a state-of-the-art research pass first (prefer-sota: PII and named-entity detection such as Microsoft Presidio, secret and entropy scanners such as gitleaks, private-term mining against public corpora), then add a host-delegated 'abcd banlist propose' that reviews the repository, its history and the local sources corpus, proposes candidate private names with their evidence held locally and never printed to a shared surface, and writes only the entries the person confirms into the keyed private store"
---

The private banned-names list (.abcd/.work.local/private-names.txt, the keyed store the pre-commit name guard checks) can only be filled by hand, entry by entry, so on a fresh install it starts empty and every commit warns that the guard checks nothing. abcd could propose its entries itself: review the repository, its history, its record and the person's other local material for the names that must never reach a committed file (private repositories, collaborators, clients, internal hosts and project code names), and offer them for the person to confirm. How to find such names well, and without leaking them while looking, is a state-of-the-art question (named-entity and PII detection, secret scanners, private-term mining), not one to answer from memory.
