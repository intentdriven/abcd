---
schema_version: 1
id: "iss-2610150733196835"
slug: "abcd-site-its-describe-report-names-the"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "itd-2609151138388536 build review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site"
remedy: "Add the record.decisions switch to the site Describe report beside issue_ledger, with a test."
---

abcd site (its Describe report) names the issue-ledger publication switch but not the new decisions switch, so a person cannot see from the report whether the Decision log page will be published.
