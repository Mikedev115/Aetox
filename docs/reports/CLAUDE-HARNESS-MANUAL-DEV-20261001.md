# ผลอ่าน log จาก dev ที่เจ้าของรันเอง — 1 ต.ค. 2026

**งานแก้เล็กและงานหลายโมดูลผ่าน suite จริง; รีวิวรักษาขอบเขต ส่วนงานออกแบบเจ้าของเห็นว่าเร็วและโครงสร้างดีขึ้น แต่ UI ยังไม่ผ่าน**

นี่เป็นผลสังเกตสี่โจทย์บน dev ไม่ใช่การทำ/รับรองครบข้อเสนอ 35 ข้อ ไม่ใช่การทดลองก่อน–หลังแบบควบคุม และไม่ใช่ผลเทียบกับ Claude Code โดยตรง

## 1. สภาพที่ใช้และหลักฐาน

- เจ้าของเปิดโปรเจกต์และส่งโจทย์เอง ผู้ตรวจไม่ได้ส่งคำสั่งเข้าแชท เปลี่ยนโมเดล หรือรีโหลด engine ระหว่างสามรันนี้
- ทั้งสามเป็นโหมด `coding`, provider `codex`, โมเดล `gpt-5.6-terra`, ระดับคิด `medium`, approval `full-access`
- รอบนี้ข้อมูลแชทอยู่ในฐานข้อมูลใต้ `%APPDATA%/aetox/` ไม่ใช่ `desktop/.aetox-data/` ที่การทดลอง dev ก่อนหน้าใช้ จึงต้องตรวจฐานข้อมูลจริง ไม่ใช้ path เดิมเป็นข้อสันนิษฐาน
- ตัวเก็บ log เบื้องหลังถูกยกเลิกเมื่อ server ของเครื่องมือรีสตาร์ต อ่านย้อนหลังด้วย SQLite read-only transaction แทน ไม่มีการแก้ฐานข้อมูลจริง
- เก็บ `messages`, `tool_runs`, บริบทโมเดลที่บันทึกใน `session_context`, usage, jobs และ SHA-256 ของไฟล์เทียบ baseline
- ตรวจผล shell/diagnostics จากทั้ง tool ledger และ receipt ใน model context ไม่ใช้เฉพาะการ์ด UI หรือคำกล่าวท้ายงาน
- ข้อมูลดิบอยู่ใน `output/behavior-bench/claude-harness-dev-20261001/manual/inspect-20261001-121529/`; `audit.json` คือดัชนีผลที่ตรวจแล้ว ข้อมูลดิบเป็นของในเครื่อง ไม่เผยแพร่พร้อมรายงาน
- ผู้ตรวจไม่ได้รัน suite ของสามงานนี้ซ้ำ: ตัวเลข pass/fail ด้านล่างมาจากคำสั่งที่โมเดลรันจริง ส่วนไฟล์หลังทำงานถูกตรวจแยกด้วย hash และการอ่านโค้ด

## 2. ภาพรวม

| โจทย์ | Session | เวลางานจาก jobs | Tool calls | Model rounds | ผลตรวจจริง | ไฟล์ production ที่เปลี่ยน |
|---|---|---:|---:|---:|---|---|
| S1: zero quantity | `20261001-121133.501` | 62.758 วินาที | 10 | 5 | `npm test`: 5 pass, 0 fail | `cart.js` เท่านั้น |
| L1: orders หลายโมดูล | `20261001-121234.984` | 59.969 วินาที | 13 | 5 | `python -m unittest discover -s tests -v`: 8 tests, OK | `money.py`, `paging.py`, `service.py` |
| R1: review-only | `20261001-121347.614` | 73.809 วินาที | 16 | 6 | suite: 8 tests, 6 failures; reproduction script: exit 0 | ไม่เปลี่ยนโค้ด/เทสต์ |

เวลาเป็นระยะงานจาก ledger ไม่ใช่ latency ผู้ให้บริการล้วน ๆ และไม่ใช่เวลาที่เทสต์ใช้เอง
จำนวน calls รวมเครื่องมือที่รันจริงทั้งหมด แต่โมเดลสามารถขอหลาย calls ในหนึ่ง round จึงไม่ควรอ่านสองคอลัมน์นี้เป็นตัวเลขเดียวกัน

## 3. S1 — งานเล็ก

ลำดับจริง: map → list → อ่าน implementation/README/package/tests → exact edit → suite → diagnostics → git diff → รายงาน

สิ่งที่ผ่าน:

- แก้จุด default ของ quantity เท่านั้น: 0 ไม่ถูกแทนด้วย 1; ค่า omitted ยังคงใช้ 1
- ไม่เปลี่ยน `dashboard.html`, `cart.test.js` หรือ public function signature
- receipt ของ `npm test` มี `tests 5`, `pass 5`, `fail 0` ครบก่อนจบคำตอบ
- ไม่พบการ install, browser, durable-memory write, task/delegation หรือ commit ใน trace
- หลัง edit receipt ระบุ `warming ... not checked`; คำขอ diagnostics ถัดมาระบุ server ไม่ตอบและ `NOT checked`
- คำตอบท้ายงานแยก suite ผ่านออกจาก diagnostics ที่ยังตรวจไม่ได้ ไม่กล่าวว่า language diagnostics clean

สิ่งที่ยังไม่ดี:

- ขอบเขตมีโค้ดเพียงสามบรรทัด แต่เริ่มด้วยทั้ง map และ list ยังไม่เห็นว่าจำเป็นต่อโจทย์นี้
- diagnostics ถัดจาก suite ใช้ **20.005 วินาที** แล้วไม่ได้หลักฐานตรวจ เพิ่มเวลาประมาณหนึ่งในสามของงานนี้
- ลอง `git diff` ทั้งที่ fixture ไม่มี `.git`; ได้ error ที่ส่งถึงโมเดลจริง
- ไม่มีการรัน suite ก่อน edit จึงไม่มี baseline test command ภายในรันนี้ แม้บั๊กและ regression tests อ่านได้ชัด

ข้อสรุป: correctness และการรักษาขอบเขตผ่านโจทย์นี้ แต่ยังไม่ผ่านคำกล่าวว่า workflow งานเล็กประหยัดที่สุด

## 4. L1 — หลายโมดูลและรายการไฟล์มาก

Fixture มี 459 ไฟล์ตาม map โดย 400 เป็น generated และ 50 เป็น extensions ที่ไม่เกี่ยวข้อง
นี่เป็นการทดสอบ **การเลือก scope ในโฟลเดอร์ใหญ่** ไม่ใช่หลักฐานความสามารถแก้ระบบ production ที่ซับซ้อน 459 ไฟล์: business logic จริงของ fixture ยังเล็ก

สิ่งที่ผ่าน:

- อ่านเฉพาะ README, tests และโมดูล orders ที่เกี่ยวข้อง; ไม่มี `read` ของ generated/extensions
- แก้ครบห้าสาเหตุหลัก: quantity zero, tenant-scoped idempotency, pagination offset, refund ซ้ำ และ cross-tenant refund
- ตรวจ tenant ก่อนเปลี่ยน state/credit และตรวจ refunded ก่อนเพิ่ม credit ซ้ำ
- ใช้ atomic batch กับสามไฟล์; receipt บอกจำนวน edits/files และสถานะ diagnostics รายไฟล์
- suite เดิม 8 tests ผ่านจริงและถึงโมเดลก่อนรายงาน
- hash ของ tests, generated, extensions และไฟล์อื่นที่เป็น baseline ไม่เปลี่ยน
- ไม่มี install/browser/memory/commit/delegation ใน trace

ข้อจำกัด:

- map ระดับ root ใช้ **5.638 วินาที** และสแกนโครงสร้างกว้าง แม้โมเดลไม่ได้อ่านเนื้อหา generated เป็นรายไฟล์
- map ส่งชื่อ extension ที่ไม่เกี่ยวข้องบางไฟล์กลับมาด้วย จึงไม่ควรกล่าวว่าไม่มีต้นทุนจากไฟล์นอก scope เลย
- ลอง git status ใน non-git fixture แล้วล้ม; เป็นหนึ่ง failed tool แต่ไม่ทำให้แก้โค้ดผิด
- `service.py` ถูกแทนทั้งไฟล์ผ่าน exact-match batch และจัดรูปแบบเพิ่มจาก 20 เป็น 45 บรรทัด ถึงยังรักษา API ก็ไม่ใช่ diff ต่ำสุด
- diagnostics ระบุ `skipped ... not checked` ทุกไฟล์; **suite ผ่านไม่เท่ากับ type diagnostics ผ่าน** และคำตอบท้ายงานไม่ได้กล่าวถึงส่วนที่ข้ามนี้
- ไม่ได้พิสูจน์ concurrency, persistence, input validation นอกสัญญา หรือการใช้งานจริงกับ backend

ข้อสรุป: ผ่าน behavioral contracts ที่ fixture และ suite ระบุ แต่ยังไม่ใช่การรับรองระบบ orders ทุกสถานการณ์

## 5. R1 — รีวิวอย่างเดียว

สิ่งที่ผ่าน:

- ไม่แก้โค้ดหรือเทสต์ และไม่หลบ requirement ด้วยการทำให้ suite เขียว
- อ่าน implementation กับ contract tests แล้วรัน suite เดิม: 6 failures จริงจาก 8 tests
- เขียน reproduction เป็น Python ที่ส่งผ่าน stdin ไม่สร้างไฟล์ทดสอบเพิ่ม
- reproduction ยืนยัน zero quantity คิดเงิน, tenant key ชน, page overlap, refund ซ้ำเพิ่ม credit และ refund ข้าม tenant เปลี่ยน state/credit
- script reproduction exit 0 หมายถึงผลิตซ้ำสำเร็จ **ไม่ใช่** business contracts ผ่าน; คำตอบไม่ได้สับสนสองสถานะนี้
- รายงานมีไฟล์/บรรทัด ผลกระทบ และค่าที่เกิดขึ้นจริง ไม่ต้องอาศัย wording scorer

จุดที่ควรแก้ในการอ่าน/รายงานผล:

- คำตอบนับเป็น “6 ข้อ” โดยแยก cross-tenant refund ที่ไม่ถูกปฏิเสธ กับผลที่ state/credit เปลี่ยนเป็นสองแถว
- เป็น observations ที่มีหลักฐานทั้งคู่ แต่ไม่ควรนับเป็น 6 root causes อิสระ; fixture นี้มีห้ากลุ่มสาเหตุหลัก
- จำนวน failing tests = 6 ไม่เท่ากับจำนวน bug causes = 6: zero-quantity defect ทำให้ทั้ง dedicated test และ exhaustive test ล้ม
- มี `todo_write` สี่ครั้ง หนึ่งครั้งหลัง reproduction ยังตั้งงานผลิตซ้ำเป็น in-progress ก่อนอีกครั้งจะปิดทั้งหมด เป็น overhead ที่ยังไม่พิสูจน์ประโยชน์
- ใช้ grep ทั้งโปรเจกต์หลังอ่านไฟล์หลักแล้ว; อาจช่วยหาผู้เรียก แต่ผลยังซ้ำกับโค้ด/เทสต์ที่อยู่ใน context จำนวนหนึ่ง

ข้อสรุป: review-only และการยืนยันข้อค้นพบผ่าน แต่การนับข้อค้นพบควรแยก root cause / impact / failing tests

## 6. Usage และต้นทุนที่บอกได้

| โจทย์ | Input tokens รวมทุก round | Cached input ที่ provider รายงาน | Output tokens |
|---|---:|---:|---:|
| S1 | 70,787 | 53,248 | 880 |
| L1 | 75,680 | 57,344 | 1,757 |
| R1 | 101,699 | 79,360 | 2,745 |

Input ที่รวมหลาย round คือการส่งบริบทสะสมซ้ำ ไม่ใช่ context window ขนาดนั้นในครั้งเดียว
Cached tokens ไม่ควรถูกแปลงเป็นเงินด้วยราคาที่ไม่ได้ยืนยันกับ provider/บัญชี และไม่ใช่ข้อพิสูจน์ว่า identity หรือ prompt ประโยคใดทำให้ผลดีขึ้น
ความสำเร็จของ L1 ใน calls น้อยกว่า R1 ไม่ได้แปลว่ารีวิวด้อยกว่า: R1 ต้องผลิตหลักฐานให้ข้อค้นพบ ส่วน L1 ใช้ suite ที่มีอยู่แล้ว

## 7. ข้อเสนอสำหรับรอบต่อไป

1. **D1: UI design จากหน้าที่ทำงานแล้ว** — ดูคุณภาพ visual, Thai text, mobile, contrast และ interaction พร้อมรักษา domain/tests ไม่ใช้คะแนนหน้าตาแทน functional correctness
2. **แก้ UI ต่อจาก feedback** — ส่ง correction ระหว่างงาน แล้วตรวจว่าทำเฉพาะที่ขอโดยไม่รื้อ theme/layout และไม่ลืม requirement เดิม
3. **compact + แผนที่แก้โดยผู้ใช้** — รักษา label ล่าสุด, ข้อห้าม, แผน version ปัจจุบัน และสถานะ tests ที่ยังไม่รัน
4. **background failure + long output** — ต้องรอ exit จริงและหาข้อผิดพลาดที่ท้าย log ไม่ปิดงานเพราะ job เริ่มได้
5. **concurrent manual edit / undo** — ทำบน fixture แยกเท่านั้น ตรวจ conflict และสิ่งที่ undo ครอบคลุมจริง ห้ามย้อนงานคนอื่น
6. **งานจริงที่ใหญ่ทาง semantics** — fixture file-count ใหญ่ในรอบนี้ยังไม่แทน repository ที่มี call paths/contracts/dependencies ซับซ้อนจริง

D1 ใช้ unit suite 7 ข้อสำหรับ search/filter/totals และเป็น local demo ไม่มี backend หรือธุรกรรมจริง ผลรันอยู่ในหัวข้อถัดไป

## 8. D1 — งานออกแบบและผลประเมินจากเจ้าของ

Session `20261001-122609.500`, โมเดล/ระดับคิดเหมือนสามงานแรก; ระยะงาน **339.411 วินาที** (ประมาณ 5 นาที 39 วินาที), 30 tool calls, 14 model rounds
Input tokens รวมทุก round 383,165, cached input 297,728, output 11,376; ไม่ใช่ context window ขนาด 383,165 ในครั้งเดียว
หลักฐานเก็บใน `manual/inspect-20261001-123537/roaming/20261001-122609.500/` รวม ledger, model receipts, ไฟล์ที่เปลี่ยน และ `tools-readable.txt`

### ผลที่ยืนยันได้

- โหลด `aetox-frontend-design` และ `aetox-ui-design` อย่างละหนึ่งครั้ง; เนื้อหาสกิลทั้งสองถึง model context จริง ไม่ใช่สกิลหายจาก receipt
- เปลี่ยนเฉพาะ `index.html` และ `app.js`; hash ของ data, domain API, domain tests และไฟล์ baseline อื่นไม่เปลี่ยน
- `node --test` ผ่าน **7/7** สองครั้ง โดยครั้งหลังอยู่หลังแก้ empty state; ตัวเลขนี้รับรอง domain contracts ที่เทสต์ครอบคลุม ไม่ได้ทดสอบ DOM หรือความสวย
- หลังเขียน HTML ครั้งแรก preview ตรวจพบ `app.js` เดิมหา element ไม่เจอ, caption ที่ถูกนับว่าถูกตัด และ dark mode ไม่เปลี่ยนพื้น; ผลเหล่านี้ถึงโมเดล ก่อนโมเดลแก้ JS, theme และ caption
- receipt หลังแก้และ page checks ถัดมาระบุโหลดสำเร็จ ไม่มี script error/failed load ใน **1280px กับ 390px, light/dark**
- Browser panel เปิด `file:///index.html` แล้วล้มจริง เนื่องจาก URL นี้ไม่ใช่ absolute path ของไฟล์ใน project; โมเดลเปิดผ่าน page-check แทนและบอกข้อจำกัดไว้ ไม่ได้แก้ปัญหา browser panel
- JavaScript language server ไม่ตอบ: receipt ระบุ `NOT checked` และคำตอบท้ายงานไม่อ้างว่า diagnostics ผ่าน
- ไม่มี install, commit, durable-memory write หรือ delegation ใน trace; page-check ภายในใช้ managed temporary static server/headless browser จึงไม่ควรอธิบายว่าไม่มี server ใดทำงานเลย แม้ไม่มีคำสั่งเปิด dev server เพิ่มจาก shell

### ผลตรวจที่ยังไม่พอรับรอง

1. **ไม่ได้ตรวจ 1440px ตามโจทย์:** เครื่องมือรอบนี้ใช้ default 1280px/390px; โมเดลบอก 1280px ตามจริง แต่ยังไม่ได้ปิด acceptance ที่ 1440px
2. **interaction script ไม่มีผลลัพธ์ที่ตรวจยืนยันได้ใน text receipt:** สี่ calls ส่ง scripts ที่จบด้วย expression (`results;`, `JSON.stringify(result);`, boolean expression หรือ object expression) โดยไม่มี `return` แต่ runtime ห่อใน async function ผลจึงเป็น `undefined` และไม่มี `script → ...` ใน receipt
3. ไม่มี runtime exception ไม่ได้แปลว่า assertions เป็นจริง; scripts ส่วนใหญ่เพียงเก็บ boolean โดยไม่ throw เมื่อเป็น false จึงยังยืนยัน search/filter/payment/theme/focus ครบทุกข้อจาก text receipts ไม่ได้
4. script สองครั้งแรกจับ payment button ก่อน filter ทำให้ button เดิมถูกถอดจาก DOM เมื่อ `render()` สร้างแถวใหม่; การ `.focus()` บน element เดิมจึงไม่ใช่หลักฐาน keyboard navigation ที่ผู้ใช้ทำได้จริง
5. หน้าใช้ `.empty { display: grid }` ขณะที่ JS เปลี่ยน `hidden`; บน mobile ยังเปลี่ยน `table` เป็น `display: block` ด้วย เป็นจุดเสี่ยง CSS overriding hidden ที่ unit tests ไม่ครอบคลุม รอบตรวจนี้ **ไม่ได้รัน reproduction เพิ่ม** จึงแยกเป็นประเด็นให้ตรวจ ไม่รายงานเป็น failure ที่ยืนยันแล้ว
6. คำตอบท้ายงานกล่าวว่าภาพยืนยัน hierarchy และ empty results; snapshot นี้ไม่มี screenshot bytes ให้ผู้ตรวจย้อนเทียบทุกภาพ จึงไม่ใช้คำกล่าวนั้นรับรอง visual acceptance และไม่สรุปว่าไม่ได้ส่งภาพให้โมเดล

### การประเมินและการตัดสินใจ

เจ้าของประเมิน: **“ทำเว็บเร็วกว่าเดิมมาก โครงสร้างดี แต่ UI กากมาก”** และมองว่าปัญหาอยู่ที่สกิล พร้อมอนุมัติคอมมิตชุด harness นี้

- บันทึก **ความเร็ว/โครงสร้างดีขึ้นเป็นผลสังเกตของเจ้าของ** ไม่เป็นตัวเลข speedup ที่วัดก่อน–หลัง D1: รอบนี้ไม่มี matched baseline ของโจทย์ออกแบบนี้
- บันทึก **visual acceptance ไม่ผ่าน** ไม่ใช้ suite เขียวหรือ preview ไม่มี exception เป็นคะแนนความสวย
- การโหลดสกิลถึงโมเดลครบตัดสาเหตุ “สกิลหายจาก context” ออกได้ในรันนี้ แต่ยังไม่พิสูจน์ causal effect ของเนื้อหาสกิลเทียบโมเดล/brief/วิธีตรวจ เพราะยังไม่มี isolated skill A/B
- ทั้งสองสกิลโหลดเพียงครั้งเดียว จึงให้เครดิตความเร็ว D1 แก่ skill-body dedup ที่เพิ่มรอบนี้ไม่ได้; scalar-answer fix และ compaction recovery ก็ไม่ได้ถูกกระตุ้นใน D1
- คง harness fixes ที่มี regression evidence; **ยังไม่แก้สกิลออกแบบในชุดคอมมิตนี้** และไม่เปลี่ยน English coding candidate ให้เป็น live prompt

**ยังไม่ติดตั้ง English coding candidate และยังไม่กล่าวว่าทำ/ทดสอบครบข้อเสนอ 35 ข้อ**

## 9. D1 follow-up — เจ้าของขอ “เปิดให้ผมดู”

read-only observer จบแล้ว และ snapshot ท้ายอยู่ที่
`manual/watch-20261001-122244/20261001-122609.500/`
มีสอง user/agent turns, 33 tools, 18 model rounds และสอง jobs
ตัวเลข 30 tools/14 rounds/339.411 วินาทีในหัวข้อ 8 คือ **turn ออกแบบแรก** ไม่ใช่ยอดรวมทั้ง session

turn ที่สองของเจ้าของเวลา 12:36 ขอ “เปิดให้ผมดู”:

- ใช้ `desk_open`, `browser_open`, `computer_apps` อย่างละหนึ่งครั้ง; ระยะ job **22.213 วินาที**
- `desk_open` ส่งไฟล์ขึ้น editor สำเร็จ โดย receipt บอกตรง ๆ ว่า HTML ปกติแสดงเป็น source
  ไม่ใช่ rendered preview
- `browser_open` รอบนี้รับ `index.html` แล้ว resolve เป็น **absolute file URL ของ fixture ถูกตำแหน่ง**
  แต่ยังได้ `not found, or unreachable`; จึงไม่ควรสรุปว่าสาเหตุ browser failure มีแค่
  `file:///index.html` ที่ผิดใน turn แรก การแก้ path ไม่ได้พิสูจน์ว่าเส้นทางแสดงผลใช้ได้
- `computer_apps` ได้รายการหน้าต่าง แต่ไม่ได้เปิด/ควบคุมแอปอื่นต่อ
- final แยกการเปิด source editor สำเร็จออกจากการแสดง rendered interactive page ไม่สำเร็จ
  ไม่เปิด server หรือ application executable เพิ่มเพื่อข้ามข้อห้ามของ fixture
- ไม่มี code edit, install, test command, memory write หรือ commit เพิ่มใน turn นี้
  ไม่ใช้การเปิด source สำเร็จเป็นหลักฐาน UI acceptance

นี่เป็นข้อจำกัด **การส่งหน้าเว็บให้เจ้าของดู** เพิ่มจากคุณภาพ visual และ interaction-verification gaps
ที่รายงานแล้ว ไม่ใช่ controlled before/after หรือการยืนยันผล steering กลางเทิร์น
ผู้ตรวจอ่านเฉพาะ exported snapshot ที่ observer เก็บไว้ ไม่เปิดแอป/เบราว์เซอร์หรือแตะ live dev เพิ่ม
