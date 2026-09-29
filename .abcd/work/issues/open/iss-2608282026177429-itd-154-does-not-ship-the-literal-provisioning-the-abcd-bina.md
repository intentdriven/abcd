---
schema_version: 1
id: "iss-2608282026177429"
slug: "itd-154-does-not-ship-the-literal-provisioning-the-abcd-bina"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "itd-154 adversarial review follow-up"
found_at: "hooks/bootstrap.sh"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker: spc-47 AC2 asks for a literal 'provisioning the abcd binary' line, which would take the only stderr line a transcript keeps from the success notice and from the refusal's cause; the record's breadcrumb design (a marker the next session folds into its own line) would satisfy both but changes the install hook, a trust path. Choosing between building the breadcrumb and amending AC2 is the ruling (drain lane drainRest, run A, 2026-09-29)."
---

itd-154 does not ship the literal 'provisioning the abcd binary...' stderr line spc-47 AC2 asks for, and the reason is a conflict inside the record rather than an oversight: only the FIRST line of a hook's stderr reaches the transcript (iss-208, measured on the first manual install), and bootstrap.sh already spends that line on the success notice's one-time ahoy-install instruction, placed first for exactly that reason (iss-207). An announcement printed ahead of it takes the line from the success and, worse, from the refusal's cause on the failing path — which is the silence itd-154 exists to end. Both orderings were reproduced during the adversarial review. What ships instead is the EXIT trap that converts a silent death into the same loud refusal, naming provisioning in the one line it emits; the residue is a run that HANGS (process still alive, trap not yet fired) or is SIGKILLed, which still leaves nothing. The fix that would satisfy both is a breadcrumb: write a marker at the start of provisioning, remove it on any terminal line, and have the next session fold 'a previous attempt did not finish' into its own terminal line rather than into a new one. Detector: kill -9 a provisioning run, start a second session, expect the next run's first line to name the previous failure.