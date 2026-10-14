---
schema_version: 1
id: "iss-2610111600044238"
slug: "outage-probe-runs-configured-transport"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-g5fq-6w8q-5jmg, filed 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/outage.go"
remedy: "Under abcd-fixes-what-it-adds this is not abcd's vulnerability: the same transport settings run on the person's own git fetch or ls-remote, and pinning them would change how the repository reaches its remote (a repository-config credential helper or ssh command is a legitimate setup), so the probe keeps the repository's transport; the prevention is the run tripwire of iss-2610111600058575, which covers the outage probe; close GHSA-g5fq as not a vulnerability in abcd and move this record to wontfix with that reason when the tripwire lands"
blocked_by: [iss-2610111600058575]
wontfix_reason: "Not abcd's vulnerability under abcd-fixes-what-it-adds: the same transport settings (a credential helper, an ssh command, a remote's upload-pack, a protocol allow rule) run on the person's own git fetch or ls-remote with no abcd involved, and pinning them for the probe alone would change how the repository reaches its remote while removing nothing an attacker can do. What abcd adds is the run tripwire of iss-2610111600058575: the outage probe, a step's and implement outage probe's, is stopped before it runs when those settings change during a run."
---

The implement loop's outage probe runs `git ls-remote --exit-code origin HEAD` with the isolated git pins only, and those pins do not set `protocol.ext.allow`, `credential.helper`, `core.sshCommand` or `remote.<name>.uploadpack`, so a program the checkout's git config names for any of them runs as the operator on the next due network probe.

Private security advisory GHSA-g5fq-6w8q-5jmg (draft, severity high). New in v0.13.4. Full text, evidence and reproduction: .abcd/.work.local/scratch/security-drain-2026-10-11/advisories/ in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (main 3d171421a): `RemoteProbe` (internal/core/implement/outage.go) calls `gitutil.RunLimitedContext` with only `-c core.askPass=false`; `ExecPins` (internal/gitutil/execpins.go) pins hooks, fsmonitor, signatures and diff.submodule and none of the four transport keys. The advisory reports, on Apple Git 2.39.5, that a later `-c remote.origin.uploadpack=` does not clear a configured value (git keeps the first), so a `-c` pin cannot close that one. Confirmed by reading the code at that commit; not re-run.

## Assessed under abcd-fixes-what-it-adds (2026-10-11)

Would it exist without abcd? Yes. A program named in the checkout's shared git settings (a content filter, a hook, a transport helper) runs on the person's own plain git commands with no abcd involved; tested 2026-10-11 by planting a filter and a hook from inside a lane and running a plain `git add` and `git commit` in the main checkout, and both ran. So it is not abcd's vulnerability, and switching those settings off for abcd alone would change how the repository works (large-file storage, its own hooks, its login and ssh helpers) while removing nothing an attacker can do. What abcd can do without changing anything is notice: the run tripwire stops abcd's next git step when those settings change during a run, and names the change. The earlier draft decision to blank filters and hooks in lane steps was withdrawn the same day.

## Grounds

- declined: Not abcd's vulnerability under abcd-fixes-what-it-adds: the same transport settings (a credential helper, an ssh command, a remote's upload-pack, a protocol allow rule) run on the person's own git fetch or ls-remote with no abcd involved, and pinning them for the probe alone would change how the repository reaches its remote while removing nothing an attacker can do. What abcd adds is the run tripwire of iss-2610111600058575: the outage probe, a step's and implement outage probe's, is stopped before it runs when those settings change during a run.
