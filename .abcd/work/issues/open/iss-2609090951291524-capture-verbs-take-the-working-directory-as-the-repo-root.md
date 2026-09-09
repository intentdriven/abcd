---
schema_version: 1
id: "iss-2609090951291524"
slug: "capture-verbs-take-the-working-directory-as-the-repo-root"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
---

Every capture verb hands the caller's working directory to the core as the repo root, verbatim and unresolved: the status, list, resolve, promote, wontfix and disposition commands each build their request with the working directory, and the fast-path capture does the same, so the core takes the explicit-root branch and never consults the git-aware discovery helper that sits behind it. A verb run anywhere but the checkout root therefore addresses a ledger that is not there. Reproduced: from a package directory two levels down the status render reports open 0, resolved 0 and wontfix 0, while the identical binary run from the checkout root reports open 365, and neither run says anything is wrong. The write half is worse than a wrong answer, because the directory tree is created on demand: a capture run from a subdirectory mints a second ledger under it, writes the record there, and reports success with a path the author will not read as unusual, so the record is filed where nothing looks for it and a stray dot-abcd tree appears in a source directory. Confirmed by running the shipped binary in a plain directory that was not a repository at all, where it created the full ledger skeleton and wrote thirteen records into it. The reading verbs get this right and resolve the toplevel first, which is what makes this an omission rather than a design. One further limb is real but currently inert here: the ledger redactor builds its scanner from the root it is handed and the scanner reads the per-repo pattern override two directories down without walking up, so a capture from a subdirectory is redacted with the built-in defaults alone; this repository carries no such override today, so nothing is under-redacted now, and any managed repository that adds one is. iss-2609020224230967 is adjacent and states that the capture package does this correctly, which is true of the core helper and false of the front door that never calls it. Fix direction: resolve the checkout root once at the surface, the way the reading verbs already do, or let the core discover it by passing no root and hardening the helper that then runs. Detector: a capture verb invoked from a subdirectory must address the checkout root ledger, and must refuse rather than silently mint a new store when there is no ledger and no repository.
