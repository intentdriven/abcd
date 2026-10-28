---
schema_version: 1
id: "iss-2610281718353935"
slug: "publishing-a-release-waits-for-a-person"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/release.yml"
remedy: "Record in an ADR that the release environment carries no required reviewer, partly superseding adr-43 point 2, and make every description of the gate say so; the live setting is changed after the merge."
---

Publishing a release waits for a person to approve the release environment's pending deployment. The product thinker ruled on 2026-10-28 that publishing a release and deploying the site run with no approval step by anyone, so the reviewer on the release environment, and every description of it, no longer matches what the project wants.
