---
schema_version: 1
id: "iss-2611260523161723"
slug: "a-drive-s-let-go-releases-holds-a-new"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "dcn-2611260036312398 follow-up, surge lane"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/terminal.go"
remedy: "Have letGo recognise a judgement or forming hold a run of this checkout was opened on (its reserved_at is the hold's claimed_at), record that lane in the drain state and leave its holds, write the drain state before any release it pushes, and have the move's finish write the drain state before it settles any hold over the network."
resolution: "A drive's let-go now recognises a lane a killed move opened on the drain's own hold and never recorded, records that lane and keeps its holds, and writes the drain state before it pushes any release; a move's finish writes the drain state before it settles any hold over the network."
impact: fix
---

A drain move that dies after it writes a new lane's run and before it writes the drain state leaves that state recording the judgement or forming holds the run now carries, and a drive's let-go that runs before the next move releases them plainly, so the run holds a released reservation; the move also releases unused holds over the network before it writes the state, which widens that window.
