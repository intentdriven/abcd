---
schema_version: 1
id: "iss-2609281546130900"
slug: "every-scanner-new-spends-four-git-execs-reading-the-caller-s"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: ruling AR"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/identity.go"
resolution: "ProbeIdentity reads every identity key in one git config -z --get-regexp listing under the same scrubbed env; a second listing runs only when the parent carries git -c configuration, which can move discovery or fail the command. A 37-shape pinned table passed at the base and passes unchanged; a counting git shim holds scanner.New to one exec (two with -c)."
impact: internal
resolved_by:
  commit: "1683ed2ce"
---

Every scanner.New spends four git execs reading the caller's identity: ProbeIdentity (internal/adapter/scanner/identity.go) runs git config --get-all user.name, --get-all user.email, -z --get-regexp over user/author/committer name and email, and --get remote.origin.url, each a separate process under the scrubbed env, although the first, second and fourth read the same configuration the third lists. The review of the ciFast lane counted about 2,193 such execs per cli test run; the cost lands on every CLI verb and hook that builds a scanner. One NUL-delimited listing under the same env yields everything the four did.

## Grounds

- pursued: scanner.New spends one git exec on identity with no -c configuration in the parent and two with it, and ProbeIdentity answers every pinned shape exactly as the four-exec probe did; a counting-shim run above one exec, or any pinned shape answering differently, would show it wrong
