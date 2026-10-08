---
schema_version: 1
id: "iss-2609290448510918"
slug: "predecessor-qualifier-sweep-past-the-six-named-intents"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents"
deferred_after: "v0.11.1"
deferral_reason: "no ruling owed; carried past v0.11.1 by lane drainDrift3 (run A 2026-09-29) on its size: the 192 sites each need a reading against the live spec their id collides with, which the lane's time box did not hold after qualifying the six intents iss-2609261536147903 named."
remedy: "Read each remaining site against the live spec its id collides with and qualify the predecessor-store ones '(predecessor store)' per the specs charter's Two spc-N Namespaces rule, after the branches carrying itd-82 and itd-130 land (the five sites named in the progress section below); grounds: the charter is the rule, and the census below (ordinal ids at or below spc-70, not the intent's own spec, no qualifier on the line) reproduces the set."
---

The predecessor-store qualifier sweep reaches past the six intents iss-2609261536147903 named: outside drafts/ and those six, 192 citation sites across about forty intents name a spc-N at or below spc-70 that is not the citing intent's own spec, with no '(predecessor store)' on the line. Some are live cross-references (the cold-reading family's intents cite each other's live specs), and some are the predecessor store's (itd-4, itd-6, itd-29, itd-47 and itd-49 describe pre-rebuild work in its terms), so each site needs the same reading against the live spec it collides with; the specs charter's Two spc-N Namespaces section is the rule. Found while sweeping the pattern of iss-2609261536147903 in lane drainDrift3; a census script over intents/{planned,shipped,disciplines} reproduces the count.

## Progress 2026-09-30 (lane drainCitations, run A)

A census at 2b9d52fbb over `intents/planned`, `intents/shipped`, `intents/disciplines` and `intents/superseded` (an ordinal `spc-N` at or below spc-70, not the citing intent's own spec, on a line without the qualifier) counts 186 sites across 42 intents: 125 outside superseded/ and 61 inside it, where itd-29, itd-47 and itd-49 now sit. The first count of 192 was not scripted in the record, so the six-site difference is not traced. Every site was read against the live spec it collides with.

- Predecessor store, now qualified (94 sites, 12 intents): itd-4 (spc-20 to spc-23), itd-6 (spc-2, spc-4, spc-5), itd-36 (spc-38, spc-39), itd-50 (spc-52), itd-65 and itd-66 (spc-64, spc-27), and superseded itd-17, itd-20, itd-27, itd-29, itd-47 and itd-49. Two shipped acceptance criteria gained the qualifier, itd-4's drift criterion and itd-65's fail-closed criterion; each names an owner or a precedent, not a promise, so both are wording clarifications. The Implementing specs sections of itd-36, itd-4 and itd-47 also misstated where those ids stand, captured and fixed as iss-2609300025372991 and iss-2609300032219282.
- Live cross-references, correct as written (72 sites): the cold-reading family's citations of spc-55 to spc-69 (itd-177 to itd-189, itd-2609020625400194, itd-2609020625400445), itd-4's audit notes on spc-24, and itd-3, itd-80, itd-94, itd-101, itd-121, itd-132, itd-133, itd-147, itd-160, itd-161, itd-2609091416295622, itd-2609111003026787 and itd-2609231013154443.
- Data, left as written (15 sites): the `routed_from` frontmatter of itd-48, itd-50 and itd-53; itd-27's dated reclassification_history reason; fixture strings quoted as evidence in itd-4, itd-28, itd-186 and itd-2609111003026787; the example resolution note inside itd-4's resolve criterion; and itd-4's audit-note line that already says "of the retired record system".

Skipped, because other branches carry these intents: itd-82 (one site, spc-24) and itd-130 (four sites, spc-35). Read in place, all five cite live specs (itd-119's promote and itd-132's data directory), so none looks owed a qualifier. The other intents named for skipping (itd-111, itd-148, itd-24, itd-103, itd-2609081951381895, itd-2609211116005482, itd-2609212103568351 and itd-2609212103572513) hold no site in the census. The record stays open until those five sites are confirmed at the merged tip.

## Progress 2026-10-08 (lane-1, run-2610080034282884)

Both branches have landed: itd-82 sits in `intents/planned` and itd-130 in `intents/shipped` at 79ac82c17. The same census over the same four folders now counts 102 sites across 36 intents, and every one was read against the live spec its id collides with. None is a predecessor-store reference without its qualifier, so no line changes.

- The five deferred sites are live cross-references, correct as written: itd-82's one spc-24 is itd-119's `capture promote`, and itd-130's four spc-35 are itd-132's download cache and owned PATH copy.
- New to the census: itd-33 entered `intents/superseded` from `drafts/` on 2026-10-03 (84bfc4bb1), bringing 4 sites. Its spc-33 is the spec a promotion of itd-33 would mint in its own press-release quote, and its spc-7 illustrates a task-granular claim (`spc-7.2`, `spc-7.3`). Both are illustrations, not references to either store's record, so they are left as written like itd-4's example resolution note.
- Not on the 2026-09-30 lists: itd-36's 2 are the "live spc-38 ... live spc-39" contrast on line 92, text 542737b21 wrote after 2b9d52fbb, which says live in words.
- Counted on 2026-09-30 as qualified, and back in the census: itd-29's 4 (lines 21, 57 and 136, the suffixed legacy-roadmap ids `spc-29-42i` and `spc-9-kbe`) are anchor sites, among the 94 the 2026-09-30 list calls qualified. d38139599 gave them "(predecessor store)"; the review follow-up 01f3f0c19 replaced that with "legacy roadmap", because the intent itself says those ids were never in a store, so the census, matching only their prefix and finding no qualifier, counts them again. They reference no record in either store and are left as written. They are the 4-site gap between the 94 the 2026-09-30 list counts as qualified and the 90 anchor sites of its 12 intents that left the census: 186 - 90 + 4 (itd-33) + 2 (itd-36) = 102.
- The other 87 sites are the live cross-references and data sites the 2026-09-30 lists already classify, among them itd-132's 11, which cite live spc-21 (itd-105's hook-binary fetch, whose verification posture itd-132 keeps) on the same lines as at 2b9d52fbb; read again at the tip, each keeps its classification.

The census, run from the repository root:

```python
import glob, re
owner = {}
for p in glob.glob(".abcd/development/specs/*/spc-*.md"):
    head = open(p).read()[:400]
    s, i = re.search(r"^id:\s*\"?(spc-\d+)", head, re.M), re.search(r"^intent:\s*\"?(itd-\d+)", head, re.M)
    if s: owner[s.group(1)] = i and i.group(1)
sites = []
for d in ("planned", "shipped", "disciplines", "superseded"):
    for p in sorted(glob.glob(f".abcd/development/intents/{d}/itd-*.md")):
        iid = re.match(r".*/(itd-\d+)", p).group(1)
        for n, line in enumerate(open(p), 1):
            if "predecessor store" in line: continue
            for m in re.finditer(r"(?<![\w-])spc-(\d+)(?!\d)", line):
                if int(m.group(1)) <= 70 and owner.get(m.group(0)) != iid:
                    sites.append((iid, n, m.group(0)))
print(len(sites), len({s[0] for s in sites}))
```
