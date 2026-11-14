---
schema_version: 1
id: "iss-2610171047088861"
slug: "the-drain-s-renewal-keeps-its-own-copies"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "building the load gate's renewal (iss-2610170926295770)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Once the drain-many-accounts change has landed in the same file, export claim's halfThrough and renewalTerm, have the loop call them and delete its own copies"
resolution: "claim now exports HalfThrough and RenewalTerm, and the implement loop calls them (RenewalTerm at its four renewal sites, HalfThrough in renewalDue) in place of the copies it kept in reservation.go, which are deleted. TestTheLoopTakesTheRenewalRuleFromClaim fails while a non-test loop source works out a reservation's term from EffectiveExpiry and ClaimedAt in the spellings TestTermReckoningsSeesEachSpelling names, and TestTheRenewalRule holds the two exported rules."
impact: fix
resolved_by:
  commit: "33e5321a88fb5541605cf27f7ce81cd035f554e9"
---

The drain's renewal keeps its own copies of when a reservation is half through its lease and of the term a renewal pushes (renewalDue and renewalTerm in internal/core/implement/loop/reservation.go), where claim's halfThrough and renewalTerm (unexported until something outside claim calls them) now hold the same two rules for the load gate's renewal: two copies of one rule can drift apart.
