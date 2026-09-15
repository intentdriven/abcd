---
schema_version: 1
id: "iss-2608301620346560"
slug: "the-scriptio-continua-refusal-names-a-three-unit-floor-while"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-179-round-5-orchestrator"
found_at: "internal/core/grounds/grounds.go"
resolution: "The no-word-breaks refusal now names both floors in the unit each is measured in."
impact: fix
resolved_by:
  intent: "itd-179"
---

the scriptio-continua refusal names a three-unit floor while the letter floor asks twenty so satisfying the message earns a second refusal

Found by the orchestrator reviewing the round-5 letter-floor fix (a46f37e3),
not by the builder. Branch-introduced: `internal/core/grounds/` exists on
neither `main` nor `experiment/cold-reading`, so the whole package is this
intent's own code and the stopping rule fixes rather than merges it.

`ValidateText` asks two floors in order. The unit floor speaks first, and when
the units are all scriptio-continua letters it says:

```
grounds text "字文" carries 2 letter(s), and the floor asks for 3 where the
script has no word breaks
```

The author adds one ideograph and is refused a second time, by a floor the
first message never mentioned:

```
grounds text "字文書" carries 3 letter(s), below the 20-letter floor
```

Both messages count in the same noun. So the author sees the count go 2 -> 3,
watches the requirement go 3 -> 20, and is told nothing about why. The first
message is not wrong about its own floor; it is wrong that satisfying it gets
the text through, which is the only reason a refusal states a number at all.

The gap is widest exactly where the round-4 work aimed: for a script with no
word breaks every unit is one letter, so the twenty-letter floor is the binding
one and the three-unit floor it names never binds alone. A Latin author reading
"below the 3-word floor" is told the truth, because three Latin words usually
carry twenty letters. A CJK author reading "asks for 3" is told a number that
cannot be enough.

Class: this is the cycle's standing class, a message asserting a requirement
that is not the case, reached this time by stating a real floor while omitting
the one that actually binds. The remedy is for the scriptio-continua refusal to
name both floors, in the unit each is measured in.

## Grounds

- pursued: We expect naming both floors to stop the second refusal, because the unit floor never binds for a script whose every unit is one letter, so an author supplied the only number the message gave and was refused again by a floor it had not mentioned.
