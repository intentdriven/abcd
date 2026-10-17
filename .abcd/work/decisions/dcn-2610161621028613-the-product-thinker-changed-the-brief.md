---
id: dcn-2610161621028613
slug: the-product-thinker-changed-the-brief
date: 2026-10-16
---
2026-10-16 — The product thinker changed the brief for iss-2610161612200864 from teaching alone to teaching and enforcement: the guard refuses a bash -c or sh -c script built from variables that runs rm, rmdir, unlink or shred, and the refusal says how to rewrite it (write the script to a file in one command and run it in the next; delete with literal paths through /bin/rm -f --, or a variable guarded as "${VAR:?}", outside the -c string), rather than warning. The agent gets the reason back and rewrites the command before the host would stop to ask a person. Only that shape is refused: a -c script with no deleting command, or one with no variable in it, still passes.
