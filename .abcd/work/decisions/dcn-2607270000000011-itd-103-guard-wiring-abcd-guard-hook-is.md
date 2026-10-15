---
id: dcn-2607270000000011
slug: itd-103-guard-wiring-abcd-guard-hook-is
date: 2026-07-27
---
2026-07-27: itd-103 guard wiring — `abcd guard hook` is a sub-verb of the USER-facing `guard` verb, not a fifth entrypoint under the hidden `hook` subtree. Reason: guard health has to be legible (AC 1), and a hidden adapter documents nothing; the other `hook` entrypoints are injection transport with no user-facing counterpart, this one is the same decision the user can also ask for by hand. Consequence: it appears in the generated CLI reference and must stay host-agnostic in its prose.
