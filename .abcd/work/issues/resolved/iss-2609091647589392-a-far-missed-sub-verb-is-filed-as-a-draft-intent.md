---
schema_version: 1
id: "iss-2609091647589392"
slug: "a-far-missed-sub-verb-is-filed-as-a-draft-intent"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "release-gate"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
resolution: "The sub-verb guard now refuses on the SHAPE it already judged: a whitespace-free token followed by a record id is a subcommand call whatever its edit distance, so a far miss is refused at exit 2 with the registered sub-verbs listed instead of being filed as a draft title. Shared helper, so capture is fixed in the same change; the near-miss did-you-mean, the retired-spelling successor and the lone-token message are unchanged, and prose still files. Detectors in intent_surface_test.go and capture_surface_test.go, watched fail then pass; six mutants on a scratch copy all bite."
impact: fix
---

The intent verb takes quoted text as a create, and guards that path twice so a mistyped sub-verb is never swallowed as a title: a near-miss is refused with a did-you-mean, and a lone bare token is refused because a one-word positional is a sub-verb by shape. Neither guard catches a far miss. The typo check decides first whether the arguments are shaped like a subcommand call, and a token followed by a record id satisfies that shape, but it then refuses only when some registered sub-verb lies within an edit distance of two. A word that is nothing like plan or ready or link or audit therefore passes: the guard has already concluded the shape is a subcommand call, finds no suggestion to offer, and returns as though the input were prose, so the words are filed as the title of a new draft intent in the durable record tier. The failure is silent and it writes. A user who types a verb this tool does not have, next to the id they meant to act on, gets a minted record whose title is their own typo, and learns nothing about the mistake; the same shape reaches every surface that reads the intents directory afterwards. Fix direction: when the arguments are already judged subcommand-shaped, refuse whether or not a suggestion exists, and offer the registered sub-verbs when none is close enough to name. The capture verb's sibling guard is the model, and its comment describes exactly this shape. Detector: an unknown first token followed by a record id refuses with exit 2 and writes nothing, whether or not it is near a real sub-verb, while genuine prose of two or more words still files a draft.

## Grounds

- pursued: the shape decides the refusal, not the availability of a suggestion — what would show it wrong is either a durable record minted from a token-plus-record-id invocation, or a legitimate multi-word title refused
