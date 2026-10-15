---
id: dcn-2608170000000003
slug: command-parents-owe-two-guarantees-not
date: 2026-08-17
---
2026-08-17 — Command parents owe TWO guarantees, not one (iss-266, #293).
A parent refuses a mistyped sub-verb only if it is Runnable AND declares an
Args validator; missing either exits 0 silently (no RunE ⇒ cobra returns
flag.ErrHelp before ValidateArgs; Args nil ⇒ legacyArgs falls through to
ArbitraryArgs). The house convention is therefore `RunE: helpRunE` plus an
explicit validator on every parent, asserted structurally over the live
command tree rather than a hand-kept list. `capture` and `intent` are a
named exemption — their positional is free text, guarded by their own
suspected-typo check — and the exemption list is itself tested against
reality so a stale name cannot quietly drop a parent from the sweep.
Consequence accepted and filed as iss-267: exit 2 is the *blocking* status
on the hook plane, so hook sub-verb names are now a compatibility contract
and a future rename must carry an alias rather than rely on the old
silent-exit-0 cushion. Rejected: fixing only the parents named in the issue
(the same hole would return with the next parent added), and asserting the
behaviour with a hand-maintained list of parents (it had already drifted —
`identity` and `intent audit` were absent yet not exempt).
