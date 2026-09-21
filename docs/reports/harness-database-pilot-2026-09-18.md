# Harness database pilot — Aetox vs Codex (2026-09-18)

Status: **Pilot สำเร็จ · ยังห้ามใช้เป็นผลเปรียบเทียบทั่วไป**

รอบนี้พิสูจน์ว่า protocol (บันทึกภายใน ไม่อยู่ในรีโปสาธารณะ — สาระคือ 5 รูปทรงงาน × 3 รอบ
× 2 ฝั่งต่อคู่ โมเดลเดียวกัน prompt เดียวกัน ตัดสินด้วย hidden tests ที่ผู้ทำไม่เห็น)
มี fixture, hidden scorer และ runner ที่ใช้ได้จริงกับคู่ Aetox/Codex แล้ว ไม่ได้พิสูจน์ว่า
harness ใดดีกว่าโดยรวม เพราะมีเพียงหนึ่งงาน แม้รันครบสามรอบต่อฝั่งแล้ว ขณะที่เกณฑ์เผยแพร่เดิม
กำหนด 5 งาน × 3 รอบ × 2 ฝั่งต่อคู่ รวม 120 รันทั้งชุด

## สิ่งที่ตรึงก่อน scored run

| รายการ | ค่า |
|---|---|
| Source / runner / fixture commit | `0b7168ee890a46d40b72439c0542162ddca3d30c` |
| Task | `sqlite-email-migration` |
| Prompt | ไฟล์เดียวกันแบบ byte-for-byte |
| Model | `gpt-5.6-luna` |
| Reasoning | `low` |
| Account class | ChatGPT account เดียวกัน |
| Follow-up | ไม่มี |
| Timeout | 20 นาทีต่อรัน |
| Aetox | 1.7.4, coding desk, bundled skills, fresh data root, full-access |
| Codex | 0.155.0-alpha.2.6, ephemeral, user config/rules ปิด, danger-full-access |
| Runtime | Python 3.13.15 · Go 1.27.0 · Windows/amd64 |

Fixture และวิธีรันอยู่ที่
[`benchmarks/harness/README.md`](../../benchmarks/harness/README.md) และ driver อยู่ที่
[`scripts/harness-bench.ps1`](../../scripts/harness-bench.ps1)

## งานและตัวตัดสิน

โปรเจกต์ตั้งต้นเป็น Python stdlib + SQLite มี migration 001 และ data-access functions
ที่ยังเทียบอีเมลแบบตรงตัว ผู้ทำต้องเพิ่ม migration ใหม่โดยไม่แก้ history, เก็บอีเมลเดิม,
สร้าง canonical key, บังคับ uniqueness ในฐานข้อมูล, รองรับ fresh/upgrade/rerun และ rollback
ทั้ง schema กับ migration record เมื่อ legacy rows ชนกัน

Hidden scorer มี 6 tests:

1. fresh install ได้ migration chain และ schema ครบ;
2. upgrade รักษา display email และ backfill key;
3. canonical duplicate ถูก unique constraint ปฏิเสธ;
4. lookup ไม่สน whitespace/case;
5. migrate ซ้ำได้;
6. legacy collision rollback schema, history และข้อมูลครบถ้วน

## Scored result

ใช้ `r8`, `r9` และ `r10` เป็น scored set เพราะทั้งหกรันใช้ hash, adapter, prompt และ scorer
เดียวกัน:

| Metric | Aetox | Codex |
|---|---:|---:|
| Hidden tests, r8/r9/r10 | **6/6 · 6/6 · 6/6** | **6/6 · 6/6 · 6/6** |
| Hidden assertions total | **18/18** | **18/18** |
| Harness/scorer exits | 0/0 ทุกครั้ง | 0/0 ทุกครั้ง |
| Wall time, r8/r9/r10 | 156.346 · 127.274 · 90.808 s | **66.925 · 56.745 · 74.558 s** |
| **Median wall time** | 127.274 s | **66.925 s** |
| Files changed | 2 ทุกครั้ง | 2 ทุกครั้ง |
| Median diff size | +24 / −10 | **+20 / −5** |
| Longest line | 81 ทุกครั้ง | 81 ทุกครั้ง |
| Median final `app.py` | 74 lines | **71 lines** |
| Median migration file | **5 lines** | 7 lines |
| Asked user / timeout | 0 / 0 | 0 / 0 |

ผลที่พูดได้จากรอบนี้:

- **Correctness เสมอกันทั้งสามรอบ:** ทั้งสองผ่าน behavior ที่ซ่อนทั้งหมด รวม atomic
  collision rollback ซึ่งเป็นส่วนยากที่สุดของโจทย์
- **Codex median เร็วกว่า 60.349 วินาที** และเร็วกว่า Aetox ในทั้งสามรอบของชุดนี้
- Codex มี median diff เล็กกว่า; longest line เท่ากัน และทั้งสองรักษาขอบเขตไฟล์เท่ากัน
- ทั้งสองแตะเฉพาะ `app.py` และ migration ใหม่ ไม่แก้ migration 001 หรือ visible tests
- Aetox ใช้ 24/19/18 tool rounds (median 19), input median 365,127 tokens,
  cached median 273,664 และ output median 3,732
- `aetox-database` ถูกเปิดใน **1 จาก 3 scored runs** (r9); r8/r10 ผ่านทั้งหมดโดยไม่เปิด
  สกิลฐานข้อมูล นี่เป็น reach signal ว่าการเลือกสกิลยังไม่สม่ำเสมอ และเป็นสิ่งที่
  ต้องวัดต่อ ไม่ใช่ซ่อนเพราะคะแนนผ่าน
- Codex รายงาน aggregate tokens 23,577 / 17,759 / 24,664 (median 23,577); telemetry
  คนละรูปแบบ จึง **ห้ามเอาตัวเลข
  token สองฝั่งมาหารหรือประกาศว่าฝั่งใดประหยัดกว่า**

ผลที่ยังพูดไม่ได้:

- ห้ามสรุปว่า Aetox หรือ Codex ดีกว่าโดยรวม;
- สามรอบของงานเดียวบอก distribution ของงานนี้ได้เพียงเบื้องต้น ไม่แทนงานรูปทรงอื่น;
- ห้ามย้ายตัวเลขนี้ขึ้น `BENCHMARK.md` หรือหน้าเผยแพร่;
- ยังไม่มีผล OpenCode/Crush/Aider และยังขาดอีกสี่ task shapes

## Database skill routing tune

หลัง scored pilot พบว่า `aetox-database` ถูกเปิดเพียง 1/3 รอบ จึงจูนการเลือกสกิลของโต๊ะโค้ด
โดยไม่เพิ่มตัวเดา intent จาก keyword/path (วิธีจูนอยู่ในบันทึกภายใน) แล้ววัดซ้ำด้วยงานเดิม
scorer เดิม:

รอบยืนยันสุดท้าย `r16`–`r18` ใช้ source commit
`c9f7054b7a769721f972bcc7dd34e753be0c25d6`:

| Metric | Baseline r8–r10 | Tuned r16–r18 |
|---|---:|---:|
| Hidden assertions | **18/18** | **18/18** |
| Opened `aetox-database` before code work | 1/3 | **3/3** |
| Unneeded database reference reads | ไม่สม่ำเสมอ | **0/3 runs** |
| Unneeded `aetox-brainstorm` reads | 0/3 | **0/3** |
| Median wall time | 127.274 s | **116.864 s** |
| Median rounds | **19** | 20 |
| Median tool calls | **18** | 19 |
| Median input / cached / output tokens | 365,127 / 273,664 / 3,732 | 370,312 / 207,360 / 3,116 |

Routing ดีขึ้นจาก 1/3 เป็น 3/3 โดยแลกหนึ่ง skill read ต่อรอบ ส่วนเวลาลด 10.410 วินาที
แต่ตัวอย่างมีเพียงสามรอบของงานเดียว จึงถือเป็น regression signal ไม่ใช่ข้อสรุปว่าความเร็วดีขึ้นทั่วไป

รอบระหว่างจูนเก็บไว้เป็นหลักฐานแต่ไม่รวมในตารางสุดท้าย: `r11` แสดงว่าการทำ generic prompt
ให้แรงขึ้นอย่างเดียวยังไม่ route database skill; `r12` route ถูกแต่บังคับอ่าน reference สี่ไฟล์;
`r13`–`r15` ตัด reference ส่วนเกินได้แต่เปิด brainstorm เกินขอบเขตทุกครั้ง

## Raw artifacts

เก็บใน working tree ของเครื่องนี้ ไม่อยู่ในคอมมิต:

- `output/harness-bench/aetox-sqlite-email-migration-r8/` ถึง `r10/`
- `output/harness-bench/codex-sqlite-email-migration-r8/` ถึง `r10/`
- `output/harness-bench/aetox-sqlite-email-migration-r11/` ถึง `r18/` สำหรับ routing tune

แต่ละโฟลเดอร์มี `result.json`, transcript, last message, `changes.patch`, hidden-test output,
changed-file list และ workspace สุดท้าย ห้ามย้าย `auth-setup.txt` เข้า Git เพราะเป็นข้อมูลสถานะ
บัญชี แม้ไม่มี token

## Pilot corrections และรันที่ไม่นับ

การนำร่องมีไว้หาเรื่องเหล่านี้ก่อนรันจริง และพบจริง:

1. Aetox r1 ตั้ง auth ของ data root ชั่วคราวผิดวิธีจนชน token rotation ของบัญชี
   จึงเปลี่ยนเป็นใช้ OAuth store (เข้ารหัส) ของ Aetox เองใน temporary data root แล้วลบทิ้งหลังรัน
2. legacy fixture เดิมปิด connection โดยไม่ commit migration record แก้ก่อน scored run
3. Git diff เดิมไม่เห็นไฟล์ใหม่ แก้ด้วย intent-to-add และเพิ่ม `.gitignore` สำหรับ Python cache
4. Codex r3 วาง approval flag หลัง `exec`; r4–r6 พบว่า build นี้เปลี่ยนเป็น read-only เมื่อใช้
   `--ignore-user-config` ร่วมกับ workspace sandbox แม้ flag/config จะระบุ workspace-write
5. smoke ยืนยันว่า isolated config + full-access ทำงาน จึงใช้ H5 ตาม protocol เดิม

Aetox r2, r3, r4 และ r7 กับ Codex r7 ผ่าน 6/6 เช่นกัน แต่ใช้ runner/source commit คนละตัว
จึงเป็นเพียงหลักฐานประกอบและไม่ถูกนำมารวมกับ scored set หรือคำนวณ median ส่วนรอบที่จบก่อน
agent ทำงานเพราะ auth/CLI/sandbox adapter คือ setup failure ไม่ใช่คะแนน 0 ของ harness

## คำสั่งทำซ้ำ

ใช้เลขรันใหม่เพราะ runner ไม่ overwrite หลักฐานเดิม:

```powershell
.\scripts\harness-bench.ps1 -Harness codex -Task sqlite-email-migration -Run 11
.\scripts\harness-bench.ps1 -Harness aetox -Task sqlite-email-migration -Run 11 -UseExistingAetoxSession
```

ก้าวถัดไปที่จะขยับจาก pilot เป็นตัวเลขมาตรฐานคือสร้าง task อีกสี่รูปทรงตามเอกสารหลัก แล้ว
เพิ่ม OpenCode/Crush/Aider เมื่อมี provider credential เดียวกันพร้อมทุกฝั่ง
