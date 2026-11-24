---
id: dcn-2611240757179929
slug: defer-the-per-push-cost-of-a-release
date: 2026-11-24
---
2026-11-24 — Defer the per-push cost of a release reservation's push placement (iss-2611240757104266) past v0.17.1: it fails safe (an unplaced push reads unverified, with the reason named), it is reached only when several hundred commits on one release branch each arrived as their own push, and the release had to ship. The claim tests' TestMain widens pusherBudget so their walk cases judge placement, not the runner's speed. Reverse by fixing iss-2611240757104266 (one walk per fetch instead of one git process per push).
