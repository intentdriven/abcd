---
id: dcn-2607220000000001
slug: a-release-is-a-release-regardless-of
date: 2026-07-22
---
2026-07-22 — A release is a release regardless of path: the clean-cutover
manual roll follows the SAME two-commit release-branch shape as a derived
ship (the content commit, then a receipts commit carrying the sha-keyed
PROMOTE receipts for docs-currency-reviewer and iss35-brief-surface-crosscheck).
The v0.4.0 roll landed as one commit with no receipts; the receipt gate
refused fail-closed — its first genuine firing, and correct. Recovery uses
the workflow's own escape hatch (tag on the receipts commit; content = tag^).
Rejected: weakening the required-gates list to unblock the tag — that edits
the release contract the whole programme was built to preserve.
