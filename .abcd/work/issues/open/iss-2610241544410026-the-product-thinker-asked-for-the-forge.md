---
schema_version: 1
id: "iss-2610241544410026"
slug: "the-product-thinker-asked-for-the-forge"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "product thinker, 2026-10-24, after a hand setup of the sweep App went wrong"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/research/notes/2026-10-24-forge-settings-catalogue.md"
remedy: "Add abcd git check, abcd git apply <setting> and abcd git mirror <setting>: forge-neutral verbs over a forge adapter (GitHub first, GitLab and others later, chosen from the repository's remote), which read each setting the catalogue names through the forge's API and report it as matching, differing or unverifiable against the repository's committed mirror; apply what the API allows after the person's yes; and, for the steps only a person can take, print the forge's current documented steps and the direct settings URL."
---

The product thinker asked for the forge settings abcd depends on to be captured as verbs and nouns, with clear, verified steps and direct URLs, after a hand setup of a GitHub App on 2026-10-24 went wrong on stale instructions. The verified catalogue is the research note .abcd/development/research/notes/2026-10-24-forge-settings-catalogue.md: 21 settings, each named forge-neutrally (default-branch protection, release-tag protection, reservation-branch protection, deploy environments and their secrets, automation apps and their bypass, and the rest), its live state read through the API, GitHub's current documented steps quoted, and its direct settings URL. The product thinker named the verb family abcd git, so that GitLab and other forges can be configured in a later update, depending on what the user hosts on.
