---
schema_version: 1
id: "iss-2608301808193750"
slug: "the-blank-grounds-enumeration-claims-the-trailing-comment-is"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-189-delta-security"
found_at: "commands/capture.md"
resolution: "Both sentences are scoped to the spellings the predicate actually tests, and neither claims the enumeration is complete. commands/capture.md names the list as the set the gate reads rather than every way YAML can carry nothing, and says what still passes green: a trailing comment on the value, and spellings built from a tag, an anchor or an alias. The code comment beside isAbsentValue says the same at its own altitude -- every test there is a spelling test, not a null test, so the tagged and anchored nulls read as PRESENT. The predicate is deliberately NOT widened: that is iss-2608301808198621's altitude question, and an eleventh literal would leave a twelfth. TestIsAbsentValueIsASpellingTestNotANullTest pins the boundary both sentences now claim, in both directions, so a later widening cannot land while the prose still names which spellings pass."
impact: fix
resolved_by:
  intent: "itd-189"
---

the blank grounds enumeration claims the trailing comment is the last gap while eleven further nothing carrying spellings pass green

Found by the itd-189 delta security review. The CLAIM is branch-introduced; the
hole it mis-describes is pre-existing in class.

`commands/capture.md` tells the user that a YAML null or an empty block scalar is
refused, "with a trailing comment on the value the one spelling it does not yet
read". `schema.go` mirrors it with a "Two things it does NOT decide" list that
enumerates the comment gap and one other and stops. Both say the enumeration is
complete. It is not.

Measured, gate verdict against report verdict, one line per spelling:

```
grounds="!!null"                     gateGreen=false proposalSilenced=true
grounds="!!null null"                gateGreen=true  proposalSilenced=true
grounds="!<tag:yaml.org,2002:null>"  gateGreen=true  proposalSilenced=true
grounds="!!str ''"                   gateGreen=true  proposalSilenced=true
grounds="&anchor"                    gateGreen=true  proposalSilenced=true
grounds="!!seq []"                   gateGreen=true  proposalSilenced=true
```

At least eleven spellings carry nothing in YAML, pass the gate, and silence the
proposal -- which is verbatim the mechanism `iss-2608301649337965` was opened
for, reached by spellings outside that record's enumeration and outside the
already-captured `iss-2608301744268001`.

Note where the dishonesty is and is not. `iss-2608301649337965`'s own resolution
note is HONEST: it says it closes three of the four spellings the body
enumerates. The user-facing doc and the code comment are the two that overclaim.

Remedy for THIS record is the reword: scope both sentences to the spellings the
predicate actually tests. The gate is strictly better than it was either way;
what may not stand is a sentence telling a reader the hole is closed when it is
not. The widening is `iss-2608301808198621` and is deliberately separate.
