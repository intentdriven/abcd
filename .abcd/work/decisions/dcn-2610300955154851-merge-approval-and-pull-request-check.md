---
id: dcn-2610300955154851
slug: merge-approval-and-pull-request-check
date: 2026-10-30
---
2026-10-30 — Merge approval and pull-request check rulings. Human mode's merge approval is a rule agents keep, not a proof of who approved: it stops an agent merging by mistake, not one set on getting round it, abcd says so wherever it describes the mode, and no proof mechanism enters this release. The person approves in the terminal through an abcd command (abcd land --approve today, or a dedicated verb), so the current act stands; the dashboard or a web app is one more channel for the same approval (itd-2610300955049260), not a gap. Agents working under a forge account of their own is optional work, designed from current practice to run in the cloud and locally, through an API key or a subscription (itd-2610300955026531). Requiring a security key touch for an approval is later work (itd-2610300955031405). The three checks every pull request runs, the change labeller, the dependency review and the vulnerability scan, stay non-blocking, and a failure in any of them is fixed without being asked until it passes.
