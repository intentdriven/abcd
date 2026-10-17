---
id: dcn-2607130000000009
slug: privacy-hygiene-reads-tracked-files
date: 2026-07-13
---
2026-07-13 (itd-85 M3): privacy-hygiene reads tracked files through os.OpenRoot
(repo-root containment), not os.ReadFile. A leaf-only O_NOFOLLOW is insufficient
— a symlinked INTERMEDIATE directory still escapes; security review PoC-confirmed
an out-of-repo arbitrary read. os.Root refuses any escaping component. Plus
O_NONBLOCK (FIFO/device non-blocking open) + IsRegular skip + 4 MiB size cap.
Requires go 1.24+ (repo is 1.25); no new dependency.
