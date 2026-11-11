---
id: dcn-2611101810211442
slug: facilitator-on-delegation-for
date: 2026-11-10
---
2026-11-10 — Facilitator on delegation, for itd-2610170634394993, reversing the part of dcn-2611101719521374 that left this account's arming already in place as it is when land's dependency check waits or refuses. A wrong queue is the costly failure, and that arming could merge the dependant before what it depends on. land now withdraws it on a wait and on every refusal of the check, through WithdrawArming, the one withdrawal land already has (auto-merge only: a pull request already queued is reported before the check runs), then asks the forge again that it came off, and the note or refusal says land withdrew it in place of saying nothing was armed. A withdrawal that fails, or that the forge does not confirm, refuses with the command that withdraws it by hand. Another account's arming is never touched. This account's arming at a head land did not gate is now refused before the check, so a wait cannot hide it. The rest of that record stands.
