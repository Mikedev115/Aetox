# Harness database pilot — Aetox vs Codex (2026-09-18)

Status: **Pilot สำเร็จ · ยังห้ามใช้เป็นผลเปรียบเทียบทั่วไป**

รอบนี้พิสูจน์ว่า protocol ใน
[`harness-bench-2026-09-14.md`](../architecture/harness-bench-2026-09-14.md)
มี fixture, hidden scorer และ runner ที่ใช้ได้จริงกับคู่ Aetox/Codex แล้ว ไม่ได้พิสูจน์ว่า
harness ใดดีกว่าโดยรวม เพราะมีเพียงหนึ่งงานและหนึ่ง scored run ต่อฝั่ง ขณะที่เกณฑ์เผยแพร่เดิม
กำหนด 5 งาน × 3 รอบ × 2 ฝั่งต่อคู่ รวม 120 รันทั้งชุด

## สิ่งที่ตรึงก่อน scored run

| รายการ | ค่า |
|---|---|
| Source / runner / fixture commit | `6a6bc6fc360167d642badb98ffb63e080bb98bd8` |
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

เฉพาะ `*-r7` เท่านั้นที่เป็นคู่ scored เพราะ hash และ adapter ตรงกัน:

| Metric | Aetox r7 | Codex r7 |
|---|---:|---:|
| Hidden tests | **6/6** | **6/6** |
| Harness exit | 0 | 0 |
| Scorer exit | 0 | 0 |
| Wall time | 113.414 s | **74.330 s** |
| Files changed | 2 | 2 |
| Diff size | +25 / −10 | **+19 / −4** |
| Longest line | 81 | 81 |
| Final `app.py` | 79 lines | **74 lines** |
| Migration file | 3 lines | 8 lines |
| Asked user | 0 | 0 |

ผลที่พูดได้จากรอบนี้:

- **Correctness เสมอกัน:** ทั้งสองผ่าน behavior ที่ซ่อนทั้งหมด รวม atomic collision rollback
  ซึ่งเป็นส่วนยากที่สุดของโจทย์
- **Codex เร็วกว่า 39.084 วินาที** และ diff เล็กกว่าในรอบนี้
- ทั้งสองแตะเฉพาะ `app.py` และ migration ใหม่ ไม่แก้ migration 001 หรือ visible tests
- Aetox อ่าน `aetox-database` พร้อม `local-workflow`/`migrations`, ใช้ 22 tool rounds,
  input 413,242 tokens (cached 269,824) และ output 2,355 tokens
- Codex รายงานเพียง aggregate `19,523 tokens`; telemetry คนละรูปแบบ จึง **ห้ามเอาตัวเลข
  token สองฝั่งมาหารหรือประกาศว่าฝั่งใดประหยัดกว่า**

ผลที่ยังพูดไม่ได้:

- ห้ามสรุปว่า Aetox หรือ Codex ดีกว่าโดยรวม;
- ห้ามเอาเวลา 1 รอบไปอ้างเป็น performance distribution;
- ห้ามย้ายตัวเลขนี้ขึ้น `BENCHMARK.md` หรือหน้าเผยแพร่;
- ยังไม่มีผล OpenCode/Crush/Aider และยังขาดอีกสี่ task shapes

## Raw artifacts

เก็บใน working tree ของเครื่องนี้ ไม่อยู่ในคอมมิต:

- `output/harness-bench/aetox-sqlite-email-migration-r7/`
- `output/harness-bench/codex-sqlite-email-migration-r7/`

แต่ละโฟลเดอร์มี `result.json`, transcript, last message, `changes.patch`, hidden-test output,
changed-file list และ workspace สุดท้าย ห้ามย้าย `auth-setup.txt` เข้า Git เพราะเป็นข้อมูลสถานะ
บัญชี แม้ไม่มี token

## Pilot corrections และรันที่ไม่นับ

การนำร่องมีไว้หาเรื่องเหล่านี้ก่อนรันจริง และพบจริง:

1. Aetox r1 ใช้การ import Codex CLI refresh token ซึ่งชน token rotation (`refresh_token_reused`)
   จึงเปลี่ยนเป็นคัดลอก encrypted Aetox OAuth store ไป temporary data root แล้วลบทิ้งหลังรัน
2. legacy fixture เดิมปิด connection โดยไม่ commit migration record แก้ก่อน scored run
3. Git diff เดิมไม่เห็นไฟล์ใหม่ แก้ด้วย intent-to-add และเพิ่ม `.gitignore` สำหรับ Python cache
4. Codex r3 วาง approval flag หลัง `exec`; r4–r6 พบว่า build นี้เปลี่ยนเป็น read-only เมื่อใช้
   `--ignore-user-config` ร่วมกับ workspace sandbox แม้ flag/config จะระบุ workspace-write
5. smoke ยืนยันว่า isolated config + full-access ทำงาน จึงใช้ H5 ตาม protocol เดิมใน r7

Aetox r2, r3 และ r4 ผ่าน 6/6 เช่นกัน แต่ใช้ runner commit คนละตัว จึงเป็นเพียงหลักฐานประกอบว่า
คำตอบไม่ได้เกิดครั้งเดียว และไม่ถูกนำมารวมกับ scored pair หรือคำนวณ median

## คำสั่งทำซ้ำ

ใช้เลขรันใหม่เพราะ runner ไม่ overwrite หลักฐานเดิม:

```powershell
.\scripts\harness-bench.ps1 -Harness codex -Task sqlite-email-migration -Run 8
.\scripts\harness-bench.ps1 -Harness aetox -Task sqlite-email-migration -Run 8 -UseExistingAetoxSession
```

ก้าวถัดไปที่จะขยับจาก pilot เป็นตัวเลขมาตรฐานคือรัน task นี้ให้ครบสามรอบต่อฝั่งบน hash
เดียวกัน แล้วสร้าง task อีกสี่รูปทรงตามเอกสารหลัก ก่อนเพิ่ม OpenCode/Crush/Aider เมื่อมี provider
credential เดียวกันพร้อมทุกฝั่ง
