---
id: dcn-2610271922493259
slug: the-reservation-sweep-joins-the
date: 2026-10-27
---
2026-10-27 — The reservation sweep joins the read-only fix for iss-2610231038335615 rather than waiting for a capture of its own. Grounds: the record names the sweep's dry run among the reads that wrote the empty tree; the sweep landed on main after the fix branch was cut, so the rebased branch would otherwise resolve the issue while one named read still wrote; and the sweep only compares chain trees with the empty tree's name and never commits on it, so the name-only form is enough. Decided by abcd-0b (technical, delegated).
