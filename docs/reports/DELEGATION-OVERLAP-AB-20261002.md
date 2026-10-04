# ผลทดลองการแบ่งงานเมน–ลูกมือ — 2 ตุลาคม 2026

## ข้อสรุป: ไม่เก็บ candidate ใดใน production

ตามเงื่อนไขเจ้าของว่าเก็บเฉพาะสิ่งที่ได้คุ้มเสีย รอบนำร่องนี้ยังไม่มีหลักฐานพอให้ติดตั้งการเปลี่ยนแปลง:

- ปรับเฉพาะ task ใช้เวลาและ input ที่ไม่ cached น้อยลง แต่คำตอบบางรันตกหล่นมากขึ้นและมีข้อกล่าวหาที่สัญญา fixture ไม่รองรับ
- ปรับเฉพาะบทบาทไม่ได้เร็วขึ้นอย่างมีนัยต่อการใช้งานที่เห็นจากรอบนี้ และมีข้อขัดแย้งกับ no-project scope
- ปรับครบชุดลดจำนวน calls แต่ไม่ได้ให้คุณภาพดีกว่าเดิม และ median เวลาสูงขึ้นเล็กน้อย
- ที่สำคัญ baseline ของ fixture นี้ **ไม่ได้ทำงานซ้ำที่เสียแบบเหตุการณ์ Telegram** จึงไม่มีหลักฐานว่าข้อเสนอแก้พฤติกรรมที่เป็นต้นเรื่องได้

ไม่มีการแก้ production source ไม่มี commit/push ไม่มี application build และไม่เปิดแอปอีกตัว ตัวเลือกทดลองกับ test binaries อยู่ใน Temp เท่านั้น

## วิธีทดลอง

- Source snapshot ของ working tree ที่ HEAD `911d8041ce03867a0da29f2b6d2bda4fc720ae7a` รวมงาน dirty ณ เวลาจับ ไม่ใช่การรับรองว่า commit นั้นล้วน ๆ ผ่าน
- **12 controlled cells:** before, roles-only, task-only, full อย่างละ 3 รัน
- เมน Codex `gpt-6.1-sol` / ลูกมือ `gpt-5.6-terra` ระดับคิด medium ตรึงเหมือนกันทุก arm มีการตรวจ configuration และโมเดลที่ใช้จริงจาก usage ledger
- ใช้ CR2 fixture เดียวกัน: review latest commit ที่ฝังข้อผิดพลาด 5 เรื่อง พร้อม visible tests ที่ผ่าน งาน review-only ไม่ให้แก้ implementation/เทสต์
- แต่ละ cell มี workspace, state root, session, memory และ DB ใหม่ ข้อความ user request, identity/thinking และ helper profiles ตรึงไว้
- ผู้ใช้ล็อกอินชุดทดสอบเอง ไม่มี credential copy แยก auth root จาก state ด้วย instrumentation เฉพาะ snapshot ที่เหมือนกันทุก arm ไม่ใช่ production change
- ใช้ Go integration-test driver ของ console/engine จริง ไม่ใช่ Wails build หรือ executable ของแอปที่ติดตั้ง
- รัน sequential; before และ full เก็บเป็นกลุ่มละ 3 ก่อน ส่วน roles/task ablations สลับลำดับระหว่าง repetition จึงยังมี order/cache/provider-latency confounding ไม่ใช่ randomized trial
- ไม่มี matching random seed และ historical main thinking level ไม่ได้บันทึก จึงไม่ใช่ replay ตรงตัวของเหตุการณ์ Telegram
- อ่าน trace และคำตอบจริงประกอบ scorer; reconcile tokens ทั้งเมนและลูกมือกับ ledger ครบทุก cell

รัน pilot แรกที่ใช้ state root ร่วมกันถูกเก็บแยกแต่ไม่นับเป็น controlled evidence ไม่ลบหรือแทนที่ผลที่ไม่ชอบ

## ผลเชิงปริมาณ

ตัวเลขเวลาและ tokens เป็น median ของ 3 รันต่อ arm; จำนวนพบข้อผิดพลาดเรียงตาม repetition หลังตรวจคำตอบจริง

| Arm | ข้อผิดพลาดที่พบจาก 5 เรื่อง | เวลา (วินาที) | Input ไม่ cached | Input รวม cached | Output | Calls เมน+ลูกมือ |
|---|---|---:|---:|---:|---:|---:|
| Before | 5, 4, 4 | 108.1 | 39,967 | 137,759 | 4,441 | 24 |
| Roles-only | 4, 4, 4 | 107.6 | 33,117 | 128,924 | 3,970 | 25 |
| Task-only | 5, 3, 4 | 89.6 | 24,656 | 74,658 | 3,336 | 21 |
| Full | 4, 4, 5 | 111.2 | 38,185 | 105,641 | 4,417 | 19 |

เทียบ median กับ before:

- Roles-only: เวลา −0.5%, input ไม่ cached −17.1%
- Task-only: เวลา −17.1%, input ไม่ cached −38.3%
- Full: เวลา +2.8%, input ไม่ cached −4.5%

ช่วงเวลาที่วัดได้:

| Arm | ต่ำสุด–สูงสุด (วินาที) |
|---|---:|
| Before | 94.8–125.4 |
| Roles-only | 92.8–131.4 |
| Task-only | 88.1–104.6 |
| Full | 110.2–167.1 |

นี่เป็นผลสังเกตจาก sample เล็ก ไม่ใช่คำรับรองความเร็วหรือผลทดสอบนัยสำคัญทางสถิติ ไม่ตีความช่วงทับกันเป็น statistical test และไม่แปลง tokens ของ subscription เป็นเงินประหยัดหรือค่าใช้จ่ายจริง

## สิ่งที่ trace และคำตอบบอก

### ไม่ได้ผลิตซ้ำงานเสียที่เป็นต้นเรื่อง

การตรวจอิสระของ baseline/full ทั้งหกรันพบว่าการอ่านร่วมกันส่วนใหญ่เป็น `package.json` เพื่อเลือกรัน tests, การให้ Git provenance/diff แก่ explore ที่ไม่มี Git tool, และการตรวจหลักฐานหลัง collect ไม่ใช่ทั้งคู่ไล่ defect question เดียวกันซ้ำโดยไม่มีเหตุผล

- Before r1 มี dispatch สองครั้ง แต่ครั้งแรกจบเพราะเข้าถึง commit diff ไม่ได้ ครั้งที่สองเพิ่ม context ที่จำเป็น ไม่ใช่เริ่มงานซ้ำทั้งที่งานแรกยังทำอยู่
- Before r2/r3 และ Full r1/r2 ใช้ task_message เพื่อส่งข้อมูล scope ไม่ใช่ redispatch
- Before r1 และ Full r3 มีการผลิตซ้ำข้อผิดพลาดหลังรับผลที่ให้หลักฐานเพิ่ม จึงไม่จัดเป็นงานซ้ำที่เสีย

ไฟล์หรือช่วงบรรทัดทับกันอย่างเดียวจึงไม่ใช่ตัววัดผลสำเร็จของการลดงานซ้ำ

### การลด calls ไม่รับรองคุณภาพ

- Task-only r2 พลาด float-money และ listener leak และรายงานว่าการไม่ emit `booking-changed` จาก `updateBooking` เป็น defect ทั้งที่ fixture ไม่กำหนดสัญญาการเชื่อมนี้
- Full r2 มีข้อกล่าวหา event coupling แบบเดียวกัน ซึ่ง raw scorer ไม่จับ: ค่า false_flags=0 ไม่ได้พิสูจน์ precision ผ่าน
- Roles-only r1 พบ float-money จาก runtime reproduction แต่พลาด listener leak; r2/r3 พลาด float-money
- Task-only r3 รายงาน cross-user modification ชัดเจน แต่ keyword scorer ให้ IDOR=false เพราะคำว่า “Cross-user/owned” ไม่ตรงรายการคำ จึงเก็บ raw score 3/5 ไว้และนับคำตอบจริงเป็น 4/5 ใน audited comparison

## โครงสร้างและความเสี่ยงที่ตรวจ

| ชุดตรวจ | ผล |
|---|---|
| Before: prompt/bootstrap/subagent | 442 pass / 3 skip, exit 0 |
| Roles candidate: ชุดเดิมและ role/frame tests | 451 pass / 3 skip, exit 0 |
| Task candidate: ชุดเดิมและ receipt test | 443 pass / 3 skip, exit 0 |
| Full candidate: role/frame/receipt และชุดเดิม | 452 pass / 3 skip, exit 0 |
| Counterexample เพิ่มเติม: helper ใน Open/no-project scope | **Fail** — ข้อความ “Use project-relative file paths” ขัดกับ “No project is focused” |

ตัวเลข pass ของ candidate เป็นชุดก่อนเพิ่ม counterexample ไม่ใช่คำรับรองว่า candidate ผ่านทุกความเสี่ยงแล้ว ข้อขัดแย้งนี้อยู่ใน roles/full เท่านั้นและไม่ถูกติดตั้งใน production

การทดลองเพิ่ม absolute root ถูกปฏิเสธและนำออก: เทสต์เดิมตั้งใจห้าม path เฉพาะเครื่องใน prompt ไม่ลด assertion เพื่อให้ผ่าน และไม่ได้สร้าง root/context channel ใหม่

ข้อทักท้วงจาก review ว่า full เหมือน baseline ถูกตรวจหักล้าง: normalized content ต่างกันจริงทั้งหก production-path files และ test-binary hashes ของทุก arm ต่างกัน จึงไม่ได้อ้างผลจากคำสรุป reviewer โดยไม่ตรวจ source

## คอมมิต/งานเดิมและสิ่งที่ไม่แตะ

- รักษา helpers opt-in จาก `23c3b130` ไม่เปิด helpers/explore เป็น default ของสินค้า การเปิด explore ใน fixture เป็นการตั้งค่าทดลองเหมือนกันทุก arm
- รักษา closed-update→collect/UPDATE NOT DELIVERED และ Claude SDK routing/continuation/provider usage ที่มีงาน dirty อยู่
- ตรวจ SHA-256 หลังทดลองแล้ว production paths เป้าหมายทั้ง 7 ไฟล์ยังตรง snapshot ก่อนแก้: prompt.go, bootstrap.go, situation.go, task.go, packed_task.go, task_guidance.go, profile.go
- ไม่แก้ identity.md/thinking.md ส่วนตัว ไม่แก้ database จริง ไม่ stash/commit/push และไม่แตะ dev/แอปของเจ้าของ

## ข้อจำกัดและสิ่งที่ยังไม่เสร็จในระดับผลิตภัณฑ์

**อาการงานทับกันจาก Telegram ยังไม่ได้แก้** การตัดสินครั้งนี้คือไม่รับ candidate ที่ยังพิสูจน์ความคุ้มไม่ได้ ไม่ใช่ประกาศว่าไม่มีปัญหา

ไม่ได้รัน control matrix สำหรับ independent-review, handover และ same-file/different-question เพิ่ม เพราะไม่มี candidate ผ่าน primary quality/benefit gate เพื่อพิจารณาติดตั้ง จึงไม่รับรองพฤติกรรมเหล่านั้นหรือความเข้ากันได้กับ Claude SDK

ก่อนทดลองข้อเสนอใหม่ ต้องมี controlled fixture ของการค้นหลาย subsystem ที่ผลิตซ้ำงานเสียแบบเหตุการณ์ต้นเรื่องได้ แล้วค่อยวัดการจูนที่สอดคล้องกันใหม่ ไม่ใช้การรีวิวไฟล์เล็กที่ baseline แบ่งงานได้อยู่แล้วเป็นหลักฐานว่าลดงานซ้ำสำเร็จ

## หลักฐานในเครื่อง

`output/behavior-bench/delegation-overlap-20261002/`:

- `baseline.json`, `source-hashes.json`, `prior-work.patch`, `relevant-commits.txt`
- `trial-manifest-v2.json`, `compile-v2-*.json` และ source hashes
- `trials-v2/<cell>/`: request, effective models, answer, report, tool_runs, token_usage, jobs, score และ overlap audit
- `comparison.json`, `manual-audit.json`, `decision-metrics.json`, `production-unchanged.json`
- `offline-*-final.jsonl` และ `open-scope-counterexample.txt`

ตัวรัน/ร่างที่ใช้ทดลองอยู่ใน `docs/internal/delegation-overlap-20261002/` ไม่ใช่ production implementation ข้อมูลดิบไม่ถูก commit หรือเผยแพร่
