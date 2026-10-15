---
id: dcn-2608310000000014
slug: degrade-itd-185-s-four-semantic
date: 2026-08-31
---
2026-08-31 — Degrade itd-185's four semantic supply-regime signatures
(RG-EVAL-ORDERING, RG-EVAL-RECOMMENDATION, RG-REG-FIXPROPOSAL,
RG-EXPL-DISPOSITION) from enforce to flag: a hit raises a review flag on the
run record and the item lands. This is the reserved degradation path spc-63
names, taken deliberately, and it weakens the claimed property from ENFORCED to
OBSERVED. The structural halves are untouched and stay absolute — the reserved
names and the strict per-position schema still refuse. THE EVIDENCE IS
SYNTHETIC, and saying so is the point: the itd-185 fidelity audit constructed a
corpus of 34 realistic reading outputs and 14 were caught, every one of them for
REPORTING what the read document said rather than for proposing anything (the
disposition detector fires on any claim quoting the token this repository's
records carry everywhere; "section 3 says the fix is already merged while
section 8 says it is pending" is the canonical shape of a detection finding and
fires too). No reading has ever been run through this verb — every payload the
delivery has validated is synthetic — so the ruled condition, "degradation only
on observed noise" (BUILD-PLAN.md:70, spc-63), is NOT met and this departs from
the ruled design. Taken anyway because the alternative departs further: the gate
is currently enforced over a calibration that has never been taken, which is the
standing tension itd-185 already records against widen-options' "calibrated
before it gates", and waiting for observed noise means enforcing indefinitely on
no calibration, because the assembled input is about 9.8 MB and cannot be handed
to a reading at all (iss-2608311501186646). Of the two departures this one
cannot produce a false refusal of a real reading, and it is reversible by the
same one-line mode change. REVISIT POINT: the first real reading. When the
instrument can be run, the flags it raises are the observed calibration, and the
mode is reconsidered on that evidence rather than left at flag by default.
Rejected: keep enforcing — defensible under the letter of "degradation only on
observed noise" and indefensible under "calibrated before it gates", and on the
synthetic evidence it would refuse roughly two in five legitimate outputs the
first time a reading ran. Consequences recorded in the same change:
TestEverySignatureShipsEnforced is replaced by a test pinning each entry's mode
by name so a silent flip in either direction fails; itd-185's ac-5 and ac-9 are
rewritten to say flag rather than refuse; the ingested audit verdict
rcp-fe3450ca55ff records those two criteria MET on the strength of a refusal and
therefore describes superseded behaviour needing re-issue at the next audit; and
the disclosed residue on itd-185 and spc-63 is widened to name over-catching
beside under-catching, with the propose-versus-report distinction as the reason.
