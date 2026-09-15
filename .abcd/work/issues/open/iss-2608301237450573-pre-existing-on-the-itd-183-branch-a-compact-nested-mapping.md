---
schema_version: 1
id: "iss-2608301237450573"
slug: "pre-existing-on-the-itd-183-branch-a-compact-nested-mapping"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-183-round-9-security"
found_at: "internal/core/reading/project.go"
---

pre-existing on the itd-183 branch: a compact nested mapping in a block sequence leaks an excluded key, and rawHTMLHeading's fence comment overclaims

Found by the round-9 adversarial security review of build/itd-183, and recorded
rather than chased: identical on HEAD and on the parent 044ac6ed. The package
does not exist on main, so nothing here is inherited from main.

1. A compact nested mapping in a block sequence leaks. In frontmatter,

```
   items:
     - origin: <value>
```

   is a real `origin` key to YAML, but `excludedKeyLineRe`'s `^\s*` cannot
   cross the `- `, and `flowKeyRe` needs a `{` or a `,`. The reviewer notes
   this shape is MORE LIKELY TO BE TYPED BY ACCIDENT than any of the round-9
   regressions -- which makes it the most probable real-world leak on the
   branch, despite being the least exotic.

2. `rawHTMLHeading`'s doc comment claims "a fenced line is replaced by an empty
   line rather than dropped". The code joins all lines unmodified and tests
   `fenced[line]` only for the OPENER, so a fenced region can still supply a
   bound for an unfenced heading (and, per iss-2608301237458660, a mask
   opener). The comment describes a behaviour the code does not have.

Both are pre-existing on the branch and are left open for the facilitator.
Item 1 in particular deserves a decision: it is a genuine hole in the exclusion
floor that no round has closed.
