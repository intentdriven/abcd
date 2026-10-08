---
schema_version: 1
id: "iss-2610071538011774"
slug: "abcd-lab-preflight-s-dual-binary-gate-cannot-pass-in-a"
severity: "minor"
category: "bug"
source: "managed-repo"
found_during: "abcd inbox report rpt-2610071219191823 from a managed repository (root commit dd203ff780118159c16a4fbf48899c880e71a88e)"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd lab preflight (binary.work, binary.pinned); abcd lab mint next steps"
remedy: "none (filed automatically)"
resolution: "a lab of a repository that is not abcd's own passes the dual-binary group as not applicable, saying why, and mint names no bin/abcd there"
impact: fix
---

abcd lab preflight's dual-binary gate cannot pass in a managed repo that is not abcd, so every such lab halts

In a managed repository that is not abcd (Dessau, a Go app), `abcd lab mint`
succeeded for a product question about the repo's control panel.
`abcd lab preflight` then passed all four isolation checks (home, snapshot,
remotes, hooks) and failed the dual-binary group:

- binary.work: "bin/abcd is absent: build the work binary once from the
  pristine snapshot into bin/abcd"
- binary.pinned: "nothing to pin until bin/abcd passes"

The lab halted on gate finding F-1.

Expected: a lab of a managed repo can pass its preflight. The check wants
bin/abcd built from the snapshot, with an embedded vintage equal to the pin.
A snapshot of a repo that is not abcd has no cmd/abcd, so no such binary can
ever exist. The only ways round it break the gate's own rules: copying the
operator's abcd is "a link to an operator-level installation" in spirit and
fails the vintage check, and either would be adapting around a failed check,
which the lab forbids. So every lab of a non-abcd repo is halted at the first
step, with no remedy available to the operator.

To see it again: in any abcd-managed repo that is not abcd, run
`abcd lab mint "<question>"`, write INTENTION.md, then
`abcd lab preflight <lab-id>`.

The maintainer chose to let the lab halt rather than prototype outside it, so
the lab stays halted until abcd changes.

Remedy the reporter proposes: Report the dual-binary group as not applicable when the snapshot is not abcd's own repository (no cmd/abcd, or another module path), or let a managed repo name its own work binary in .abcd/config, held to the same rules (built once from the pristine snapshot, a regular file, never rebuilt); and stop lab mint's next steps telling every repo to build bin/abcd.

Reported by a managed repository (root commit dd203ff780118159c16a4fbf48899c880e71a88e) through the abcd inbox as rpt-2610071219191823, a defect against abcd v0.13.1, surface abcd lab preflight (binary.work, binary.pinned); abcd lab mint next steps.

Evidence:

- rpt-2610071219191823 (the report, kept in the inbox)
- lab-261007121616-f69bf1e (gate finding F-1)
- f69bf1ecd40395b419cad2ecf69d717acbf6b8c1 (the pin, a Dessau commit)
