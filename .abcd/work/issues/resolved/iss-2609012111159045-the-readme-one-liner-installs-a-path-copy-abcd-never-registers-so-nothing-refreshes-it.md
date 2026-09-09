---
schema_version: 1
id: "iss-2609012111159045"
slug: "the-readme-one-liner-installs-a-path-copy-abcd-never-registers-so-nothing-refreshes-it"
severity: "major"
category: "drift"
source: "agent-finding"
found_during: "ship-audit-itd-130-itd-132-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/bootstrap.sh"
related_issues: ["iss-2609012111168716"]
resolution: "the install one-liners register the PATH copy (c637a734), so the bootstrap refreshes it and ahoy owns it, and abcd update now proves that copy from its path-entry record rather than from a manifest the forge may have deleted"
impact: fix
---

A PATH copy installed by the README one-liner (install -m 0755 into ~/.local/bin) writes no ~/.abcd/path-entry provenance record. The bootstrap refreshes only a registered copy that still matches its recorded hash, so a one-liner copy is never refreshed on plugin update; ahoy classifies it foreign; RefreshPathEntryDigest 'refreshes provenance, never creates it'; and abcd update can only vouch for it while its release manifest is still published. itd-132's press release promises 'the abcd command on PATH keeps working across any number of updates' as 'a regular file abcd owns and refreshes', but its ac-4 Given is ahoy install, so the one-liner route is outside what was delivered (audit receipt rcp-acde3e9ce729, diverged and missing items). Directions, none adopted: the one-liner registers the entry (a post-install 'abcd ahoy adopt', or the binary self-registering on first run of an unregistered regular-file PATH copy after proving it against the release manifest); or abcd update creates the record when it proves a file; or the docs stop promising refresh for that route. Part of the ease-of-update design work.

Amended 2026-09-09 after checking every clause against the tree rather than
against another agent's remark. Three are answered by the hook PATH-rung change
(c637a734): all three install one-liners now write `path=` and `binary_sha256=`
into ~/.abcd/path-entry, so the copy IS registered; hooks/bootstrap.sh's refresh
block is gated only on those two fields plus the file still matching its
recorded hash, so a one-liner copy is refreshed on the next new release —
pinned now by TestBootstrapRefreshesAOneLinerInstalledPathCopy, whose fixture
record shape is read off the shipped one-liner so a change to it fails there
rather than silently retiring the route; and classifyBinTarget reaches
binTargetOwnedCopy through isOwnedCopyFile, so `ahoy` no longer calls that copy
foreign. The fourth clause — `abcd update` can vouch for the copy only while its
release manifest is still published — is what this change fixes: path-entry is
consulted as an ownership proof BEFORE the swap, not merely re-stamped after it
(the itd-130 audit's ac-1 concern (b)). The ac-4 divergence is retired from both
ends: concern (a), the cache being invisible to a terminal `ahoy install`, was
closed by the plugin root's .data-dir stamp under iss-2609012111168716, and
concern (b), the one-liner sitting outside ac-4's Given, is what the one-liner
change delivers. One direction is deliberately NOT taken: RefreshPathEntryDigest
still only refreshes, so a PATH copy installed by a one-liner from before
c637a734 carries no record and an update does not adopt it. That is the pre-1.0
population the maintainer ruled out of scope; re-running the one-liner registers
it, and the install guide says so.

## Grounds

- pursued: registering the one-liner copy makes every install route refreshable by the same two mechanisms; a one-liner install that ahoy still calls foreign, or that a new release leaves stale, or that abcd update refuses once its release objects are gone, would show it wrong
