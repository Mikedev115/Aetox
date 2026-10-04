# ตามเทสต์ที่ล้มหลังแก้ slash skill — เริ่ม 1 ต.ค. ตรวจต่อ 2 ต.ค. 2026

ขอบเขต: ตาม assertion ที่ล้มและสองจุดที่ถูกหยุดด้วย timeout ในชุดทดสอบก่อนหน้า
แยกข้อบกพร่องของเทสต์ออกจากข้อบกพร่องของแอป ไม่เปลี่ยนค่าเริ่มต้นของผลิตภัณฑ์เพื่อให้เทสต์ผ่าน
ไม่บิ้ว/เปิดแอป ไม่ทดลองกับโมเดลจริง ไม่แตะฐานข้อมูลผู้ใช้ ไม่ commit หรือ push

## ผลและอายุที่มีหลักฐาน

| เรื่อง | สาเหตุ / ผลกระทบ | ประวัติที่ตรวจได้ | การแก้ |
|:--|:--|:--|:--|
| สีเอเจน | `TestTheShippedRosterIsSpreadAcrossTheColourWheel` เรียก `Chairs` ที่รวมเอเจนผู้ใช้ แต่ไม่แยก data root จึงเอาสีที่ผู้ใช้เลือกมาตัดสินชุด bundled | เทสต์นี้ไม่มี isolation ตั้งแต่ commit `a9e95e19` วันที่ 6 ก.ย. 2026 | ใช้ `isolate(t)` เดิม ไม่เปลี่ยนสีหรือโปรไฟล์ของผู้ใช้ |
| เทสต์ delegation 4 ตัว | fixture สร้าง `Config` โดยไม่เปิด helpers แต่ยังคาดหวัง `explore`, กลุ่ม HELPERS และ task tool อยู่ | เริ่มไม่ตรงกับสัญญาหลัง `23c3b130` วันที่ 1 ต.ค. 2026 เปลี่ยน helpers เป็น positive opt-in; ค่าเริ่มต้น startup ที่ resolve แล้วเป็นอีกกรณี | เปิด helpers ใน fixture ที่ทดสอบ reach และเปิดสวิตช์ในเทสต์ provider factory เท่านั้น ไม่เปลี่ยน default ใน runtime |
| `impact` / `symbol` ได้ข้อมูลไม่ครบ | gopls อ่านรีโปใหญ่ไม่ทันคำขอ hover แรก แต่คำขอ definition/references ที่ตามมาสำเร็จ จึงคืนผลบางส่วนโดยไม่ส่งต่อสาเหตุที่คำขอแรกพัง ชื่อ receiver หาย และคำขอ references ที่ล้มอาจดูเหมือนไม่มี callers | ทางคืนผลบางส่วนนี้มีตั้งแต่ `e489b6d3` วันที่ 28 ก.ค.; `impact` เริ่มใช้ตั้งแต่ `9a0a53dd` วันที่ 16 ก.ย. ไม่ใช่หลักฐานว่าเกิดอาการกับผู้ใช้จริงตั้งแต่วันนั้น | ลองซ้ำเฉพาะ hover/definition ที่ timeout อย่างละไม่เกินครั้งเดียว เมื่อคำตอบอื่นพิสูจน์ว่า server พร้อมแล้ว; คงคำขอที่ยังล้มเป็น warning และไม่รายงานจุดใช้งานเป็นจุดประกาศ หรือผลที่ไม่ได้เช็คเป็น 0 callers |
| แชทโค้ดที่พักไว้ | ทางเปิด draft ใน diff ที่ยังไม่ commit ข้าม `resolveStation` และข้ามประวัติที่บันทึกแล้ว หาก in-memory transcript ว่าง จึงเปิดได้แม้ทีม/เอเจนหาย หรือส่ง transcript เปล่าทั้งที่ฐานข้อมูลมีข้อความ | พบในงานที่ยังไม่ commit ของทรีนี้ ไม่พบทาง draft นี้ใน `HEAD`; ระบุอายุการใช้งานจริงไม่ได้ | เมื่อมีแถว sessions ให้ใช้ทางอ่านประวัติเดิมก่อน; draft ที่ไม่มีแถวต้องผ่าน station gate เดิม ไม่มี migration หรือการแก้/ลบแถว |
| WSL / rewind timeout | ชุดเดิมถูกจำกัดเวลารวมทั้งแพ็กเกจไว้ 120 วินาที และหมดเวลาตอนสองเทสต์นี้กำลังทำงาน ไม่ได้พิสูจน์ว่าสองกลไกเสีย | รันแยกได้ PASS: WSL 12.35 วินาที; rewind 6.15 วินาที | ไม่แก้ shell หรือ snapshot/undo เพิ่มเวลาชุดกว้างเป็น 10 นาทีเพื่อรันให้จบ |
| เทสต์ media download | ตรวจว่าออนไลน์ผ่านไซต์อื่น แล้วคาดหวังว่า `github.com/github.png` ต้องตอบทันเสมอ รอบเต็มอีกครั้งหมดเวลารอ headers | fixture นี้มาจาก `17a7f054` วันที่ 1 ก.ย. 2026 (ตามประวัติ rename ด้วย `--follow`) | เสิร์ฟ PNG จาก HTTP server ในเครื่อง ยังใช้ dispatcher และ media_fetch จริง และเทียบ bytes ที่บันทึกกับต้นฉบับ |
| เทสต์ shell kill | เริ่ม fixture `sleep 60` ตั้งแต่ต้นชุด coverage พอเครื่องมือก่อนหน้าใช้เวลานาน process จบเองก่อนเรียก kill จึงตรวจไม่พบคำว่า killed | fixture นี้มาจาก `68d0118f` วันที่ 28 ก.ค. 2026 (ตามประวัติ rename) | เริ่ม process ใหม่ก่อน case kill และลงทะเบียน cleanup เฉพาะ handles ที่เทสต์นี้สร้าง ไม่แก้ runtime kill |

วันที่ข้างบนคือวันที่ **โค้ดที่เปิดทางให้เกิดปัญหา** อยู่ในประวัติ git ไม่ใช่การยืนยันวันเกิดอาการครั้งแรกหรือรุ่นที่เคยปล่อย

## หลักฐานก่อนและหลังแก้

- สีและ delegation สองตัวแรกทำซ้ำได้ก่อนแก้ และผ่านหลังแก้
- รัน engine ให้จบด้วย `-timeout=10m` พบ delegation อีกสองตัวที่ fixture ยังไม่เปิด helpers และ `TestAChairAtTheCodingDeskNeedsItsTeamAndHoldsTheDesksTools` ที่ไม่ปฏิเสธทีมที่ถูกลบ
- probe ของ `impact` บนรีโปจริงได้ `engine.SendMessage` โดยไม่มี receiver; ถาม symbol ซ้ำเมื่อ gopls อุ่นแล้วได้ hover ของ `func (a *Engine) SendMessage(...)` ครบ ลบ probe ออกจากเทสต์หลังยืนยันสาเหตุ
- `TestImpactLiveRealGopls` ใช้ assertion เดิม ไม่ลดเงื่อนไขและไม่เพิ่มการ warm ในเทสต์: หลังแก้ได้ `Impact: engine.Engine.SendMessage` และ PASS ใน 35.32 วินาที
- regression ใหม่ `TestParkedCodeDraftCannotHideStoredTranscript` ก่อนแก้ได้ข้อความ `[]`; หลังแก้คืนคู่ข้อความเดิมภายใต้ session id เดิม
- regression ใหม่ตรวจ draft ที่ทีม/โปรไฟล์เอเจนถูกลบ: ก่อนแก้เปิดได้ทั้งคู่; หลังแก้ปฏิเสธและไม่เปลี่ยนแชทที่กำลังเปิด
- เคยทดลองเปิด helpers ใน fixture กลางแล้ว `TestATeamRoomIsAChatWithItsHeadAtNoDesk` ซึ่งต้องการ helpers ปิดล้ม: เอาการทดลองนั้นออกและเปิด helpers เฉพาะเทสต์ที่ต้องใช้ เทสต์ head กลับมาผ่าน ไม่ใช่การแก้ default ของแอป
- `TestCoverageMediaAndKillFixturesAreIndependent` จงใจจบ handle เก่าก่อนเริ่ม case kill แล้วตรวจว่าต้องสร้าง handle ใหม่และฆ่าได้ พร้อมตรวจ bytes ของ PNG ที่โหลดผ่าน HTTP ในเครื่อง — PASS

## เทสต์ที่เพิ่มและขอบเขตความเสี่ยง

- `internal/lsp/symbol_evidence_test.go`: timeout ที่ฟื้นเมื่อ server พร้อม, เพดาน retry, null ที่ถูกต้อง, refusal, server เงียบ, references ที่พัง และ cancellation; fake protocol ไม่ออกเน็ตจริง
- `internal/skill/impact_evidence_test.go`: ข้อมูลไม่ครบต้องแจ้งชัด ไม่ประดิษฐ์ declaration/0 callers และไม่ทิ้ง references ที่สำเร็จเมื่อเฉพาะ hover ล้ม
- `internal/engine/code_draft_boundary_test.go`: ประวัติที่บันทึกแล้วต้องไม่ถูก draft บัง และต้องตรวจทีม/เอเจนก่อนเปิด draft
- `internal/engine/tool_coverage_fixtures_test.go`: media download ไม่ต้องออกเน็ต และ kill ไม่ต้องอาศัยว่า fixture จากต้นชุดยังไม่จบ
- retry เป็นงานอ่านอย่างเดียว ใช้ deadline ของคำขอเดิม ไม่ retry งานเขียนหรือคำขอที่ถูกปฏิเสธ อาจเพิ่มเวลาสองคำขอ metadata ได้กรณีที่ยัง timeout ซ้ำ แต่มีเพดานและยกเลิกได้
- จุด draft เกี่ยวกับการแสดงประวัติ จึงแก้เฉพาะทางอ่าน ไม่ย้ายข้อมูล ไม่แก้ schema ไม่เปลี่ยนการบันทึก/ลบ และรักษางานสลับโหมดที่มีอยู่ในทรี
- ไม่พบหลักฐานว่า timeout เดิมทำให้ข้อมูล snapshot เสีย จึงไม่แก้กลไกย้อนข้อมูลตามการคาดเดา

## ผลชุดทดสอบ

- `go test ./internal/lsp -count=1 -timeout=180s` — PASS
- `go test ./internal/skill -count=1 -timeout=10m` — PASS (146.21 วินาที)
- `go test ./internal/config ./internal/bootstrap ./internal/cognitive ./internal/turn ./internal/app ./internal/subagent -count=1 -timeout=180s` — PASS ทั้ง 6 แพ็กเกจ
- เทสต์ station/draft/delegation ที่เกี่ยวข้องใน engine — PASS รวมทั้งเทสต์ draft เดิม
- `go test ./internal/lsp ./internal/skill -run 'Test(Symbol|Impact)' -skip '^TestImpactLiveRealGopls$' -count=3 -timeout=90s` — PASS 3 รอบ รวม regression เพิ่มเติมของผลที่ไม่ครบ
- `go test ./internal/engine -run '^TestEveryToolRunsThroughTheRealDispatcher$' -count=1 -timeout=180s` — PASS (15.50 วินาที) หลังแก้ fixture download/kill
- `go test ./internal/engine -run '^TestCoverageMediaAndKillFixturesAreIndependent$' -count=5 -timeout=90s` — PASS ทั้ง 5 รอบ
- `go vet ./internal/lsp ./internal/skill ./internal/engine` — PASS
- `go test ./internal/engine -count=1 -timeout=10m` — PASS ทั้งแพ็กเกจ (175.33 วินาที) หลังจำกัดการเปิด helpers และแก้ fixture download/kill

ผลนี้พิสูจน์กลไกและการทดสอบที่ระบุ ไม่ใช่คำรับรองว่าทั้งรีโปหรือพฤติกรรมโมเดลจริงไม่มีบั๊ก
