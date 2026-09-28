---
schema_version: 1
id: "iss-2608301646046226"
slug: "grounds-text-the-floor-accepts-can-be-refused-by-the-site-re"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-179-round-5-security"
found_at: "internal/core/intent/grounds.go"
resolution: "grounds.New and grounds.NewDerived refuse, with nothing written, a bullet the site renderer refuses, asking the renderer itself through one predicate (mdrender.RefusalIn, the renderer moved to a leaf so the grounds package can import it), strict on reference links and images. TestNewRefusesWhatTheSiteRendererRefuses covers the five reviewed constructs and five siblings from the renderer's refusal list; TestNewDerivedRefusesWhatTheSiteRendererRefuses covers the wontfix path. All 645 committed grounds entries pass, so none is stranded."
impact: fix
resolved_by:
  commit: "cce6eb9f"
---

grounds text the floor accepts can be refused by the site renderer and the entry is append only so no verb can remove it

Found by the round-5 security review, which recommended it be filed because no
record covered it.

Grounds text that clears the substance floor and the readability guard can
still be refused by the site renderer. The reviewer verified five constructs
accepted by `grounds.New` plus `RecordGrounds` and refused by `abcd site
build`: an unclosed code span, a remote image, a raw `<div>`, an undefined
reference link and a footnote link. The natural one is the first, and it needs
no malice at all:

```
pursued: we expect the `--grounds flag to make the conjecture survive the session
```

One unbalanced backtick in a sentence naming a flag. `site-render` is one of
the six `make preflight` gates, a CI step and a release gate, so the record
lands and the build goes red.

The class is PRE-EXISTING: the same unbalanced backtick in an ordinary issue
body does the same thing today. Two things make the grounds path worse rather
than equal. Recording grounds is now MANDATORY, because `groundsCheck` refuses
a planned record without an entry. And the ledger is APPEND-ONLY by design,
with no verb that removes an entry, so where a body can be re-edited by a
command this residue can only be removed by hand-editing the committed record.

The readability guard's own doc is silent about this: it says a bullet that
fails to raise the count is refused, which is true of `ParseGrounds` and says
nothing about the second reader that runs in the same preflight.

## Grounds

- pursued: no grounds entry a verb can write makes site build refuse the record it lands in; a grounds text that New accepts and site-render then refuses would show it wrong
