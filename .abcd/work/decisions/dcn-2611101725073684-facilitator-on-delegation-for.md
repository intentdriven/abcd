---
id: dcn-2611101725073684
slug: facilitator-on-delegation-for
date: 2026-11-10
---
2026-11-10 — Facilitator on delegation, for itd-2610170634394993, correcting dcn-2611101719521374. That record said a forge client that fails, as the real one does on a missing number, is refused with its sanitised message. A read on 2026-11-10 showed gh exits 1 on an answer carrying a NOT_FOUND error and keeps the answer on stdout, which land's runner does not keep, so land now also reads the client's failure naming that very number (Could not resolve to a PullRequest with the number of N.) as no pull request, refused with the same reason and fix as the NOT_FOUND answer, telling the author to take the number out of the Depends-On line. A failure naming any other number, or none, is still refused as a forge that could not be asked, with its sanitised message. Each is pinned by a test.
