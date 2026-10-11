---
schema_version: 1
id: "iss-2608301237450573"
slug: "pre-existing-on-the-itd-183-branch-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-183-round-9-security"
found_at: "internal/core/reading/project.go"
resolution: "The floor refuses the nesting behind every block indicator, not one spelling of it: the run of sequence and explicit-value indicators is read off and a tag, anchor, explicit key or compact mapping after it is refused whatever the key is named, and the flow scan reads a key after an opening bracket. The recorded shape and its siblings (a second indicator, a node property, an explicit key in the entry, a compact explicit value, a flow pair in a flow sequence) are refused; rawHTMLHeading's comment states only what the fence does there."
impact: fix
resolved_by:
  commit: "9088559086d6d4b5334a336257bd926c45d8b409"
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

## Grounds

- pursued: every compact nested mapping a frontmatter line can open behind block indicators refuses the assembly, while committed flow-mapping history rows and scalar sequences are admitted; an excluded key admitted in any such spelling, or a committed record refused by the new rule, would show it wrong
