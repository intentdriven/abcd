---
schema_version: 1
id: "iss-216"
slug: "install-instructions-need-a-getting-started-page-once-more-t"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "itd-108 grill session (2026-08-11)"
found_at: "docs/"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Are install instructions a sanctioned place to name a harness (a docs-lint escape) before a getting-started page exists?"
---

Install instructions need a getting-started page once more than one harness is supported, and the version floor needs a home in it. Raised at the itd-108 grill (2026-08-11) while ruling that no migration is owed to existing users, because there are none yet.

Two things have nowhere to live today. First, the archive plugin source itd-108 adopts requires a minimum harness version (v2.1.224 for the harness abcd ships hooks for); below it an install fails, and on older builds the whole marketplace fails to load. A README line is the minimum, but a floor stated only in a README ages badly and is invisible at the moment it bites. Second, abcd is host-agnostic by design and the install story is currently written for exactly one harness, so the moment a second is supported the instructions become a matrix — per-harness add command, per-harness version floor, per-harness plugin-root behaviour — and a single README section cannot carry it without becoming the change-narration docs-lint forbids.

Shape, not adopted: a Diataxis tutorial (getting started) distinct from the how-to and reference pages already under docs/, holding the install path end to end for each supported harness, with the version floor stated where the user meets it rather than in a footnote. Note the constraint that makes this awkward: user-facing prose is held host-agnostic by the harness/* docs-lint rules, and naming a harness is confined to attribution surfaces, so a per-harness install matrix needs either the docs-lint allow escape or a decision that install instructions are a sanctioned place to name a host. That decision is the real content of this issue and should be settled before the page is written, not after.