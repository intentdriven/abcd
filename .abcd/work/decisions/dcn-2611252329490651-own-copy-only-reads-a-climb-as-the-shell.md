---
id: dcn-2611252329490651
slug: own-copy-only-reads-a-climb-as-the-shell
date: 2026-11-25
---
2026-11-25 — own_copy_only reads a climb as the shell reads it and judges by depth, refining dcn-2611252239588252: quoted parts and escaped characters are literal, a brace that does not expand and an unclosed [ stay as written, and a substitution or an extended pattern is judged by its literal prefix and never kept; a climbing pattern that matches outside the copy reads broken only where no folder of its clone is deep enough for the climb to stay inside it or where it names another copy's folder as written, and unexamined otherwise, as does a walk past its bound; a chmod or chflags of a climb reads broken; a bare climb stays kept, because no transcript scan sees every read, and what keeps a launched copy from finding its siblings' folders is that the launch sets the run's folders in the hand-off store and the clone store to mode 0300 while its copies run, owner write and traverse without read, since 0711 keeps the owner's read and the copies run as the owner, giving their modes back on every end, a stop and a failure included.
