---
schema_version: 1
id: "iss-2609292030070275"
slug: "the-dependency-bump-re-authoring-itd-2609221842494980"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25, lane depReauthor"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/dependency-reauthor.yml"
deferred_after: "v0.11.1"
deferral_reason: "owed by the person (rulings 7 and H2 of 2026-09-29): creating and installing the GitHub App, storing its two Dependabot secrets and setting the owner are the person's acts; the lane built everything else and cannot run the live forge"
remedy: "the person creates and installs the App, stores the two Dependabot secrets and sets the owner; one live bump is watched through, then spc-2609221843355061 closes"
---

The dependency-bump re-authoring (itd-2609221842494980, spc-2609221843355061) has never run against the live forge, so its first acceptance criterion is proven only offline, against a stand-in App API and a local remote. The person's steps are owed: create a GitHub App with Contents read and write, install it on this repository, store its id and private key as the Dependabot secrets DEPENDENCY_REAUTHOR_APP_ID and DEPENDENCY_REAUTHOR_APP_KEY, and set owner_name and owner_email in .abcd/config/dependency-reauthor.conf. Until then every in-bound bump is refused by name and stays a person's to land. Done when one real dependabot go_modules or pip bump is watched re-authored as the owner, re-checked green by the attribution gate and merged; then the spec closes and the intent ships (Refs: iss-2609221820487644).
