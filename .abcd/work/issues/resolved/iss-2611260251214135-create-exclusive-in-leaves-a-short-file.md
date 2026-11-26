---
schema_version: 1
id: "iss-2611260251214135"
slug: "create-exclusive-in-leaves-a-short-file"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/fsutil.go"
remedy: "write the content to a temp name in the same directory under the same os.Root, fsync it, hard-link it to the final name (link refuses an existing name, keeping the exclusive contract), remove the temp name and fsync the directory"
resolution: "CreateExclusiveIn now writes the content to a temp name in the same directory under the same os.Root, flushes it and hard-links it to the final name, which refuses an existing name; on a filesystem with no hard links it claims the name with an empty O_EXCL create and renames the whole temp file over it. The consistency reports and the route-return placement route through it"
impact: fix
---

fsutil.CreateExclusiveIn creates the final name with O_EXCL and only then writes the content, so a process killed between the create and the end of the write leaves a short or empty write-once record behind, and because the name now exists every later exclusive create refuses it: the record can never be rewritten
