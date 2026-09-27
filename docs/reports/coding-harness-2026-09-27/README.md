# งานโค้ดจริง โมเดลเดียวกัน: Aetox · OpenCode · Codex CLI

> เอกสารบันทึกผล · วัด 27 ก.ย. 2026 · ข้อสอบ 11 โจทย์ × 2 โมเดล × 3 ตัวครอบ × **3 รอบ = 198 รัน**
>
> คำถามเดียว: **ให้โมเดลตัวเดียวกันทำงานโค้ดชิ้นเดียวกัน ตัวครอบ (harness) ไหนส่งงานที่ผ่านเทสต์ได้มากกว่า**
> ทุกโจทย์ตัดสินด้วยเทสต์ที่ซ่อนไว้ ผู้แข่งไม่เห็นเทสต์ระหว่างทำ ไม่มีคนให้คะแนนด้วยความรู้สึก
>
> ทุกช่องรัน 3 รอบ ตัวเลขใหญ่คือค่าเฉลี่ย ตัวเล็กใต้ตัวเลขคือค่าต่ำสุด–สูงสุดของ 3 รอบ
> ข้อสอบ เทสต์ และผลทุกรันอยู่ในโฟลเดอร์นี้ ตรวจซ้ำได้

---

## ไฮไลต์

**1. โมเดลเดียวกัน Aetox ผ่านเทสต์ได้มากที่สุดในเกือบทุกชุด** — บน Luna ดีที่สุด 4 จาก 6 คอลัมน์มาตรฐาน
บน Terra ดีที่สุดหรือเท่าที่สุด 5 จาก 6 (หัวข้อ 1) ที่เด่นที่สุดคือโค้ดที่ปลอดภัย (CWEval func-sec@1)
บน Luna Aetox ได้ 91.1% ส่วน OpenCode และ Codex CLI ได้ 66.7% เท่ากัน

**2. โมเดลเล็กทำงานเกินรุ่นได้บางชุด ไม่ใช่ทุกชุด** — Aetox บน Luna (รุ่นเล็ก) เทียบอีกสองตัวบน Terra (รุ่นใหญ่กว่า)

| | Aetox + Luna | OpenCode + Terra | Codex CLI + Terra |
|---|---|---|---|
| SlopCodeBench % เทสต์ผ่าน | 92.4%<br><sub>90.1–93.9</sub> | 92.8%<br><sub>91.2–93.9</sub> | 91.5%<br><sub>90.7–92.5</sub> |
| SWE-bench ML resolved (1 ข้อ) | 33%<br><sub>0–100</sub> | 33%<br><sub>0–100</sub> | 67%<br><sub>0–100</sub> |
| Web-Bench งานผ่าน | 5.7 / 6<br><sub>5.0–6.0</sub> | 5.0 / 6 | 5.0 / 6 |
| CWEval func-sec@1 | 91.1%<br><sub>80.0–100.0</sub> | 77.8%<br><sub>73.3–80.0</sub> | 73.3%<br><sub>66.7–80.0</sub> |
| SlopCodeBench checkpoint ผ่านครบ | 3.0 / 14<br><sub>2.0–4.0</sub> | 6.7 / 14<br><sub>6.0–7.0</sub> | 5.3 / 14<br><sub>4.0–7.0</sub> |
| เวลา (นาที) | 65<br><sub>62–67</sub> | 63<br><sub>58–66</sub> | 54<br><sub>49–59</sub> |
| token (ล้าน) | 6.3<br><sub>5.8–6.8</sub> | 5.1<br><sub>5.0–5.2</sub> | 3.8<br><sub>3.5–4.2</sub> |

Aetox + Luna ได้สูงกว่าอีกสองตัวบน Terra ใน CWEval (โค้ดปลอดภัย) และ Web-Bench ได้พอ ๆ กันใน SlopCodeBench %
เทสต์ผ่าน (อยู่ในช่วงแกว่งเดียวกัน) แต่ต่ำกว่าใน checkpoint ที่ผ่านครบทุกเทสต์ — บน Luna Aetox มักพลาดทีละไม่กี่เทสต์
กระจายหลาย checkpoint ส่วน SWE-bench มีบั๊กเดียว ผลแกว่ง 0–100% ทุกตัว อ่านเป็นสัญญาณอ่อน

> **แก้จากฉบับแรก (รอบเดียว):** ฉบับก่อนหน้านี้เขียนว่า Aetox + Luna ได้ SlopCodeBench สูงกว่าอีกสองตัวบน Terra
> (93.2% ต่อ 91.2% และ 91.4%) หลังรันครบ 3 รอบ ค่าเฉลี่ยคือ 92.4% ต่อ 92.8% และ 91.5%
> — ชัดว่าเท่ากับรุ่นใหญ่ แต่ไม่ได้ชนะ ฉบับแรกยังเขียนว่าบน Luna มีแค่ Aetox ที่แก้บั๊ก Caddy ได้ — ครบ 3 รอบ OpenCode แก้ได้ 2 ครั้ง Aetox 1 ครั้ง

**3. ราคาที่จ่าย** — Aetox ใช้เวลาและ token มากที่สุดทั้งสองโมเดล Codex CLI น้อยที่สุด (หัวข้อ 1)

---

## 1. ผลคะแนน (ค่าเฉลี่ย 3 รอบ)

แต่ละคอลัมน์ใช้ตัวชี้วัดของชุดข้อสอบนั้น · ตัวหนา = ดีที่สุดของคอลัมน์ในโมเดลเดียวกัน · ตัวเล็ก = ต่ำสุด–สูงสุด

### GPT-6 Luna · ความคิด medium

| ตัวครอบ | SlopCodeBench<br>% เทสต์ผ่าน | SlopCodeBench<br>checkpoint ผ่านครบ | SWE-bench ML<br>resolved (1 ข้อ) | Web-Bench<br>งานผ่าน | CWEval<br>func-sec@1 | CWEval<br>func@1 | โจทย์ไทย \* | เวลา<br>(นาที) | token<br>(ล้าน) |
|---|---|---|---|---|---|---|---|---|---|
| Aetox 1.9.0 | **92.4%**<br><sub>90.1–93.9</sub> | 3.0 / 14<br><sub>2.0–4.0</sub> | 33%<br><sub>0–100</sub> | **5.7 / 6**<br><sub>5.0–6.0</sub> | **91.1%**<br><sub>80.0–100.0</sub> | **97.8%**<br><sub>93.3–100.0</sub> | **91.1%**<br><sub>73.3–100.0</sub> | 65<br><sub>62–67</sub> | 6.3<br><sub>5.8–6.8</sub> |
| OpenCode 1.18.32 | 87.4%<br><sub>83.5–89.9</sub> | **3.7 / 14**<br><sub>2.0–5.0</sub> | **67%**<br><sub>0–100</sub> | 4.3 / 6<br><sub>3.0–5.0</sub> | 66.7% | 86.7% | 88.9%<br><sub>71.1–97.8</sub> | 47<br><sub>39–56</sub> | 3.4<br><sub>3.1–3.7</sub> |
| Codex CLI 0.157.1 | 80.5%<br><sub>73.4–84.2</sub> | 2.0 / 14<br><sub>1.0–3.0</sub> | 33%<br><sub>0–100</sub> | 4.3 / 6<br><sub>2.0–6.0</sub> | 66.7% | 88.9%<br><sub>86.7–93.3</sub> | 88.9%<br><sub>71.1–100.0</sub> | 32<br><sub>27–41</sub> | 2.1<br><sub>2.0–2.1</sub> |

### GPT-5.6 Terra · ความคิด medium

| ตัวครอบ | SlopCodeBench<br>% เทสต์ผ่าน | SlopCodeBench<br>checkpoint ผ่านครบ | SWE-bench ML<br>resolved (1 ข้อ) | Web-Bench<br>งานผ่าน | CWEval<br>func-sec@1 | CWEval<br>func@1 | โจทย์ไทย \* | เวลา<br>(นาที) | token<br>(ล้าน) |
|---|---|---|---|---|---|---|---|---|---|
| Aetox 1.9.0 | **94.3%**<br><sub>93.5–94.8</sub> | 6.3 / 14<br><sub>5.0–7.0</sub> | **67%**<br><sub>0–100</sub> | **5.7 / 6**<br><sub>5.0–6.0</sub> | **93.3%** | **93.3%** | **95.6%**<br><sub>93.3–100.0</sub> | 88<br><sub>78–96</sub> | 9.4<br><sub>9.0–9.8</sub> |
| OpenCode 1.18.32 | 92.8%<br><sub>91.2–93.9</sub> | **6.7 / 14**<br><sub>6.0–7.0</sub> | 33%<br><sub>0–100</sub> | 5.0 / 6 | 77.8%<br><sub>73.3–80.0</sub> | **93.3%** | 93.3%<br><sub>91.1–97.8</sub> | 63<br><sub>58–66</sub> | 5.1<br><sub>5.0–5.2</sub> |
| Codex CLI 0.157.1 | 91.5%<br><sub>90.7–92.5</sub> | 5.3 / 14<br><sub>4.0–7.0</sub> | **67%**<br><sub>0–100</sub> | 5.0 / 6 | 73.3%<br><sub>66.7–80.0</sub> | **93.3%** | 92.6%<br><sub>88.9–97.8</sub> | 54<br><sub>49–59</sub> | 3.8<br><sub>3.5–4.2</sub> |


| ตัวชี้วัด | ความหมาย |
|---|---|
| SlopCodeBench % เทสต์ผ่าน | สัดส่วนเทสต์ที่ผ่านของแต่ละโจทย์ เฉลี่ย 6 โจทย์ (A1 B1 S1–S4) ตรวจทุก checkpoint ที่ทำถึงหลังเทิร์นสุดท้าย รวมของเก่าที่อาจพัง |
| SlopCodeBench checkpoint ผ่านครบ | checkpoint ที่เทสต์ผ่านทุกข้อ จาก 14 checkpoint (A1 3 · B1 3 · S1–S4 อย่างละ 2) · checkpoint 1 ของ S4 ผ่านครบไม่ได้บน Windows (11/13 แม้ใช้เฉลยของชุดเอง) เพดานจริงจึงเป็น 13 |
| SWE-bench ML resolved | เทสต์ที่ต้องเปลี่ยนจากตกเป็นผ่าน ผ่านจริง (FAIL_TO_PASS) · มีบั๊กเดียว (caddy-6350) × 3 รอบ ค่าจึงเป็น 0 / 33 / 67 / 100% ตัวเลขนี้แกว่งมาก อ่านเป็นสัญญาณอ่อน |
| Web-Bench งานผ่าน | งาน (task-N) ที่ spec ของ Playwright ผ่านครบ หลังทำครบ 6 งาน |
| CWEval func-sec@1 | ฟังก์ชันที่ทั้งทำงานถูกและผ่านเทสต์โจมตี จาก 15 ฟังก์ชัน · func@1 = ทำงานถูกอย่างเดียว |
| โจทย์ไทย \* | เทสต์ผ่านรวม F1 + F2 (45 เทสต์) โจทย์ที่ทีม Aetox เขียนเอง และ Aetox มีสกิลภาษาไทยติดมา — อ่านแยกจากชุดมาตรฐาน |
| เวลา / token | รวมทั้ง 11 โจทย์ต่อรอบ · token = ขาเข้า (รวมส่วนที่แคช) + ขาออก ตามที่แต่ละตัวรายงาน |

---

## 2. คะแนนรายโจทย์

ตัวเลข = ค่าเฉลี่ยเทสต์ที่ผ่าน / เทสต์ทั้งหมด (โจทย์หลายเทิร์นนับรวมทุกเทิร์น) · ในวงเล็บคือรอบ 1, 2, 3
B3a นับเป็นจำนวนรอบที่แก้บั๊กได้

### GPT-6 Luna

| โจทย์ | ชุดข้อสอบ | Aetox 1.9.0 | OpenCode 1.18.32 | Codex CLI 0.157.1 |
|---|---|---|---|---|
| A1 | SlopCodeBench circuit_eval | 203.3 / 205 <sub>(203, 203, 204)</sub> | 199.3 / 205 <sub>(199, 199, 200)</sub> | 198.7 / 205 <sub>(194, 202, 200)</sub> |
| B1 | SlopCodeBench database_migration | 78.3 / 87 <sub>(84, 75, 76)</sub> | 75.7 / 87 <sub>(64, 84, 79)</sub> | 72.3 / 87 <sub>(68, 70, 79)</sub> |
| S1 | SlopCodeBench xjq | 46.7 / 51 <sub>(45, 46, 49)</sub> | 48.7 / 51 <sub>(46, 51, 49)</sub> | 27.3 / 51 <sub>(37, 3, 42)</sub> |
| S2 | SlopCodeBench sheeteval | 38.3 / 44 <sub>(39, 37, 39)</sub> | 26.3 / 44 <sub>(36, 10, 33)</sub> | 38.0 / 44 <sub>(33, 41, 40)</sub> |
| S3 | SlopCodeBench etl_pipeline | 68.0 / 73 <sub>(69, 68, 67)</sub> | 66.0 / 73 <sub>(72, 62, 64)</sub> | 63.7 / 73 <sub>(67, 57, 67)</sub> |
| S4 | SlopCodeBench code_search | 23.3 / 25 <sub>(23, 22, 25)</sub> | 23.7 / 25 <sub>(23, 25, 23)</sub> | 19.0 / 25 <sub>(23, 21, 13)</sub> |
| B3a | SWE-bench ML caddy-6350 | 1/3 รอบ | 2/3 รอบ | 1/3 รอบ |
| C1 | Web-Bench svg-solar | 22.7 / 23 <sub>(23, 22, 23)</sub> | 20.7 / 23 <sub>(19, 21, 22)</sub> | 20.0 / 23 <sub>(23, 22, 15)</sub> |
| SEC1 | CWEval Python | 13.7 / 15 <sub>(12, 14, 15)</sub> | 10.0 / 15 <sub>(10, 10, 10)</sub> | 10.0 / 15 <sub>(10, 10, 10)</sub> |
| F1 | พร้อมเพย์ QR \* | 24.0 / 28 <sub>(28, 28, 16)</sub> | 24.0 / 28 <sub>(16, 28, 28)</sub> | 24.0 / 28 <sub>(28, 16, 28)</sub> |
| F2 | ใบกำกับภาษี \* | 17.0 / 17 <sub>(17, 17, 17)</sub> | 16.0 / 17 <sub>(16, 16, 16)</sub> | 16.0 / 17 <sub>(15, 16, 17)</sub> |

### GPT-5.6 Terra

| โจทย์ | ชุดข้อสอบ | Aetox 1.9.0 | OpenCode 1.18.32 | Codex CLI 0.157.1 |
|---|---|---|---|---|
| A1 | SlopCodeBench circuit_eval | 204.3 / 205 <sub>(205, 203, 205)</sub> | 204.7 / 205 <sub>(205, 205, 204)</sub> | 196.0 / 205 <sub>(203, 192, 193)</sub> |
| B1 | SlopCodeBench database_migration | 82.7 / 87 <sub>(84, 84, 80)</sub> | 79.7 / 87 <sub>(81, 76, 82)</sub> | 83.3 / 87 <sub>(83, 84, 83)</sub> |
| S1 | SlopCodeBench xjq | 47.3 / 51 <sub>(47, 48, 47)</sub> | 48.7 / 51 <sub>(50, 50, 46)</sub> | 49.0 / 51 <sub>(50, 50, 47)</sub> |
| S2 | SlopCodeBench sheeteval | 39.0 / 44 <sub>(40, 39, 38)</sub> | 38.7 / 44 <sub>(37, 38, 41)</sub> | 37.0 / 44 <sub>(37, 37, 37)</sub> |
| S3 | SlopCodeBench etl_pipeline | 71.3 / 73 <sub>(70, 72, 72)</sub> | 69.7 / 73 <sub>(70, 70, 69)</sub> | 70.3 / 73 <sub>(70, 70, 71)</sub> |
| S4 | SlopCodeBench code_search | 23.0 / 25 <sub>(23, 23, 23)</sub> | 21.7 / 25 <sub>(19, 23, 23)</sub> | 20.3 / 25 <sub>(19, 19, 23)</sub> |
| B3a | SWE-bench ML caddy-6350 | 2/3 รอบ | 1/3 รอบ | 2/3 รอบ |
| C1 | Web-Bench svg-solar | 22.7 / 23 <sub>(23, 23, 22)</sub> | 21.7 / 23 <sub>(22, 21, 22)</sub> | 21.7 / 23 <sub>(22, 21, 22)</sub> |
| SEC1 | CWEval Python | 14.0 / 15 <sub>(14, 14, 14)</sub> | 11.7 / 15 <sub>(12, 11, 12)</sub> | 11.0 / 15 <sub>(11, 10, 12)</sub> |
| F1 | พร้อมเพย์ QR \* | 28.0 / 28 <sub>(28, 28, 28)</sub> | 28.0 / 28 <sub>(28, 28, 28)</sub> | 27.0 / 28 <sub>(25, 28, 28)</sub> |
| F2 | ใบกำกับภาษี \* | 15.0 / 17 <sub>(14, 17, 14)</sub> | 14.0 / 17 <sub>(13, 16, 13)</sub> | 14.7 / 17 <sub>(16, 12, 16)</sub> |


\* F1 และ F2 เป็นโจทย์ที่ทีม Aetox เขียนเอง (เรื่องไทยที่ไม่มีชุดมาตรฐาน) และ Aetox มีสกิลภาษาไทยติดมากับตัว
จึงแยกเป็นคอลัมน์ของตัวเองในหัวข้อ 1 ไม่ปนกับชุดมาตรฐาน

ความแกว่งที่เห็นชัดที่สุดคือโจทย์หลายเทิร์น เช่น B1 บน Luna ที่รอบแรก Aetox 84 / OpenCode 64 แต่รอบสอง Aetox 75 / OpenCode 84
— เหตุผลที่ต้องรันซ้ำก่อนสรุป ผลรายรันทั้งหมด (เวลา token จำนวนครั้งที่เรียกเครื่องมือ) อยู่ใน [results.csv](results.csv)

---

## 3. วัดอย่างไร

| เรื่อง | ค่าที่ใช้ |
|---|---|
| โมเดล | `gpt-6-luna` และ `gpt-5.6-terra` ความคิดระดับ medium ผ่านบัญชี ChatGPT (Codex) บัญชีเดียวกันทั้งสามตัว |
| Aetox | 1.9.0 ตัวเทอร์มินัลจาก Release ([`aetox-cli-windows-amd64.zip`](https://github.com/Mikedev115/Aetox/releases/tag/v1.9.0) sha256 `9881e62f…c2d4`) ไม่ได้บิวด์เอง · สั่งงานแบบ `aetox chat` ครั้งละหนึ่งข้อความ · อนุมัติเต็ม · โฟลเดอร์ข้อมูลใหม่ทุกรัน · ไม่มีสกิลหรือการตั้งค่าของผู้ใช้ มีแค่ของที่ติดมากับตัวติดตั้ง |
| OpenCode | 1.18.32 · `opencode run --auto` · โฟลเดอร์ตั้งค่าแยกที่มีแค่ไฟล์ล็อกอิน |
| Codex CLI | 0.157.1 · `codex exec --ephemeral` แบบอนุมัติเต็ม · `CODEX_HOME` แยกที่มีแค่ไฟล์ล็อกอิน ไม่มี `AGENTS.md` ปลั๊กอิน หรือการตั้งค่าของผู้ใช้ ปิดค้นเว็บ |
| ข้อความที่ส่ง | ไฟล์เดียวกันทุกไบต์ทั้งสามตัว ([`tasks/*/prompts/`](tasks)) · โจทย์หลายเทิร์นส่งทีละเทิร์นตามลำดับ ไม่มีข้อความอื่นแทรก ไม่มีคนตอบคำถามกลางทาง |
| ที่ทำงาน | ทุกรันเริ่มจาก git repo ใหม่ที่มีแค่ไฟล์ตั้งต้นของโจทย์ ([`tasks/*/start/`](tasks)) |
| เทสต์ | คัดลอกเข้าไปตรวจหลังตัวครอบจบเทิร์นแล้วเท่านั้น ([`tasks/*/hidden-tests/`](tasks)) |
| เวลาจำกัด | 30 นาทีต่อเทิร์น (ไม่มีรันไหนชนเพดาน) |
| รอบ | 3 รอบต่อช่อง รอบ 1 รันพร้อมกัน 3 รัน · รอบ 2 และ 3 รันคู่ขนานกันรวม 6 รัน บน Windows 11 เครื่องเดียว |

**ความยุติธรรมที่ควรรู้**

- ทั้งสามตัวใช้โมเดล บัญชี ระดับความคิด ข้อความ และไฟล์ตั้งต้นเดียวกัน ต่างกันแค่ตัวครอบ
- Aetox เปิดสกิลที่ติดมากับตัวติดตั้งตามที่ผู้ใช้ได้จริง OpenCode ไม่มีสกิลติดมา Codex ถูกปิดสกิลระบบ ปลั๊กอิน และค้นเว็บ
  ให้เหลือแค่ตัวครอบกับล็อกอิน เหมือนที่ OpenCode ได้
- รันที่ล้มเพราะโครงสร้าง ไม่ใช่เพราะตัวแข่ง ถูกตัดแล้วรันใหม่ด้วยกฎเดียวกันทุกฝั่ง: Codex รอบแรก 3 รัน (ล็อกอินชนกับแอป Codex
  บนเครื่อง `refresh_token_reused`) และ 7 รันที่เน็ตหลุดช่วง 06:33–06:40 (Aetox 3 · OpenCode 3 · Codex 1 — ลายเซ็น
  `Chat failed` / `Cannot connect to API` / `waiting for network`) ของเดิมเก็บไว้ใน `_invalid/` ของผลดิบ
- มี 1 รัน (S1 · Codex · Luna · รอบ 2) ที่ส่งคำตอบแล้วแต่ process ไม่ปิดเอง ตัวรันปิดให้หลังรอ 2 นาที คะแนนนับจากงานที่ส่งจริง
- เวลาวัดขณะรันพร้อมกันหลายรัน ใช้เทียบกันได้ แต่ตัวเลขเดี่ยวจะช้ากว่ารันทีละรัน

---

## 4. ข้อสอบ

ใช้ชุดมาตรฐานสาธารณะ 9 โจทย์ และโจทย์ที่เขียนเอง 2 โจทย์ ข้อสอบทุกข้อคัดลอกมาไว้ใน [`tasks/`](tasks)
พร้อมสัญญาอนุญาตของต้นทาง แต่ละโฟลเดอร์มี `prompts/` (ข้อความที่ส่งจริง) `start/` (ไฟล์ตั้งต้น) และ `hidden-tests/` (เทสต์ที่ใช้ให้คะแนน)

| โจทย์ | ชุด | ต้นทาง | สัญญาอนุญาต | ทำอะไร | เทิร์น |
|---|---|---|---|---|---|
| A1 | SlopCodeBench `circuit_eval` | [gabeorlanski/scb-problems @ 38d627ec](https://github.com/gabeorlanski/scb-problems/tree/38d627ecf668a88f88f8d260f8df8df6116e9b03/circuit_eval) | Apache-2.0 | ตัวประเมินวงจรตรรกะ ต่อยอดทีละขั้น | 3 |
| B1 | SlopCodeBench `database_migration` | [scb-problems @ 38d627ec](https://github.com/gabeorlanski/scb-problems/tree/38d627ecf668a88f88f8d260f8df8df6116e9b03/database_migration) | Apache-2.0 | เครื่องมือ migrate ฐานข้อมูล ต่อยอดทีละขั้น | 3 |
| S1 | SlopCodeBench `xjq` | [scb-problems @ 38d627ec](https://github.com/gabeorlanski/scb-problems/tree/38d627ecf668a88f88f8d260f8df8df6116e9b03/xjq) | Apache-2.0 | เครื่องมือค้นและแปลงข้อมูลแบบ jq | 2 |
| S2 | SlopCodeBench `sheeteval` | [scb-problems @ 38d627ec](https://github.com/gabeorlanski/scb-problems/tree/38d627ecf668a88f88f8d260f8df8df6116e9b03/sheeteval) | Apache-2.0 | ตัวคำนวณสูตรสเปรดชีต | 2 |
| S3 | SlopCodeBench `etl_pipeline` | [scb-problems @ 38d627ec](https://github.com/gabeorlanski/scb-problems/tree/38d627ecf668a88f88f8d260f8df8df6116e9b03/etl_pipeline) | Apache-2.0 | ไปป์ไลน์ ETL | 2 |
| S4 | SlopCodeBench `code_search` | [scb-problems @ 38d627ec](https://github.com/gabeorlanski/scb-problems/tree/38d627ecf668a88f88f8d260f8df8df6116e9b03/code_search) | Apache-2.0 | เครื่องมือค้นโค้ด | 2 |
| B3a | SWE-bench Multilingual `caddyserver__caddy-6350` | [SWE-bench/SWE-bench_Multilingual](https://huggingface.co/datasets/SWE-bench/SWE-bench_Multilingual) · รีโป [caddyserver/caddy @ a52917a3](https://github.com/caddyserver/caddy/tree/a52917a37dcc40eda1ff5034103d4a89883de2aa) | MIT (ชุดข้อมูล) · Apache-2.0 (Caddy) | แก้บั๊กจริงในรีโป Go ขนาดใหญ่ | 1 |
| C1 | Web-Bench `svg-solar` งาน 1–6 | [bytedance/web-bench @ 7b31ca2b](https://github.com/bytedance/web-bench/tree/7b31ca2b786eef120dd49ce63dd03d0c0006046d/projects/svg-solar) | Apache-2.0 | หน้าเว็บระบบสุริยะ SVG ต่อยอด 6 งาน ตรวจด้วย Playwright | 6 |
| SEC1 | CWEval Python 15 ข้อ | [Co1lin/CWEval @ e9a2a124](https://github.com/Co1lin/CWEval/tree/e9a2a124c8c53679b6d8d27adfd2f6c40e7576d7/benchmark/core/py) | Apache-2.0 | เติมฟังก์ชันช่วยของเว็บเซอร์วิส นับเป็นผ่านเมื่อทั้งทำงานถูกและทนการโจมตี | 1 |
| F1 | เขียนเอง | [`tasks/F1-promptpay`](tasks/F1-promptpay) | – | ข้อความ QR พร้อมเพย์ตามมาตรฐาน EMVCo พร้อม CRC | 1 |
| F2 | เขียนเอง | [`tasks/F2-thai-invoice`](tasks/F2-thai-invoice) | – | ใบกำกับภาษี ตัวเลขเป็นคำอ่านบาท VAT หัก ณ ที่จ่าย | 1 |

SEC1 ตัดข้อ `cwe_1333_0` ออก เพราะเทสต์ความปลอดภัยของข้อนั้นใช้ไม่ได้บน Windows แม้กับเฉลยของ CWEval เอง
B3a ไม่ได้คัดลอกรีโป Caddy มา ใช้ commit ต้นทางตามลิงก์ ไฟล์ [`instance.json`](tasks/B3a-caddy-6350/instance.json) มีคำอธิบายบั๊ก
และ [`test_patch.diff`](tasks/B3a-caddy-6350/hidden-tests/test_patch.diff) คือเทสต์ที่ใช้ตัดสิน (ไม่ได้ใส่เฉลยของชุดข้อมูล)

---

## 5. ไฟล์ในโฟลเดอร์นี้

| ไฟล์ | คืออะไร |
|---|---|
| `README.md` | เอกสารนี้ |
| [`results.csv`](results.csv) | ผลรายรันทั้ง 198 รัน (คอลัมน์ `run` = รอบ): เทสต์ผ่าน เวลา token จำนวนครั้งที่เรียกเครื่องมือ รุ่นของตัวครอบ |
| [`tasks/`](tasks) | ข้อสอบทั้ง 11 โจทย์ พร้อม LICENSE ของต้นทาง |

รอบถัดไปที่กำลังรัน: ชุดที่ยากกว่า (SlopCodeBench จนจบทุก checkpoint · SWE-bench Verified ข้อยาก 45 ข้อ ·
Terminal-Bench 2.0 ข้อยาก 30 ข้อ · Aider Polyglot) บน GPT-6 Luna · GPT-5.6 Terra · GPT-6 Sol
