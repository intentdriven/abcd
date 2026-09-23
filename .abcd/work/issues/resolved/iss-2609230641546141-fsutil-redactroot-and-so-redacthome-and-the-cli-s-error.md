---
schema_version: 1
id: "iss-2609230641546141"
slug: "fsutil-redactroot-and-so-redacthome-and-the-cli-s-error"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/paths.go"
resolution: "RedactRoot redacts a root only where it starts a path (start of string, or after a non-segment character or a separator); RedactHome redacts both the environment's and the symlink-resolved home spelling, and the peers surface's local workaround is removed."
impact: fix
resolved_by:
  commit: "d80383c6"
---

fsutil.RedactRoot (and so RedactHome and the CLI's error scrub) replaces a root that ends at a path boundary but never checks the boundary BEFORE it, so a root that appears as a suffix of a longer absolute path is redacted inside it: with HOME=/var/folders/x/T/home and git reporting the resolved /private/var/folders/x/T/home/wt/a, RedactHome returns /private~/wt/a — the prefix stranded in front of the tilde, and the path neither redacted nor intact. Seen while building the peer listing on macOS, whose temp dirs sit behind the /var -> /private/var symlink; the peers surface works around it by redacting the resolved home spelling first (internal/surface/cli/peers.go redactHomePath). Expected: a match that does not start at the string's start or after a non-path character is left alone, the way the right-hand boundary already is.

## Grounds

- pursued: a home path inside a longer path (HOME /var/... within /private/var/...) is left intact while a real prefix still redacts, pinned by TestRedactRootRequiresABoundaryBeforeTheRoot and TestRedactHomeRedactsBothSpellingsOfASymlinkedHome; a rendered '/private~/' or an unredacted resolved home in abcd peers output would show it wrong
