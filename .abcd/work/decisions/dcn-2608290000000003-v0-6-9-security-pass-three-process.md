---
id: dcn-2608290000000003
slug: v0-6-9-security-pass-three-process
date: 2026-08-29
---
2026-08-29 — v0.6.9 security pass, three process decisions recorded so they are not re-litigated. (a) The four GitHub bug-hunt issues (#485–#488) were verified before any capture and found already fixed on main by the 2026-08-27 security cut; they were closed against their fixing commits rather than re-captured, because a duplicate record beside a resolved one is noise the ledger's status signal cannot carry. (b) The installer's proxy/CA-bundle lockdown keeps no environment-variable escape hatch: the lockdown is the GHSA-x4v8 fix and an env opt-out reopens the vector; the availability cost (NixOS, corporate proxies) is captured as an open record for a product decision, and only the failure message was improved. (c) Every fix branch received two independent adversarial reviews (security + code) from reviewers that did not author the change, repeated until neither returned FIX-FIRST, and the assembled multi-branch diff received the same two again before the PR; the fix PR merges with a merge commit, never squash or rebase, so every resolved_by.commit stamp stays reachable (RS003).
