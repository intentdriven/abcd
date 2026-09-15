---
schema_version: 1
id: "iss-2608301308368922"
slug: "one-committed-symlink-at-the-admissions-root-stands-down-the"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "itd-189-round-2-security"
found_at: "internal/core/lint/readingoutstanding.go"
---

one committed symlink at the admissions root stands down the widening leg for every run while record_schema stays silent

Found by the round-2 adversarial security review of build/itd-189. A NEW
INSTANCE of a pre-existing, accepted pattern: the `dispositions` root behaves
identically on 932629f9.

Committing `.abcd/work/issues/admissions -> .adm-real` makes `realDir` (an
Lstat) return false, so `tree.rootUnreadable` is true, `unknown(run)` is true
for every run, and both the widening `Undispositioned` branch and the
`Unadmitted` leg break for every proposal in the repository. Reproduced: with
two unanswered proposals in different runs, output collapsed to one line and
exit 0.

Meanwhile the GATE's `os.ReadDir` FOLLOWS the symlink, and its parent-store
loop sees `admissions` as a non-.md entry and skips it -- so record_schema says
nothing.

Assessment, and why it is minor: this is documented intent (`rootUnreadable is
the whole-tree verdict`), the report is `info` by construction, and the Unsafe
line names the path. The report is loud about standing down. What is wrong is
the GATE's silence, and that is the same os.ReadFile/ReadGuarded asymmetry as
iss-2608301203521317 -- it closes with it.
