# ก่อน–หลัง Claude-style coding direction และ diagnostics/receipt — 1 ต.ค. 2026

**รอบแรกครบ 32 รัน ผลผสม ยังไม่ผ่านข้อสรุปว่า Coding draft ดีขึ้นโดยรวม จึงยังไม่แทนโหมดที่ใช้งานจริง**

แยกสองการทดลอง: พรอมป์ต์ใช้เครื่องยนต์ตัวเดียวทั้งก่อน–หลัง ส่วน runtime ใช้ failure fixtures เดิมก่อน–หลังแก้ซอร์ส ไม่เอาผลของสองการทดลองมาให้เครดิตกัน
นี่เป็นรอบแรกของขอบเขต 35 ข้อที่เจ้าของรับ **ไม่ใช่การทำหรือทดสอบครบ 35 ข้อ**

## 1. ทดลองพรอมป์ต์ด้วยโมเดลจริง

- `gpt-5.6-terra` / `gpt-6-luna`, ระดับคิด `high`
- `NUI1`, `B6`, `CR3`, `D2`; สองซ้ำต่อ task/model/arm = 16 ก่อน + 16 หลัง
- ตรึงซอร์สบนดิสก์รวมงานยังไม่คอมมิต แล้วสร้างเฉพาะ Go test driver ไม่ build/เปิด application executable และไม่ใช้ driver เก่าจากการทดลองครั้งก่อน
- ทั้งสองฝั่งใช้ test driver ตัวเดียว, identity/thinking ภาษาอังกฤษที่อนุมัติแล้ว, installed skills ชุดเดียว และ task/scorer ชุดเดียว
- เปลี่ยนเฉพาะ English coding direction ใน data root ที่แยก; หลังเป็นคำแปลร่างที่รับ พร้อม acting marker เดิมสำหรับ looking stances
- Workspace และ memory เริ่มใหม่ทุกรัน ไม่อ่านหรือย้าย live memory และไม่แตะ wails dev
- เก็บ answer, workspace/diff, tool trace, shell receipts, scores, token tally และเวลา;ข้อมูลดิบอยู่ที่ `output/behavior-bench/claude-harness-20261001/` บนเครื่อง ไม่เผยแพร่ผลดิบ

### ผลที่แยกจากปัญหาของ scorer ได้

| สิ่งที่วัด | ก่อน | หลัง | ความหมาย |
|---|---:|---:|---|
| NUI1: เรียงวันที่ถูก ไม่แตะ dashboard ไม่เปิด design skill | 4/4 | 4/4 | งานเล็กยังทำตรงเป้า |
| CR3: แก้ข้อที่ถูก คง integer money/เทสต์สำคัญ และ suite ผ่าน | 2/4 | 3/4 | Terra 2/2 ทั้งคู่; Luna 0/2 → 1/2 แต่ยังลบเทสต์สำคัญอีกรอบ |
| D2: มี migration pair, up ได้, คงคอลัมน์เดิม และ down คืนข้อมูลได้ | 4/4 | 4/4 | ความปลอดภัยและ rollback ไม่ได้ดีขึ้นในชุดนี้ |
| D2: ตรงกติกาแบ่งชื่อที่ hidden test กำหนด | 0/4 | 2/4 | brief ไม่ระบุกติกานี้ จึงใช้รับรองว่า migration ดีกว่าไม่ได้ |
| B6: รีวิวโดยไม่แก้ workspace | 4/4 | 4/4 | ขอบเขต review-only ยังอยู่ |

**B6 ยังเป็นจุดอ่อนของ Luna:** หลังทั้งสองรอบรายงาน SQL injection, double refund และ pagination แต่ไม่รายงานความผิดพลาดการปัดส่วนลดและการกลืน exception เป็น findings; ก่อนรอบแรกมีเรื่องการปัดส่วนลดและกล่าวถึงการกลืน exception ก่อนรอบสองก็ยังตกหล่นเรื่องเหล่านี้ จึงเป็นสัญญาณให้ระวัง ไม่ใช่หลักฐานแยกเหตุจากประโยคใด

**CR3 มีความล้มเหลวจริง ไม่ใช่แค่คะแนน:** ก่อน Luna รอบแรกเปลี่ยน integer satang เป็น decimal baht พร้อมเปลี่ยน README และลบ exhaustive test; ก่อนรอบสองยังลบ test หลังรอบแรกคง integer money แต่ยังลบ test หลังรอบสองจึงคงทั้งสองอย่างและแก้บั๊กได้ครบ

### คะแนนและต้นทุนที่ต้องอ่านอย่างระวัง

| ค่าจากตัวรัน | ก่อน | หลัง |
|---|---:|---:|
| ผ่านเกณฑ์ของ mechanical scorer ครบทุกข้อ | 6/16 | 9/16 |
| Input tokens รวม (รวมส่วนที่ cache) | 2,239,070 | 2,255,603 |
| Output tokens รวม | 65,068 | 57,876 |
| Median wall time ต่อรัน | 116.1 วินาที | 100.1 วินาที |

**ไม่ใช้ 6 → 9 เป็นข้อสรุปว่าพฤติกรรมดีขึ้น:** คะแนนที่เพิ่มมาจาก B6 หนึ่งรันและ D2 สองรัน ซึ่งมีข้อจำกัดของการจับคำ/กติกาแฝง ส่วน CR3 ที่ผลงานจริงดีขึ้นหนึ่งรันยังอาจถูก answer regex ตัดคะแนน
Input tokens ไม่ได้ลดลงโดยรวม เวลาในรอบนี้ลดลงไม่เท่ากันทุกงาน และงานที่ตรวจพบปัญหาน้อยลงก็อาจใช้เวลาน้อยลงด้วย จึงไม่เรียกว่าประสิทธิภาพดีขึ้นจากตัวเลขเวลานี้อย่างเดียว

### ข้อจำกัดที่ตรวจพบในเกณฑ์วัด

1. **B6 keyword matcher:** คำตอบที่บอก SQL ต่อ string แล้วข้าม customer boundary พร้อม repro อาจไม่ถูกนับถ้าไม่ใช้คำในชุด keywords หรือแยกตำแหน่งกับ finding คนละ paragraph
2. **CR3 answer regex:** ก่อน Terra รอบสองบอก “ไม่ลบ `test/cart.slow.test.js`” พร้อมเวลาและเหตุผล แต่ matcher ต้องการชื่อ test มาก่อนข้อความปฏิเสธจึงไม่ให้แต้ม หลังบางรันส่งรายละเอียดผ่าน report artifact ไม่ใช่ final answer ซึ่ง answer-only regex ไม่อ่าน
3. **D2 hidden convention:** scorer กำหนดคำแรกเป็น first name และคำที่เหลือเป็น last name แต่ข้อความโจทย์ไม่ได้กำหนด; การเก็บ fullname เดิมและ rollback สำเร็จยังไม่ทำให้ convention หนึ่งถูกกว่าอีกแบบโดยอัตโนมัติ

คง scorer เดิมทั้งก่อน–หลัง ไม่เพิ่ม keywords หรือเปลี่ยนกติกาย้อนหลังเพื่อให้ร่างผ่าน ตาราง code/safety เป็นการแยกผลของ executable checks เดิมออกจาก answer regex ไม่ใช่คะแนนรวมใหม่

## 2. Runtime ที่แก้และทดสอบจริง

แก้เฉพาะ diagnostics หลัง edit และทางส่ง check feedback ถึงโมเดล:

| Failure fixture | ก่อน | หลัง |
|---|---|---|
| LSP อ่านไฟล์ไม่ได้ | ตอบว่างเหมือน clean | ส่ง read error |
| ส่ง didOpen ไม่สำเร็จ | ตอบว่างเหมือน clean | ส่ง request error |
| ไฟล์นี้ไม่มีคำตอบแต่ server เคยตอบไฟล์อื่น | ตอบว่างเหมือน clean | timeout / not checked |
| Scratch workspace ถูกข้ามการตรวจ | มีแต่ write success | บอก skipped / not checked |
| Tool body ยาวจนตัด appendix | diagnostics หายจาก model receipt | แยก check feedback และส่งหลังตัด body |
| Check notes เองยาวเกิน budget | จำนวนปัญหาท้ายข้อความอาจหาย | บอก coverage partial และยอด problems รวม |

ยังคง **edit success แยกจาก diagnostic success** ไม่เปลี่ยน advisory failure ให้กลายเป็น failed edit, ไม่ติดตั้ง LSP อัตโนมัติจาก after-edit path, คง errors-only, ยอดปัญหาของ batch, redaction และ PostToolUse feedback
มี end-to-end fixture ใช้จริงผ่าน registry → packed change/write → executor → agent:ไฟล์ถูกเขียนจริง agent ได้ skipped/not-checked status และไม่มี tool-error flag ปลอม; fixture นี้แดงเมื่อ replay บน frozen baseline และเขียวหลังแก้

### Regression checks

- ชุด `internal/lsp`, `internal/turn`, `internal/mode`, `internal/prompt` ผ่านในรอบสุดท้าย
- `internal/skill` หลังตรวจซ้ำผ่าน **733 tests/subtests**, ข้าม browser/live-page tests และแยก live navigation test ที่ timeout ออก
- ชุดพื้นที่ edit/write/batch/rename/diagnostics/receipt ผ่าน; `go vet` ของสามแพ็กเกจที่แก้และ compile-only ของ engine/bootstrap/console ผ่าน
- เทสต์ path receipt เดิมสามตัวถูกปรับให้ตรวจ write body เดิมแบบ exact โดยแยก check feedback ใหม่ ไม่เปลี่ยนข้อยืนยันเรื่อง placement;ปิด shared LSP ของ test fixture ก่อน TempDir cleanup
- **ไม่อ้าง full suite เขียว:** `TestImpactLiveRealGopls` บน root จริงยัง timeout; replay ด้วยโค้ด runtime ก่อนแก้บน root เดียวกันก็ timeout จึงไม่ใช้มันให้เครดิต/โทษการแก้รอบนี้ และยังเป็นปัญหาที่ไม่ได้แก้
- ชุด baseline ที่ copy source ไม่มีกิต index บน snapshot คืนศูนย์ references ใน live navigation test จึงไม่ใช้ผลผ่านนั้นเป็น baseline ของ navigation บน root จริง

รายละเอียดและ logs อยู่ในผลดิบที่เครื่อง แยก initial failures, final rechecks และ same-root baseline ไม่ลบผลที่แดง

## 3. การตัดสินใจและสิ่งที่ยังค้าง

- **ยังไม่แทน Coding live:** ผลจริงยังผสมและยังมีการลบเทสต์ที่ควรเก็บ/รีวิวตกหล่น
- **เก็บ runtime fix ที่มีแดง→เขียวในซอร์ส:** diagnostics/status และ check feedback ไม่ถูกซ่อนจากโมเดล
- ไม่เปลี่ยน identity/thinking ที่ติดตั้งแล้ว ไม่ทับ Thai draft ที่เจ้าของแก้เอง และไม่ migrate memory
- Context/compact, evidence/goals/jobs, memory index/topics/scopes, tool discovery, scoped delegation และ permissions/checkpoints ยังต้องทำและเทสต์ก่อน–หลังเป็นชุดถัดไป ไม่ถือว่าทดสอบแล้วจาก 32 รันนี้
- ก่อนย้าย browser-only-on-user-request ไปใช้จริงต้องรักษาให้ตรงทั้ง core และ tool guidance;รอบนี้ไม่ทดสอบ UI/browser และไม่เปลี่ยน runtime permission policy

สองซ้ำต่อ cell เป็น **pilot**; ก่อนมาก่อนหลังจึงมี order/time/provider confounds การ audit ผลงานในรายงานนี้ไม่ใช่ blind independent review ไม่พิสูจน์เหตุจากข้อความเดียว ไม่รับรอง provider/model อื่น และไม่พิสูจน์ว่า Aetox ดีกว่า Claude Code
Go test driver ไม่ใช้ runtime auto-install และไม่มี browser/UI acceptance ในโจทย์ชุดนี้; stored SQLite previews บางแถวถูก byte clamp กลาง UTF-8 ใช้ replacement character เฉพาะไฟล์วิเคราะห์และเก็บ SQLite ต้นฉบับไว้ ไม่สร้าง output ที่ขาดขึ้นมาเอง
ระหว่าง pilot ไม่มี application build/launch, commit, push หรือ publication; การอนุมัติคอมมิต foundation batch เกิดหลังเจ้าของรัน dev และประเมินผลแล้ว ตามหัวข้อเพิ่มเติมด้านล่าง

## 4. Foundation runtime เพิ่มเติมและจุดหยุดที่เจ้าของรับ

หลัง pilot แก้และเพิ่ม regression fixtures อีกชุด โดย **ไม่ติดตั้ง English coding candidate**:

| กลไก | ปัญหาที่แก้ | ขอบเขตที่ยังไม่รับรอง |
|---|---|---|
| Tool-loop completion | รับคำตอบหนึ่งตัวอักษรที่ถูกต้อง เช่น `5`, `0`, `ก`, `✓` แทนการถือว่าเป็น empty reply | ไม่พิสูจน์ว่าคำตอบสั้นทุกคำถูกตามโจทย์ |
| Plan-only completion | ส่ง stance `Planning` จาก bootstrap และไม่ส่ง acting-oriented promise/unfinished-work recovery | ไม่เปลี่ยน permission policy หรือให้ plan state เป็นหลักฐานว่าลงมือแล้ว |
| Manual compact | ส่ง provider/empty-summary/cancellation error ตามจริง; persist state ทันที รวม micro-sweep ที่เกิดก่อน summary ล้ม | ยังมีข้อสังเกต meter counters หลัง reopen ที่ไม่ได้แก้ในชุดนี้ |
| Plan recovery | อ่าน structured plan ล่าสุดจาก store ตอน compact, บอก version และกำกับว่า step state ไม่ใช่ test evidence | excerpt จำกัด 12,000 ตัวอักษรและต้องอ่านแผนเต็มก่อนทำงานที่ต้องใช้รายละเอียดนอก excerpt |
| Invoked skills | พก historical skill receipts ที่มี budget หลัง compact/reopen โดยไม่แก้ stable system prefix | จำกัด 24,000 ตัวอักษร; เอกสารเกิน budget มี omitted marker ไม่ตัดเงียบ |
| Skill-body dedup | ส่ง unchanged pointer เฉพาะ body เดิมที่ยังมี model-visible receipt ครบในแชทนั้น | body เปลี่ยน/หาย/ถูกตัด หรืออีกแชท ยังได้เอกสารเต็ม ไม่ใช่การลดทุก call อัตโนมัติ |

Scalar fixture ที่ไม่มี tool definitions ใช้เส้นทาง `Respond` จึง **ไม่ใช้เป็น before/after evidence ของ tool loop**
fixture ที่ถูกต้องเพิ่ม tool definition และ replay completion guards เดิมด้วย source overlay โดยไม่เขียนทับไฟล์ที่ใช้งานจริง: old guard แดง; guard ใหม่เขียว
การแยกนี้รับรองบั๊กในกลไก ไม่ใช่การให้เครดิต identity/thinking หรือข้อความประโยคหนึ่ง

### Dev observation แยกจาก prompt pilot

- การทดลอง dev อัตโนมัติก่อนหน้าใช้ data root แยก `desktop/.aetox-data/`, โมเดล `codex/gpt-6.1-sol`, ระดับคิด `low`; root นี้ไม่มี approved identity/thinking ที่ติดตั้งใน `%APPDATA%/aetox/`
- จึงใช้รันเหล่านี้ตรวจ runtime behavior ได้ แต่ไม่ใช้เทียบผล approved identity/thinking หรือ Terra/Luna `high` ใน pilot; ไม่ได้คัดลอก prompt เข้า dev เงียบ ๆ
- เจ้าของรัน S1/L1/R1/D1 เองในรอบถัดมา ข้อมูลจริงอยู่ใต้ `%APPDATA%/aetox/`, โมเดล Terra `medium`; รายละเอียด receipts, files, usage และข้อจำกัดอยู่ใน [รายงาน dev ที่เจ้าของรันเอง](CLAUDE-HARNESS-MANUAL-DEV-20261001.md)
- S1 suite 5/5, L1 suite 8/8 และคง generated/tests, R1 ผลิตซ้ำ defects โดยไม่แก้ไฟล์; D1 domain suite 7/7 แต่เจ้าของ **ไม่รับคุณภาพ UI** แม้เห็นว่าเร็วขึ้น/โครงสร้างดีขึ้น
- pending goal/breakpoint และ mid-turn steering จากการทดลองอัตโนมัติก่อนหน้า **ยังไม่ถือเป็น acceptance สำเร็จ** เมื่อยังไม่ได้ audit ครบ และไม่ได้ส่งงานอัตโนมัติต่อแทนเจ้าของ

## 5. Expanded checks และ failures ที่ไม่ซ่อน

ค่าด้านล่างนับ Go tests รวม subtests ตาม terminal events ใน JSONL แต่ละไฟล์; หลาย batch ทดสอบซ้ำกัน จึงห้ามรวมเป็นจำนวน distinct tests

| Batch | Pass | Skip | Fail | หมายเหตุ |
|---|---:|---:|---:|---|
| Foundation | 811 | 2 | 1 | roster colour-wheel test |
| Targeted engine | 193 | 0 | 0 | ไม่ใช่ full engine |
| Tools/LSP | 758 | 10 | 0 | ไม่รวม live impact acceptance ที่ timeout |
| Final core | 491 | 0 | 0 | หลาย tests ซ้ำกับ foundation |
| Full engine | 1,164 | 19 | 4 | failures ตามรายชื่อด้านล่าง |
| Pre-commit core (cognitive/turn/lsp/bootstrap/prompt) | 456 | 1 | 0 | fresh recheck ใน shared working tree |
| Pre-commit targeted skill | 78 | 0 | 0 | diagnostics, edit/write/batch, skill lifecycle |
| Pre-commit compact engine | 2 | 0 | 0 | persistence/current structured plan |

**Full engine failures**:

- `TestTheCodingDeskCannotHandWorkToTheOffice`
- `TestApplyConfigHandsTheDelegateItsOwnProviderFactory`
- `TestTaskToolIsRegisteredForTheMainAgent`
- `TestTheModelIsToldWhichTeamItIsOnAndWhenItIsOnNone`

Frozen-source recheck มีสี่ failures ชื่อเดียวกันอีกครั้ง; ความหมายคือ reproduce ได้ ไม่ใช่ข้อพิสูจน์โดยตัวมันเองว่าเป็น failures บน committed main ก่อนงานทั้งหมด
การรัน baseline test binary เก่าที่ตอบ “no tests to run” **ไม่นับเป็น baseline evidence**
`TestTheShippedRosterIsSpreadAcrossTheColourWheel` ใน foundation และ `TestImpactLiveRealGopls` ที่ timeout ยังไม่ถูกแก้ในชุดนี้

Frontend full run มี **2,933 pass, 152 fail**; recheck เฉพาะส่วนด้วย 2 workers/timeout 30s มี **35 pass, 8 fail** ไม่ใช่ full run ที่เหลือแปด failures
recheck ที่แดงอยู่ใน `agentMemorySettings.test.ts` (empty-memory/helper-tab) และ `deskTools.test.ts` (six language-server-card assertions)
ไม่มี isolated before/after frontend baseline จึงไม่เรียก failures เหล่านี้ว่า pre-existing เพียงเพราะไม่ได้แก้ frontend ในชุดนี้
Svelte check ก่อนหน้าผ่าน แต่ไม่ได้แทน runtime/visual checks

ระหว่างตรวจคอมมิต shared tree มีงานอื่นเปลี่ยนพร้อมกัน: `go vet` ที่รวม engine ล่าสุดเจอ compile error ใน `written_desk_test.go:35` เพราะ `a.autoOpenWritten` ไม่มีใน `Engine`
ผล recheck ข้างต้นจึงเป็นของ shared working tree ณ เวลารัน ไม่ใช่การรับรอง snapshot ที่คอมมิตทั้งต้นไม้; stage เฉพาะไฟล์และ hunks ของ harness ไม่รวมงานอื่น และไม่แก้ test/โค้ดของงานอื่นเพื่อทำให้รายงานเขียว
ข้อมูลดิบและดัชนีอยู่ใน `output/behavior-bench/claude-harness-20261001/commit-evidence-index.json` บนเครื่อง

### สถานะส่งมอบ

เจ้าของรับ **foundation harness batch นี้เพื่อคอมมิต** หลังรัน dev เอง ไม่ใช่รับรองครบ 35 ข้อ และยังไม่ใช่ release
ยังไม่มีผลครบสำหรับ memory recall/index/freshness/project identity, scoped rules, context inspection, goal evidence/no-progress, permission/checkpoint/conflict และ browser-only-on-request guidance ทุกเส้นทาง
คง approved identity/thinking ภาษาอังกฤษและ Thai draft ที่เจ้าของแก้ไว้; การติดตั้ง prompt ใน data root ไม่ใช่ source commit และไม่มี live memory migration
งานแก้สกิลออกแบบเป็นเรื่องถัดไป: D1 พบสกิลถึงโมเดลครบแต่ visual acceptance ไม่ผ่าน และยังไม่มี isolated skill A/B ที่ระบุเหตุได้
ไม่ push/publish หรือบิ้ว application เพื่อคอมมิตชุดนี้
