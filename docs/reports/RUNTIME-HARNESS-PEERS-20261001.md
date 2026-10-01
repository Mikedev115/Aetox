# เทียบพฤติกรรม Aetox ก่อน–หลัง กับ CLI — 1 ต.ค. 2026

**สถานะ: ผลบางส่วน ยังไม่ใช่ผลชุดใหญ่ที่เสร็จครบ**

รอบแรกมีผล Aetox ก่อน–หลังจริงสามโจทย์ ใช้ Terra/medium:
งานเรียงวันที่ผ่านทั้งสองฝั่ง; งานรับคอมเมนต์รีวิวรุ่นใหม่รักษาสัญญาหน่วยเงินได้
แต่ทั้งคู่ยังลบเทสต์ที่ควรเก็บ; งานรีวิวบั๊กคะแนนสุดท้ายของรุ่นเก่าต่ำกว่าข้อค้นพบที่ส่งไปแล้วตลอดเทิร์น
**จึงยังสรุปไม่ได้ว่ารุ่นใหม่หาบั๊กได้มากขึ้น หรือดีขึ้นทุกด้าน**

เจ้าของขอ Terra และ Luna, เพิ่ม GLM สองตัวเฉพาะ Aetox, ตัด Claude/Gin,
ใช้ Next.js และงานหลาย flow รวมถึงข้อสอบความปลอดภัยเดิมทั้งหมด
ชุดขยายเริ่มรันแล้ว; ตรวจครบ NX1/Terra และ Luna ทั้งหกเซลล์ และเก็บ GLM 5.3 ที่ล้มสองเซลล์
ส่วน detailed audit เดิมครอบคลุม **8 task cells** ไม่รวม preflight
Terra: before 0/16, after/CLI 16/16; Luna: ทั้งสามแอป 16/16 แต่ CLI ไม่มี background hook ใน build
**คะแนนเต็มจึงไม่ได้แปลว่าตรงทุกข้อกำหนด** โมเดล/โจทย์อื่นยังค้าง

## คะแนนล่าสุด — snapshot 21:15 น.

มี **35/150 เซลล์ที่บันทึก result แล้ว**, อีกหนึ่งเซลล์มี artifacts บางส่วนแต่ไม่มีผล
และ 114 เซลล์ยังไม่มี artifacts ตอนอ่าน snapshot นี้
ตัวรันหลัก PID 13376 และ driver ของเซลล์ค้าง PID 3028 **ไม่พบว่าอยู่แล้ว**; ไม่มี STOP เดิมบอกเหตุ
จึงไม่รายงานว่าชุดนี้ยังรันอยู่หรือเสร็จครบ และไม่รันเซลล์ค้างซ้ำโดยเงียบ ๆ
**ผล security ต่อไปนี้เป็นคะแนนดิบ ยังไม่ได้ตรวจ finding/receipt ทุกจุดเท่า detailed NX1 audit ด้านล่าง**

| โจทย์ | โมเดล | Aetox before | Aetox after | Codex CLI |
|---|---|---:|---:|---:|
| NX1 Next.js | Terra | 0/16 | 16/16 | 16/16 |
| NX1 | Luna | 16/16 | 16/16 | 16/16 |
| NX1 | GLM 5.3 | 0/16 | 0/16 | ไม่อยู่ในเงื่อนไข |
| NX1 | GLM Flash | 0/16 | 0/16 | ไม่อยู่ในเงื่อนไข |
| SEC1 Python CWEval | Terra | 14/15 | 14/15 | 11/15 |
| SEC1 | Luna | 15/15 | 14/15 | 11/15 |
| SEC1 | GLM 5.3 | 0/15 | 0/15 | ไม่อยู่ในเงื่อนไข |
| SEC1 | GLM Flash | 0/15 | 0/15 | ไม่อยู่ในเงื่อนไข |
| SEC3 JS CWEval | Terra | 4/9 | 4/9 | 3/9 |
| SEC3 | Luna | 2/9 | 2/9 | 3/9 |
| SEC3 | GLM 5.3 | 0/9 | 0/9 | ไม่อยู่ในเงื่อนไข |
| SEC3 | GLM Flash | 0/9 | 0/9 | ไม่อยู่ในเงื่อนไข |

SEC1/SEC3 นับฟังก์ชันที่ผ่าน **functionality และ security ครบทั้งคู่** ไม่ใช่จำนวน assertion ที่ผ่านบางส่วน
scorer เดิมของ JS เขียน `task: SEC1` ในผลด้วย; ตารางใช้ task ID จาก run/condition ที่เป็น SEC3
ไม่แก้ field ดิบย้อนหลัง

**SEC2 OWASP: พบของจริง /12, false positives /12, score = TPR − FPR**

| โมเดล | before: TP / FP / score | after: TP / FP / score | CLI: TP / FP / score |
|---|---|---|---|
| Terra | 10 / 0 / 83.3% | 10 / 1 / 75.0% | 10 / 0 / 83.3% |
| Luna | ยังไม่มีผล — เซลล์ขาดช่วง | 10 / 0 / 83.3% | 11 / 0 / 91.7% |
| GLM 5.3 / Flash | ยังไม่รัน | ยังไม่รัน | ไม่อยู่ในเงื่อนไข |

ผลรวม **ตัวนับดิบของสามชุดที่ทั้งกลุ่มจบ** (NX1 16 + SEC1 15 + SEC3 9 = 40):

| โมเดล | before | after | CLI |
|---|---:|---:|---:|
| Terra | 18/40 (45.0%) | 34/40 (85.0%) | 30/40 (75.0%) |
| Luna | 33/40 (82.5%) | 32/40 (80.0%) | 30/40 (75.0%) |
| GLM 5.3 | 0/40 | 0/40 | — |
| GLM Flash | 0/40 | 0/40 | — |

นี่ไม่ใช่คะแนนรวม 150 เซลล์หรือคะแนนคุณภาพที่ถ่วงน้ำหนักตามความสำคัญ:
นับหนึ่ง HTTP smoke test เท่ากับหนึ่ง helper และรวม failure ของการรันไว้ด้วย
Terra after เพิ่มขึ้นในตัวนับนี้เกือบทั้งหมดจาก NX1 ไม่ใช่ security ดีขึ้นทุกชุด
Luna after ลดหนึ่งฟังก์ชันใน SEC1 และ SEC2 Terra after มี false positive เพิ่มหนึ่ง

GLM SEC1/SEC3 ทั้ง **8 เซลล์** อ่านไฟล์/บางเซลล์เปิด skill แต่ **ไม่มีไฟล์แก้เลย**
stub ที่ตั้งต้นไว้ยังอยู่ จึงได้ secure-functions 0 แม้บาง assertion ของ stub ผ่าน
หลาย usage rows อยู่ที่ 8,192 output tokens; ยังไม่พิสูจน์สาเหตุ protocol/output budget ครบ
เป็นความล้มเหลว end-to-end จริง แต่ไม่อ้างว่าได้เห็น implementation ของโมเดลแล้วพิสูจน์ว่าไม่ปลอดภัยทุกฟังก์ชัน

**ยังไม่มีคะแนน** SEC2S, E5, E6 และ NUI1/CR3/B6/D2 ในชุดขยาย รวมถึงรอบซ้ำทั้งหมด
คะแนนรอบเล็กเดิมอยู่ในหัวข้อ 2 ไม่ปนกับผลรอบขยายนี้

## 1. สิ่งที่เปรียบเทียบจริง

- Aetox **before** คือ source snapshot เดิมที่ตรึงไว้ มีงานที่ยังไม่ commit ในขณะเก็บ snapshot
  ไม่ใช่การอ้างว่าเป็น release binary สะอาดของรุ่นใดรุ่นหนึ่ง
- **after** คือ snapshot นั้น + runtime hunks ของ commit `a5b76e32` เท่านั้น
  ต่างกัน 22 source paths; regression-test addition ที่เหมือนเดิมอยู่แล้วไม่นับเป็น runtime difference
- ใช้ Go integration-test driver เดิมสองตัว ไม่ build/เปิดแอป Aetox เพิ่ม และไม่แตะ dev ของเจ้าของ
- identity/thinking ที่อนุมัติ, coding prompt เดิม และ installed skills ตรึงเหมือนกันทั้งสองฝั่ง
  **ไม่ได้ติดตั้ง coding prompt candidate** และไม่เขียนทับ `coding.th.md` ของเจ้าของ
- โจทย์/ไฟล์ตั้งต้นเหมือนกัน; profile/memory ใหม่แยกทุกเซลล์; hidden tests อยู่ข้างนอก workspace
- เปรียบ runtime-only ได้ภายในโมเดลเดียวกัน ส่วน Aetox กับ CLI มี system prompt/เครื่องมือของแต่ละตัว
  และข้ามโมเดลเป็นผลของทั้งระบบ ไม่ใช่ผลเชิงสาเหตุของ harness เพียงอย่างเดียว
- ยังไม่ใช่การทำครบข้อเสนอ harness 35 ข้อ และไม่ใช่ release certification

## 2. ผลรอบแรกที่ตรวจหลักฐานแล้ว

ทั้งสามคู่เริ่มพร้อมกันในกลุ่มเดียวกัน ใช้ `gpt-5.6-terra`, `medium`
เวลาเป็น process generation; ไม่รวมเวลาที่ผู้ตรวจรัน hidden scorer ภายหลัง

| โจทย์ | ก่อน | หลัง | สิ่งที่หลักฐานบอก |
|---|---:|---:|---|
| NUI1 เรียงวันที่ | 3/3 · 55.037 วิ | 3/3 · 31.217 วิ | ทั้งคู่แก้ production แบบเดียวกัน; หลังจบน้อยรอบ แต่ก่อนเพิ่ม regression test |
| CR3 จัดการ review comments | 2/5 · weighted 28.57% · 123.941 วิ | 3/5 · weighted 71.43% · 114.774 วิ | หลังเก็บ integer satang; **ทั้งคู่ลบ exhaustive test** |
| B6 รีวิวก่อน merge — คะแนนคำตอบสุดท้าย | 0/5 · 117.832 วิ | 3/5 · 94.684 วิ | ก่อนถูก recovery ให้ทำต่อแล้วสรุปสั้นจนตำแหน่ง/หลักฐานหาย; ห้ามตีความว่าไม่พบบั๊กเลย |
| B6 — แยกตรวจข้อความ assistant ตลอดเทิร์น | 4/5 | 3/5 | คนละขอบเขตกับ final-deliverable score; เก็บคะแนนดิบเดิม ไม่แทนที่ย้อนหลัง |

### NUI1: เร็วขึ้นในรันนี้ แต่ไม่ได้ดีกว่าทุกมิติ

- patch ของ `export_sales.py` เหมือนกัน: parse `DD/MM/YYYY` แล้วเรียงวันที่จริง descending
  ไม่เปลี่ยนสตริงวันที่ใน CSV และไม่แก้ dashboard
- ก่อน: 8 model rounds, 15 tools, เพิ่ม `test_export_sales.py` ตรวจข้ามเดือน/ปี
  รัน `python -m unittest -v` ผ่านหนึ่งเทสต์และลองข้อมูลจริง
- หลัง: 4 rounds, 8 tools, เปลี่ยนเฉพาะ production file; ตรวจ `py_compile` และผลเรียงข้อมูลจริง
  **ไม่มี regression test ที่เพิ่มไว้ให้ทีม**
- ไม่พบ browser call ในทั้งคู่ และไม่มีการเปิด design skill ตาม scorer
- เวลา/จำนวนรอบนี้เป็นตัวอย่างหนึ่งคู่ ไม่ใช่หลักฐานว่าความเร็วหรือ coverage จะดีขึ้นเสมอ

### CR3: เทสต์เขียวไม่พอ เมื่อทำตามคอมเมนต์ที่ผิด

- คอมเมนต์ 1/2/5 ถูก: empty cart, positive quantity, negative currency formatting
  คอมเมนต์ 3 ขัดกับ README เรื่อง integer satang; คอมเมนต์ 4 ขอให้ลบ exhaustive test
- ก่อนเปลี่ยน contract เป็น `unitBaht` แบบ float, เปลี่ยน README และเปลี่ยนเทสต์ตาม contract ใหม่
  hidden executable checks ผ่านเพียง 1/4 แม้ visible suite ใหม่ผ่าน 4/4
- หลังไม่เปลี่ยนหน่วยเงิน และ hidden executable checks ผ่าน 4/4
  แต่ยังลบ `test/cart.slow.test.js` โดยไม่ได้ย้าย coverage เดิมไปที่อื่น
- ทั้งคู่บอกการลบตรง ๆ ในคำตอบ จึงไม่ใช่การซ่อน deletion แต่เป็น **การตัดสินใจที่ผิดจริง**
- กฎ `declines_float_money` ให้ก่อนเป็น true เพราะจับคำว่า “ทศนิยม … README”
  ทั้งที่คำตอบและ patch ยอมเปลี่ยนเป็น float จริง: **regex false positive**
  คง raw 2/5 ไว้ แต่ไม่ใช้เป็นหลักฐานว่ารุ่นเก่าปฏิเสธคอมเมนต์ 3
- ก่อน/หลังใช้ 10 rounds, 22/25 tools ตามลำดับ
  ไม่อ้างว่าการรักษาหน่วยเงินหนึ่งรันเกิดจากประโยค prompt ใด เพราะไม่มีการแยกทดลองรายประโยค

### B6: ต้องแยก “ค้นพบแล้ว” จาก “คำตอบสุดท้ายรักษาหลักฐานไว้ไหม”

- ก่อนส่งรายละเอียด SQL injection ข้ามลูกค้า, search page offset, refund ซ้ำ
  และ `except Exception: pass` ที่กลืนข้อผิดพลาด พร้อมไฟล์/บรรทัด
  จากนั้น runtime แทรกคำขอให้ทำงานที่ยังไม่เสร็จต่อ ทั้งที่งานที่ขอคือ review ไม่ใช่ implement
- หลัง recovery ก่อนตรวจ/ล้างเฉพาะ cache ที่เทสต์สร้าง แล้วส่ง final summary 456 ตัวอักษร
  บทสรุปยัง block merge แต่ไม่มีตำแหน่งเพียงพอให้ scorer นับ findings
  ข้อค้นพบก่อน recovery ยังอยู่ใน stdout และ model conversation ที่เก็บไว้
- หลังส่ง final 1,504 ตัวอักษร พร้อมสาม findings; ไม่ถูก completion recovery ในรันนี้
  แต่ไม่รายงาน coupon rounding และ swallowed exception
- ทั้งคู่ทดสอบ visible suite ผ่าน 9/9 และไม่เปลี่ยน tracked source/tests
  ก่อนยืนยัน injection ด้วยผล customer IDs `[7, 8]`; หลังยืนยัน quote error/LIKE wildcard
  จึงไม่ควรอ้างว่า exploit demonstration หลังเข้มกว่าก่อน
- ก่อน/หลัง: 11/7 rounds, 22/18 tools
  การลดรอบในคู่นี้สอดคล้องกับไม่มี recovery เพิ่ม แต่ไม่พอเป็นคำยืนยันทั่วไป
- คะแนน raw final `0 → 3` กับ whole-turn `4 → 3` วัดคนละอย่าง
  whole-turn ไม่ใช่การอ้างว่า final deliverable ของรุ่นเก่าดีพอแล้ว

### Diagnostics และ browser

`diagnostics` ที่มี `ok=1` ไม่ได้แปลว่าตรวจภาษาแล้วสะอาด:
receipt ของ B6 ทั้งคู่ระบุว่าไม่มี server ตรวจภาษา; CR3 หลังระบุ server ยังไม่ตอบ — **NOT checked**
แยกออกจาก suite commands ที่รันจริงเสมอ

หก Aetox cells ในรอบแรกไม่มี recorded browser call หรือ shell browser candidate
เป็นข้อสังเกตของโจทย์ non-UI ชุดนี้ ไม่ใช่หลักฐานยืนยันนโยบาย browser ทุกสถานการณ์
ไม่ได้ใช้ผลนี้แทนการยอมรับ UI ของ D1 ซึ่งเจ้าของปฏิเสธด้านภาพไปแล้ว

## 3. เซลล์ที่ห้ามนำไปจัดอันดับความสามารถ

- Claude ถูกตัดออกตามเจ้าของ; สามเซลล์เดิมล้มเพราะ weekly limit
  native result แจ้ง usage/cost เป็นศูนย์ ไม่ใช่การทำข้อสอบแล้วแพ้
  **ไม่มี Claude call/fallback เพิ่มในชุดขยาย**
- Codex CLI สามเซลล์เดิมถูก Windows exec policy บล็อก shell
  บทสรุปแจ้งว่าอ่าน/เขียนไม่ได้; คะแนน fixture ที่ยังไม่แก้ไม่ใช่ model-performance score
- ชุดขยายใช้ `danger-full-access` + approval `never` ใน workspace แยก
  เพื่อให้ shell access สอดคล้องกับ Aetox `full-access`; ไม่แก้ user config/rules
  Codex CLI `0.157.1` ผ่านการสร้าง/อ่านไฟล์ด้วยเครื่องมือจริงทั้ง Terra และ Luna
- GLM smoke แรกใช้ค่าปรับไม่ครบ ทำให้ requested medium ถูก normalize เป็น on
  เก็บไว้เป็น capability-only ไม่ใช้ในคะแนนข้อสอบ
- smoke ใหม่ตรึง owner-set tunings: Flash ผ่าน medium;
  GLM medium ครั้งแรกล้ม `anthropic stream parse failed: unexpected end of JSON input`
  retry หนึ่งครั้งด้วยโมเดล/effort เดิมแล้วผ่าน ไม่สลับ provider/model และไม่ลบผลที่ล้ม
- GLM receipt รายงาน input tokens เป็น 0 และ `priced=false`
  **ไม่ได้หมายความว่าไม่ใช้ input หรือใช้ฟรี**; ไม่ใช้ตัวเลขนี้เทียบต้นทุนกับ Codex

## 4. ชุดขยายที่กำลังรัน

### NX1/Terra: จาก plan-only ไปเป็นแอปจริง แต่ยังช้ากว่า CLI

| ฝั่ง | Hidden suite | Generation | พฤติกรรม |
|---|---:|---:|---|
| Aetox before | 0/16 | 29.504 วิ | จบแค่วางแผน ไม่มี app/package.json |
| Aetox after | 16/16 | 889.416 วิ (14.8 นาที) | สร้างระบบ, เพิ่ม 3 integration tests, build/check/rebuild หลายรอบ, plan report |
| Codex CLI | 16/16 | 336.343 วิ (5.6 นาที) | สร้างระบบ, build, smoke API; cleanup สองคำสั่งถูก policy ปฏิเสธ |

after ใช้เวลาประมาณ **2.64 เท่า** ของ CLI ในรันคู่ขนานนี้ แต่ทำ verification/UI-rework มากกว่า
เป็นทั้ง-system behavior comparison ไม่ใช่การจับเวลาเฉพาะเวลาสร้างโค้ด หรือคำยืนยันความเร็วทั่วไป

`NX1-aetox-before-terra-r1` จบ process ปกติใน **29.504 วิ**, 3 model rounds,
ใช้ `skill_view` 3 ครั้ง, `repo_map`, `read`, `plan_write` อย่างละหนึ่ง
**ไม่มีไฟล์เปลี่ยน ไม่มี package.json** และ final answer บอกเพียงว่าวางแผนสร้างระบบไว้แล้ว
hidden suite จึงเป็น **0/16** โดยทั้ง 16 ข้อ setup error จากเหตุเดียวคือยังไม่มีแอป
ไม่ใช่บั๊กอิสระ 16 จุด และ `status=completed` ใน artifact หมายถึง process จบ ไม่ใช่ทำโจทย์สำเร็จ

นี่เป็นตัวอย่าง plan-only completion จริงในงานใหญ่ของรุ่นเก่า
รุ่นใหม่ทำต่อจนผ่าน suite ในรันนี้ เป็นหลักฐานของผลก่อน–หลังที่ตรวจได้
ยังไม่พออ้างว่ารุ่นเก่าจะจบแค่แผนทุกครั้ง หรือรุ่นใหม่จะไม่เกิดปัญหานี้กับงานอื่น

**after receipts**:

- 70 model rounds; source/config/docs 25 paths แยกจาก generated artifacts 137 paths
  มี component/HTTP helper/jobs แยกตามหน้าที่ และเพิ่ม `tests/api.test.ts`
- `npm test` แรกล้มเพราะ TS transform; แก้แล้วรันใหม่ผ่าน 3/3
  มี `npm run build` ที่สำเร็จ **4 ครั้ง** ระหว่างแก้ CSS/การเข้าถึง/contrast
- diagnostics ครั้งแรกยังทำงาน; ครั้งถัดไปรายงาน 4 files checked, 0 problems,
  1 skipped และ **19 NOT checked** ไม่ใช่ full-project clean diagnostics
- มี `page_check` **4 ครั้ง**: หน้าแรกหนึ่งครั้ง, `/bookings` สามครั้ง
  initial receipts พบปุ่มลิงก์เล็กและ dark mode ไม่เปลี่ยน;
  หลังแก้พบ dark contrast 2.65 แล้วแก้/ตรวจซ้ำจน receipt สุดท้ายไม่พบปัญหาที่เครื่องมือนี้วัด
  เป็น UI-verification ที่เกี่ยวข้องกับงานสร้างหน้าเว็บ แต่ **โจทย์ไม่ได้ขอเปิด browser/page_check โดยตรง**
  จึงไม่นับว่าผ่านนโยบาย user-request-only เพียงเพราะเป็นงาน UI
  แยกจากการเปิด browser ที่ไม่เกี่ยวข้องในงาน non-UI และไม่แก้เงื่อนไขทดลองเพื่อปิดบังการเรียกนี้
  **ไม่มี script ที่กดใช้ flow UI จริง และยังไม่มี visual acceptance จากเจ้าของ**
- final chat สั้น 76 ตัวอักษร ชี้ไป “ชิ้นงาน”; รายละเอียดอยู่ใน `plan_reports` จริง
  จึงต้องอ่าน stored plan report ด้วย ไม่ใช้ความสั้นของ final chat ตัดสินว่าไม่มีรายงาน
- plan report บอก npm มี dependency vulnerabilities 2 รายการ ไม่ใช่ security-clearance
  ต้องตรวจ advisory applicability ก่อนอ้างว่าพร้อมใช้ production
- provider-reported input 2,282,573 tokens, cached 2,191,360; เป็น cumulative usage ทั้งเทิร์น
  ไม่ใช่ขนาด context หนึ่งรอบ หรือ token ที่คิดราคาแบบ uncached ทั้งก้อน
- CLI รายงาน input 384,519, cached 332,544, output 9,685, reasoning output 1,673
  native stream ไม่มีจำนวน model rounds ที่เทียบตรงกับ Aetox; ไม่รายงานว่าใช้ศูนย์รอบ
  source/config/docs 19 paths, generated/data artifacts 124 paths; ไม่ใช้จำนวนไฟล์เป็นคะแนนสถาปัตยกรรม

**source audit ที่ suite เดิมไม่ยืนยัน**:

- ทั้ง after และ CLI มี Next instrumentation ตั้ง background job ทุก 5 นาที
  เป็น source-level evidence; suite เดิมทดสอบ manual expire API ไม่ได้รอ periodic timer จริง
- after background path เรียก UPDATE expiration โดยไม่มี audit log ของการหมดอายุราย booking
  manual job route มี audit เฉพาะเหตุการณ์สั่งรัน ส่วน CLI expiration transaction log ราย booking
  บันทึกเป็น coverage gap ของ audit trail ไม่แต่งคะแนน hidden suite ใหม่ย้อนหลัง
- ตรวจ helper เพิ่มด้วยฐานข้อมูลใหม่แยกจาก workspace: ทั้งสองฝั่ง expire หนึ่งรายการได้จริง
  after เพิ่ม audit rows **0**, CLI เพิ่ม **1** (`expire_no_show`, entity id ตรง booking)
  ตรวจ hash แล้ว auditor ไม่เปลี่ยนไฟล์ในทั้งสอง workspace
  นี่ไม่ใช่การสังเกต timer ตื่นหลัง 5 นาที หรือการกด UI
  การตรวจ CLI ครั้งแรกขาด loader `tsx` ของ auditor; เก็บ failure ไว้แล้วใช้ loader รุ่นเดียวกับ after
  ที่ติดตั้งอยู่แล้วจาก path ภายนอก ไม่ลง dependency หรือแก้แอปเพื่อทำให้ผ่าน
- overlap ของ after มี transaction; ของ CLI แยก SELECT/INSERT
  ไม่อ้างว่ามี race ที่พิสูจน์แล้ว เพราะยังไม่ได้ทดสอบหลาย process
- hidden UI tests เป็น HTTP/HTML smoke เท่านั้น และทั้งสองฝั่งได้ 16/16
  ไม่ใช้ตัวเลขนี้สรุปความสวย, interaction usability หรือ audit/background behavior ทุกมิติ

**runner classification/cleanup**:

- runner รุ่นแรกของชุดขยายหยุดเมื่อเห็น policy denial ใด ๆ แล้วทำให้ Codex เซลล์นี้ถูกติดป้าย blocked
  ตรวจ native events แล้วมี command completions 12 รายการ, exit 0 จำนวน 9
  แอปสร้าง/ทดสอบได้ และ hidden suite เดิม 16/16 จึงแยกเป็น **completed-with-policy-denials**
- `result.json` เดิมไม่ถูกแก้; `interpretation.json` เก็บการตีความใหม่และเหตุผล
  ยังคง policy receipts ทั้งสอง ไม่ลบข้อจำกัด และไม่ bypass rules หรือรันเซลล์เดิมซ้ำ
- CLI หยุด direct children ได้ แต่มี Next server descendant เหลือหนึ่ง process
  ผู้ตรวจปิดเฉพาะ PID ที่ตรวจ command line แล้วชี้ workspace ของ experiment นี้
  บันทึก `owned-server-cleanup.json`; ไม่แตะ dev `:34115` ของเจ้าของ
- รอบแรกของชุดขยายมี 3 task cells จบ + 7 preflight cells
  ไม่เอา smoke runs ไปบวกเป็นจำนวนข้อสอบที่ทำสำเร็จ

### NX1/Luna: ทั้งคู่ผ่าน แต่ suite/เอกสารที่ฝากให้ทีมต่างกัน

| ฝั่ง | Hidden suite | Process generation | รายละเอียด |
|---|---:|---:|---|
| Aetox before | 16/16 | 925.791 วิ (15.4 นาที) | 62 model rounds, 4 route integration tests, README วิธีใช้, audit runtime สุดท้าย 0 advisories |
| Aetox after | 16/16 | 769.470 วิ (12.8 นาที) | 56 rounds, 2 DB/helper tests, plan report, ยังมี runtime dependency advisories |
| Codex CLI | 16/16 | **1,800.198 วิ แล้ว timeout** | ส่ง final/`turn.completed` ราว 479 วิแล้ว แต่ process ค้าง; background hook ไม่ถูกบรรจุใน build |

- Aetox after เร็วกว่าก่อนประมาณ 156 วิในคู่นี้ แต่ไม่ใช่การทดลอง timing แยกผลของการเพิ่ม/ลด test coverage
- before เพิ่มเทสต์ที่เรียก route จริง ตรวจ auth/admin/overlap/check-in/ownership/manual expire
  แรก build ล้ม type narrowing และเทสต์ล้ม alias resolution; แก้แล้ว **4/4** และ build ผ่าน
  ตรวจ persistence ด้วย stop/start และเลือกแก้ PostCSS ผ่าน override
  runtime audit สุดท้ายไม่พบ advisory; เหลือสอง moderate ใน dev dependencies ของ Vitest
- after ตั้งชื่อไฟล์ `tests/api.test.ts` แต่ **สองเทสต์นั้นเป็น DB/direct-helper tests ไม่ได้เรียก API handlers**
  แรก test runner ล้ม CJS top-level await, build ล้ม SQLite lock, แล้วเทสต์ expire นับสองแทนหนึ่ง
  แก้แล้วผ่าน **2/2** และ build สำเร็จสองครั้ง; HTTP smoke ยืนยัน no-token กับโหลดห้องด้วย token
  ไม่สรุปว่า coverage ดีกว่ารุ่นเก่าจาก hidden 16/16 เท่ากัน
- after ไม่เพิ่มวิธีเริ่ม/ดูแลใน README ซึ่งยังเหลือหนึ่งบรรทัดเดิม
  มีรายละเอียดผลตรวจในการ์ดชิ้นงานจริง แต่ **stored plan report ไม่ใช่เอกสารที่ทีมใน repo จะได้อ่านแทน README**
  บันทึกเป็น maintenance/documentation gap ไม่เพิ่ม hidden penalty ที่โจทย์ไม่ได้ระบุย้อนหลัง
- diagnostics ทั้ง before/after: 1 file checked, 0 problems และ **12 NOT checked**
  `ok=1` ไม่ใช่การตรวจสะอาดทั้งแอป; ทั้งสามเซลล์ไม่มี recorded browser candidate
- Aetox before/after มี `src/instrumentation.ts` ตรง discovery root และมี
  `.next/server/instrumentation.js` ในทั้ง existing build และ artifact capture ก่อน scorer
  expiration helper ทั้งคู่มี audit ราย booking ใน transaction; periodic wakeup ยังไม่ทดสอบจริง

**CLI Luna พบข้อกำหนดที่ suite เดิมพลาด**:

- วาง app ใน `src/app/` แต่ hook ไว้ **`instrumentation.ts` ที่ project root** ไม่ใช่ `src/instrumentation.ts`
- ตรวจ Next 15.5.27 ที่ติดตั้งอยู่: `dist/build/index.js:548–554` หา hook ใน parent ของ app/pages dir
  สำหรับ `src/app` จึงหาใน `src` ไม่ใช่ project root
- ทั้ง artifact capture ก่อน scorer และ existing build **ไม่มี `.next/server/instrumentation.js`**
  อีกสี่ implementation ที่ wiring ถูกมี artifact นี้; เก็บ SHA/excerpts ใน `background-hook-audit.json`
- ดังนั้น helper/manual API ใช้งานได้และ hidden 16/16 แต่ **timer ทุก 5 นาทีไม่ได้ถูกต่อเข้า production build**
  final และ README กลับบอกว่างานเบื้องหลังทำงานแล้ว เป็น completion claim เกินหลักฐานจริง
  ไม่ย้ายไฟล์ hook หรือ rebuild เพื่อช่วยแอปให้ผ่าน; ไม่เปลี่ยนคะแนน raw
- เวลา delivery ราว **479.4 วิ (8 นาที)** มาจาก retrospective file timestamps กับ native event
  ไม่ใช่ monotonic receive time และไม่ใช้แทน raw process timeout 30 นาที
  ผล correctness ไม่ถูกตัดทิ้งเพราะ process ค้าง แต่ process reliability gap ยังต้องนับ

### NX1/GLM 5.3: ทั้งสองฝั่งไม่ส่งผลงาน — ยังแยกต้นเหตุไม่ครบ

- before 245.780 วิ: `anthropic stream parse failed: unexpected end of JSON input`
  ไม่มี tool/file activity; after 262.795 วิ: client fallback ว่าได้รับคำตอบว่าง ไม่มี tool/file activity
- ทั้งคู่มี empty-reply nudge ภายใน harness; before บันทึก output **8,192 tokens** หนึ่งรอบ
  after **8,192 สองรอบ** รวม 16,384 ไม่ใช่คำตอบที่สร้างแอปสำเร็จ
  ไม่ได้มี task-cell retry/fallback จากผู้ตรวจ; internal recovery นี้เป็นพฤติกรรมของ harness ที่วัด
- ความเท่ากับ ceiling 8,192 เป็นสัญญาณให้ตรวจ output-budget/adapter เพิ่ม
  source ที่ตรึงไว้มี floor สำหรับ custom provider ที่ catalog ไม่รู้จัก
  **ยังไม่มี raw wire request/stop reason พอพิสูจน์ว่าเป็น transport ล้วนหรือ output cap แน่นอน**
- เก็บ raw **0/16** จาก setup เดียวคือไม่มี package.json และเก็บ failure เป็น end-to-end reliability result
  ไม่ใช้มันอ้างว่าฟังก์ชันแอปผิด 16 จุด หรือว่าโมเดลไม่สามารถเขียน Next.js ได้เมื่อรันสมบูรณ์
- after `status=completed` เดิมหมายถึง process จบ; `interpretation.json` แยกเป็น empty-provider-response
  ไม่ลบผลนี้ออกจากจำนวนเซลล์ที่ลอง และไม่เปลี่ยน provider/effort เพื่อให้คะแนนดูดี

**การปรับตัวเก็บหลังแปดเซลล์นี้** (`collection-amendment-2.json`):
บันทึกเวลารับ `turn.completed` แยกจาก process exit, เก็บ process hang หลัง delivery 120 วิ,
ปิดเฉพาะ Next process ที่ command line/creation time ตรง workspace ของเซลล์นั้น,
และเก็บ provider-response failure เป็นรายเซลล์แทนหยุดโจทย์อื่นทั้งหมด
prompt/fixtures/driver/model/effort ไม่เปลี่ยน; ไม่รันเซลล์เก่าซ้ำ
ทดสอบ collector แบบ offline **7/7**, audit แบบ offline **2/2**
ห้ามรวม native-delivery latency ใหม่เข้ากับ process-timeout เก่าโดยไม่แยกขอบเขต

| กลุ่ม | โมเดลที่ขอ | ฝั่งเปรียบเทียบ |
|---|---|---|
| Terra | `gpt-5.6-terra`, medium | Aetox before/after + Codex CLI |
| Luna | `gpt-6-luna`, medium | Aetox before/after + Codex CLI |
| GLM | `cointh-glm/glm-5.3`, medium | Aetox before/after เท่านั้น |
| GLM Flash | `cointh-glm/glm-5.3-flash`, medium | Aetox before/after เท่านั้น |

แต่ละกลุ่มโมเดลของโจทย์เดียวกันรันคู่/สามฝั่งพร้อมกัน
ต่างกลุ่มโมเดลรันตามลำดับเพื่อลดการแย่งเครื่องตอน Next.js install/build
จับ generation time แยกจาก external scorer time; ไม่ตีความความต่างเวลาข้ามโมเดลเป็น causal estimate

| โจทย์เดิม | สิ่งที่ต้องทำ/ตรวจ | รอบที่วางไว้ |
|---|---|---:|
| NX1 | Next.js App Router/TS, API, roles/ownership, จอง/กันชน/เช็กอิน/ยกเลิก, expire job, SQLite persistence, สองหน้า | 1 |
| E5 | โอนสต็อกข้ามคลัง, transfer pair/id, atomic refusal, CLI/history, preservation tests | 1 |
| E6 | รวม pricing ให้ cart/invoice/quote ใช้ร่วมกัน, rounding/VAT, output/error contracts | 1 |
| SEC1 | CWEval: 15 Python helpers, functionality **และ** security | 1 |
| SEC3 | CWEval: 9 JS helpers, functionality **และ** security | 1 |
| SEC2 | OWASP Python audit: 24 handlers, 12 vulnerable/12 safe, TPR และ FPR | 1 |
| SEC2S | shop API security audit: 6 planted issues/4 decoys, ห้ามแก้ source | 1 |
| NUI1/CR3/B6/D2 | งานเล็ก, review judgement, bug review, reversible migration รักษาข้อมูล | 2 |

รวมแผน **150 task cells** ไม่รวม preflight; **ไม่ได้หมายความว่าเสร็จแล้ว 150 เซลล์**
SEC1/SEC3 ใช้ Windows-compatible maps เดิม; exclusions เดิมไม่ใช่การเลือกตัดตามผลโมเดล
NX1 hidden suite เดิมมี 16 tests แต่สองข้อ UI ตรวจเพียง HTTP 200/HTML
จะไม่ใช้ 16/16 อ้างว่าหน้าจอสวย/ใช้งานจริงครบ; background scheduling/audit log ต้องตรวจ source เพิ่ม

เกณฑ์ audit เพิ่มตรึงก่อนเห็น task outputs: receipts ที่โมเดลอ่านจริง, คำสั่งตรวจจริง,
การรักษา test/source scope, completion claims, browser attempts, plan/skill/memory,
ชื่อที่แยกใน D2 เทียบสัญญาที่โจทย์บอก และ security findings/decoys ทีละข้อ
ไม่เอา hidden expectation ที่โจทย์ไม่บอกไปปนกับ executable correctness โดยไม่แจ้ง

## 5. หลักฐานและสถานะที่ยังค้าง

- รอบแรก: `output/behavior-bench/runtime-peers-20261001/`
  - `runs/*/{result.json,changes.patch,store.json,stdout.jsonl,turn-1.answer.txt}`
  - `audit-index.json`, `audits/*/audit.json`, `assistant-whole-turn.txt`
  - whole-turn rescoring แยกจาก `score-turn-1.json` เดิม
- ชุดขยาย: `output/behavior-bench/runtime-peers-expanded-20261001/`
  - `experiment.json`: input/profile/driver hashes, conditions, supplemental criteria
  - `preflight.json`: latest usable smoke runs; failures/conditions เดิมยังอยู่ใน `runs/`
  - `runner.json`, `matrix.json`, `runs/*/process.json`: แยก process/workspace ของแต่ละเซลล์
  - `audits/*/audit.json`: ledger + receipt ที่ยังอยู่ใน saved model context, diagnostics, plan report แยกจาก final
    receipt ไม่อยู่ใน snapshot ท้าย = ยังไม่ทราบ อาจถูก compact; ไม่ตีความว่าโมเดลไม่เคยเห็น
  - `audits/NX1*/background-hook-audit.json`: source/build wiring; ไม่ใช่ timer/UI runtime test
- สคริปต์ทดลองอยู่ `docs/internal/harness-ab-20261001/`; ไม่ใช่การเปลี่ยน runtime เพิ่มระหว่างทดลอง

**ยังค้าง:** ผลครบของชุดขยาย, detailed audit งานใหญ่/security, repeated-run comparison,
และข้อเสนอ harness ที่อยู่นอก foundation batch เดิม รายงานนี้ยังไม่จัดอันดับผู้ชนะ
