---
id: dcn-2611130923123497
slug: a-drain-move-s-spot-check-note
date: 2026-11-13
---
2026-11-13 — A drain move's spot-check note (spc-2610301925429250 step 1) counts the records the move tried to write and could not, with the reason. A record the store can never keep, a hand-back with no time or an id outside the store's form, is counted apart, in a number that stays the same at every move while the drain's state holds it, rather than named once: naming it once would need a field the drain's state file does not have, and its schema version stays as it is.
