---
schema_version: 1
id: "iss-2610241529139689"
slug: "setting-up-the-reservation-sweep-s"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "setting up the reservation-sweep App with the product thinker, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/reservation-sweep.yml"
remedy: "Add a setup verb for the sweep's App that reads the current state first (the App, its installation, the environment's branch rule, the two secrets, the ruleset's bypass actor), reports what is already done, creates the App through GitHub's App manifest flow so the person approves one page and the key returns to the verb without being downloaded by hand, then sets the secrets and the bypass through the API; and keep the hand steps out of the design record, which goes stale as GitHub's settings pages move."
---

Setting up the reservation sweep's GitHub App took the product thinker about twenty minutes of wrong turns on 2026-10-24. The only instructions are hand steps in the sweep's design record, and GitHub has since moved them: private keys now sit under Credentials, then Key pairs, and installing an App owned by one organisation skips the choice of account. Nothing checked what already existed: an earlier App, the environment's branch rule and both secrets were set up on 2026-10-17, and only the ruleset's bypass actor was missing, so a second App was created and the first is now orphaned. The person asked for the setup to be precise; it should not depend on a person following settings pages at all.
