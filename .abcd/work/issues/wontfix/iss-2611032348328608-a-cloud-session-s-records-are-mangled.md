---
schema_version: 1
id: "iss-2611032348328608"
slug: "a-cloud-session-s-records-are-mangled"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes and the walled subscription test, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/cloud/connectors/claude/setup.sh"
remedy: "Name the cloud setup's account something no record contains, or exempt generic account names from the scanner; a test writes a record naming abcd under that account and asserts it survives."
wontfix_reason: "The same defect as iss-2611041104094416: the identity scanner redacted the cloud setup's account name, and #1384 resolved that one by listing the name among those that identify no person, so a record written under that account now keeps the product's name."
duplicates: [iss-2611041104094416]
---

A cloud session's records are mangled: the cloud connector's setup names its OS account abcd, and the identity scanner flags the caller's account name, so record text containing abcd is redacted to [redacted-user]. #1384 fixed only the tests this breaks, by running their scans as a neutral caller; a record written in a cloud session still loses the product's own name.

## Grounds

- declined: The same defect as iss-2611041104094416: the identity scanner redacted the cloud setup's account name, and #1384 resolved that one by listing the name among those that identify no person, so a record written under that account now keeps the product's name.
