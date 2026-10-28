---
schema_version: 1
id: "iss-2610241212237390"
slug: "there-is-no-command-to-write-a"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "setup redesign build, identity lane, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd ahoy identity recording (branch feat/setup-redesign-identity)"
remedy: "add an abcd ahoy identity record step (or flag) that writes the per-repository record after a confirm, reusing identity.WriteRecord"
---

there is no command to write a per-repository identity record under ~/.abcd.noindex/identity/<root-sha>.json; a person with a work and a personal identity can only create it by hand or through the legacy-pin migration.
