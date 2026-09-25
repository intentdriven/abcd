---
schema_version: 1
id: "iss-2609250955219864"
slug: "internal-core-ahoy-firstoutoffenceh1-marker-go-which-picks"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/marker.go"
resolution: "firstOutOfFenceH1 reads live lines through mdrecord.Mask; the marker block lands after the real H1 when a longer fence quotes a shorter one, a tilde line sits in a backtick fence, or a heading is parked in a comment."
impact: fix
resolved_by:
  commit: "e58602ff6c6547109d28568d10c56a39f609820c"
---

internal/core/ahoy firstOutOfFenceH1 (marker.go), which picks where the identity marker block is inserted in a file with no frontmatter, keeps a private fence toggle that diverges from mdrecord.Mask. It flips on any line starting with three backticks or three tildes, whatever the marker, run length or info string, and it cannot see an HTML comment. So a three-backtick line quoted inside a four-backtick fence closes the fence early. Probe at 70daf701: in a README opening with a four-backtick markdown example that quotes a three-backtick sh block holding '# not the title', the function returns the offset after '# not the title'. The marker block is then written inside the fenced example, above the real '# Real Title'. That is a write into the operator's file at the wrong place. A '# ' heading parked in a comment is also taken as the H1.

## Grounds

- pursued: the block is placed after the first live H1 under CommonMark; TestMarkerInsertFollowsTheCommonMarkFenceRule shows it, and a fixture whose example is split or whose block lands above the real title would show it wrong
