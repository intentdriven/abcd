---
schema_version: 1
id: "iss-104"
slug: "intent-quoted-text-create-treats-any-non"
severity: "minor"
impact: fix
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "Wired the shared suspectedTypoedSubcommand guard into the intent quoted-text create path; generalised its shape check from issIDRe to a separate recordIDRe (iss|itd|spc) so intent's itd/spc ids are recognised, leaving issIDRe's --blocked-by validation untouched. intent paln / intent lnk itd-5 now exit 2 with a did-you-mean and file nothing; prose titles still file. Detector: intent_surface_test.go, watched fail->pass. ruthless SHIP."
---

intent quoted-text create treats any non-subcommand first token as create text (no suspectedTypoedSubcommand guard like capture): 'abcd intent paln "x"' files a draft instead of erroring; consider porting capture's typo heuristic for symmetry