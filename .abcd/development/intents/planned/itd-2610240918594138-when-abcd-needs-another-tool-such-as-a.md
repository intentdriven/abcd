---
id: itd-2610240918594138
slug: when-abcd-needs-another-tool-such-as-a
spec_id: spc-2610241008092356
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610240823055889]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# abcd installs the tools it needs for the person alone, checked

## Press Release

> When abcd needs another tool, such as a secret scanner, it installs it for the person alone, with no admin rights, into their own folder, and checks the download is the one the tool's makers published before it runs anything. A copy the person already has is used rather than duplicated, and setup says plainly how far each download could be checked.

## Why This Matters

_Split out of itd-2610240823055889 by the product thinker on 2026-10-24; drafted by the agent, not yet confirmed._

On 2026-10-24 setup offered to install a secret scanner through Homebrew on an account that cannot write to Homebrew, failed on every run, and passed on Homebrew's advice to use admin rights (iss-2610240824571059). Research the same day found that a supported Homebrew for one non-admin account cannot be created on a Mac by a tool, so the product thinker chose a checked download into the person's own folder as the default for every tool (dcn-2610240845548776).

## Mechanism

- We expect what lands in the person's folder to be what abcd shipped, with no admin rights needed, because every download is named and checked in abcd's own release.


## Scope Conditions

- macOS and Linux, on Apple and Intel processors (kept because abcd already ships an Intel Mac build and each tool needs only one more download entry; the product thinker asked for Intel only if it costs no extra effort). <!-- cond: cond-2610241008093719 -->
- An account with or without admin rights. <!-- cond: cond-2610241008096511 -->
- Tools whose makers publish releases with checksums. <!-- cond: cond-2610241008090433 -->


## Acceptance Criteria

_Each criterion confirmed by the product thinker, one at a time, on 2026-10-24; the third with the product thinker's change that an existing copy is used only when it is compatible._

- Given a tool abcd needs is missing, when setup installs it, then it downloads the version fixed in abcd's release into the person's own folder, checks it before writing anything, needs no admin rights, and never shows a step that needs admin rights.
- Given a download fails its check, when setup installs it, then nothing is written and setup says why; and given a tool that publishes no signed build record, when setup installs it, then setup says it was checked by checksum only.
- Given the tool is already installed, when setup runs, then it uses that copy and installs nothing only when the copy is compatible with what abcd needs; otherwise it offers abcd's own copy and says which of the two will run; it never changes the other copy.
- Given abcd installed a tool, when abcd runs it, then abcd runs that copy whatever else comes first on the PATH; and when abcd is removed, then the copies it installed are removed and nothing else.

## Open Questions

- resolved: Decision 2026-10-24 on tool installs meets brief invariant 7 ("provisioning fetches only a manifest-named, pinned, checksum-verified artifact. No version-discovery request exists anywhere in abcd") only if abcd fixes each tool's version and checksum in its own release; and a tool that publishes no signed build record, such as gitleaks, is checksum-checked only. Is the decision read that way? (raised 2026-10-24 by the design review) (answered 2026-10-24; see Decisions)

## Decisions

- 2026-10-24, the product thinker: setup works on an account without admin rights. Every tool it installs, on macOS and Linux alike, is a verified download of the tool's official release into `~/.local/bin` (checked against the published checksums and the signed build record where one exists), the route abcd already uses for itself; Homebrew is not the default (dcn-2610240845548776).
- 2026-10-24, the product thinker: when a tool setup needs is already installed some other way (by Homebrew, a package manager, or by hand), setup uses it when it runs and is recent enough, and installs no second copy; when it is too old, setup offers abcd's own copy in `~/.local/bin` and says which of the two will run, since the first on the PATH wins; setup never removes or changes the other copy.
- 2026-10-24, the product thinker: each tool's version and checksum are fixed in abcd's own release, as brief invariant 7 requires, so a tool updates only when abcd does; a tool that publishes no signed build record, such as gitleaks, is checksum-checked only, and setup says so.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect a second contributor and an account without admin rights to set abcd up with no surprises; it is shown wrong if either still hits a step they cannot do.
