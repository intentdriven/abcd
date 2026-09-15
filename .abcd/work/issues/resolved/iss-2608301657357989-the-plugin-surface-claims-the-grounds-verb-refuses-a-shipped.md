---
schema_version: 1
id: "iss-2608301657357989"
slug: "the-plugin-surface-claims-the-grounds-verb-refuses-a-shipped"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-179-round-5-ruthless"
found_at: "commands/intent.md"
resolution: "The surface claim and the RecordGrounds doc now say what the refusal establishes — that no grounds this tool writes can land on a terminal record — instead of asserting the corpus carries none, and both name the migration exception: itd-177, itd-182 and itd-188 carry a section relocated by hand from the pre-tooling `## Grounds (pursued)` section on the matching spec. A DECISIONS entry draws the relocation-versus-backfill line. The enforcement is unchanged and stays maximally strict, because nothing in it can tell relocated text from invented text."
impact: internal
resolved_by:
  intent: "itd-179"
---

the plugin surface claims the grounds verb refuses a shipped record so no terminal bucket carries grounds while three shipped intents do

Found by the round-5 ruthless review. Branch-introduced, and it is the standing
class pointed at the corpus rather than at a mechanism.

`commands/intent.md`, and the same claim in `RecordGrounds`'s doc, tells the
host that the verb refuses a `shipped/` or `superseded/` record, "so the
exemption stays a true statement about the corpus". It is not true of this
corpus. Three shipped intents carry a `## Grounds` section -- itd-177, itd-182
and itd-188 -- hand-written by this branch's own migration commit.

The sharper half: the tree's state is unreachable through the tool that ships
it. `abcd intent ready itd-177 --grounds "..."` exits 2, so nobody can
reproduce or extend the migration through the documented path.

A defence exists and the reviewer stated it: relocating a pre-tooling
`## Grounds (pursued)` section from its spec is arguably not a BACKFILL, since
the text was authored at pursuit time and nothing was reconstructed. That
defence is credible. What it is not is written down anywhere -- the sentence is
unconditional about the corpus, DECISIONS.md records the floor rulings and says
nothing about migrating into terminal buckets, and the enforcement cannot tell
relocation from invention.

Remedy: one sentence naming the migration exception, in the surface claim and
in the doc, or a DECISIONS entry that draws the relocation/backfill line. Not a
silent contradiction.

## Grounds

- pursued: we expect a claim about the tool to survive where a claim about the corpus cannot, because the corpus changes under a hand-written migration while the refusal does not, and a fourth terminal record acquiring a section without falsifying the sentence is what would show it
