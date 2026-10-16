---
id: dcn-2607150000000001
slug: the-lifeboat-coverage-experiment-s
date: 2026-07-15
---
2026-07-15 (M2 gate — maintainer-approved) — The lifeboat coverage experiment's
cross-repo readout is in. Corpus (private repos anonymised): abcd-cli
(git+conventions+abcd-native, 21/2/0 grounded/partial/blank), test repo 1 and
test repo 2 (abcd-native scaffolding but no authored brief, 4/8/11 and 2/6/15),
test repo 3 (git+conventions, no abcd, 3→4/8/11), and a git-only floor (0/2/21).
Headline finding: **scaffolding is not a record** — test repos 1 and 2 carry
`.abcd/` directories yet ground barely more than the record-less test repo 3,
because their `.abcd/development/` has no `brief/`, no ADRs, no issue ledger; the
native adapter is honest and grounds only authored prose. The
brief structure holds (excluding the dogfood repo, 9 of 23 sections are blank across
the messy corpus, not half). Decisions: (1) `product/personas` is demoted to a
human-answered question in the lifeboat brief — the corpus confirms the M0 prediction
that it is not derivable from a repository. (2) The other 8 always-blank sections stay
in the brief but split: `product/mental-model`, `delivery/verification-matrix`,
`delivery/out-of-scope` become human-answered questions; `evidence/what-didnt`,
`evidence/open-questions`, `constraints/naming`, `glossary`, `internals` are blank more
from thin adapters than genuine non-derivability and get adapter work before M3 decides
(iss-98, iss-99, iss-100). (3) The dependency-manifest adapter under-detected Python/
Ruby/PHP packaging (test repo 3's pyproject.toml+uv.lock read as blank) — fixed now, so
test repo 3's `constraints/dependencies` grounds. M3 (the packer) builds to this list:
grounded/partial sections extracted-and-cited, the human-question sections surfaced as
the blanks-with-questions the coverage report already produces.
