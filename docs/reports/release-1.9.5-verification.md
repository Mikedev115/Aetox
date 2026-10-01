# ตรวจการปล่อยและแพ็กเกจ Aetox v1.9.5 — 2 ต.ค. 2026

## รุ่นและด่านปล่อย

- ซอร์ส: `79e34bb8134a1469ed1eff96a160ef2c2f6dfa98` ในรีโป private
- workflow `36908337891` ครั้งที่ 2 ผ่าน รวม `go vet ./...` และ `go test -timeout 20m ./...` บน Windows ใช้ 39 นาที 41 วินาที
- รุ่นเผยแพร่: <https://github.com/Mikedev115/Aetox/releases/tag/v1.9.5> เป็น Latest และไม่ใช่ draft
- แท็กสาธารณะชี้คอมมิตเอกสาร `fd72204d90d55041e26823591433347b7bcebfd4` ซอร์สอยู่ private ตามเดิม
- assets สาธารณะครบ 8 รายการ state เป็น uploaded ขนาดและ SHA-256 digest ตรงกับไฟล์ที่ตรวจ

รอบแรก `36904008981` หยุดที่ `TestDeskTerminalUsesCallingConversation` ซึ่งเทียบพาธชื่อสั้น/เต็ม
ด้วยตัวอักษร และ `TestUnsupportedAndMissingServersAreSilent` ซึ่งสมมติว่า runner ไม่มี rust-analyzer
แก้ fixture และการเทียบตัวตนโฟลเดอร์แล้วทดสอบเฉพาะจุดก่อนขยับแท็ก private ตามคำอนุญาตเจ้าของ
workflow ใหม่ครั้งที่ 1 ขาดการติดต่อกับ runner ไม่มี log ให้ยืนยันสาเหตุ จึงรันซ้ำบนคอมมิตเดิม
ครั้งที่ 2 ผ่านครบ ไม่ข้ามเทสต์หรือเผยแพร่ไฟล์จากรอบที่แดง

## ตรวจไฟล์จริง

- ed25519 ของ `checksums.txt` ผ่านกับกุญแจสาธารณะใน `internal/update/signing.go`
- SHA-256 ของไฟล์ Release ทุกตัวตรงกับ checksums และ digest ที่ GitHub รายงาน
- portable มี `aetox.exe` กับ `aetox-engine.exe` เท่านั้น CLI เป็น zip แยกและมี engine hash เดียวกัน
- MSIX ถือเลข `1.9.5.0` identity `AetoxAI.Aetox` มี exe สองตัวที่ hash ตรงกับ portable
- MSIX มี `resources.pri`, ไอคอน targetsize-24 unplated และ targetsize-32 ครบ
- Scoop ทั้งแอปและ CLI ใช้ hash จาก checksums ที่ดาวน์โหลดและตรวจลายเซ็นจาก Release สาธารณะหลังเผยแพร่

## ขนาด

MiB = bytes / 1,048,576 วัดจาก artifact ที่ CI สร้าง ไม่ได้บิ้วหรือเปิดแอป production ในเครื่อง

| ไฟล์ | bytes | MiB |
|---|---:|---:|
| `aetox.exe` | 58,488,320 | 55.8 |
| `aetox-engine.exe` | 37,515,264 | 35.8 |
| รวม exe ของแอป | 96,003,584 | 91.6 |
| `aetox-amd64-installer.exe` | 38,861,777 | 37.1 |
| `aetox-windows-amd64-portable.zip` | 37,821,832 | 36.1 |
| `aetox-cli-setup.exe` | 23,757,782 | 22.7 |
| `aetox-cli-windows-amd64.zip` | 32,688,172 | 31.2 |
| `aetox-windows-amd64.msix` | 39,178,275 | 37.4 |

## SHA-256

| ไฟล์ | SHA-256 |
|---|---|
| `aetox-amd64-installer.exe` | `99EB12C76812933870295242CCA5D235DA797B4EF752FA9838616504800C22BC` |
| `aetox-windows-amd64-portable.zip` | `7D6E90F895742378BEA43966B5E9BB03FDA7B31FE5351B78D4F23726DC5D811D` |
| `aetox-cli-windows-amd64.zip` | `FF152B21B45C94655D281E517A9CA846BE7AFE075797EF73A1EF7551B61AA0C6` |
| `aetox-cli-setup.exe` | `9D0B6706D63BA6B2AA9E46CC722D596CE07D73AA48A329D9B10CC132FFE58188` |
| `aetox.exe` | `79629F5E96794A753F8C99BD776DD2A565AA37DF3CA01DC15A36775B5879CEC6` |
| `aetox-engine.exe` | `23E6F44FDC8FE3F7754179F5640B73BA61CC2680B6E048000EE17D91351572EF` |
| `aetox-engine-linux-amd64` | `6C4464418679B9B79B1F45E601AF834366933D62E044B3FB90657FCDE7C38954` |
| `aetox-engine-linux-arm64` | `0A75352BBC6A135C8E993913593E2331EB216C0CF7FA858B09EF0C96836D203D` |
| MSIX ของ Store | `1CE091CFA6DC60EAC03820D75ECFA9A7697A8D2DF042FD7D00F6511C93B4598F` |

## ขอบเขตผลและงานที่ยังค้าง

workflow ตรวจชุด Go และสร้าง frontend จริง ไม่ได้รัน Vitest เต็มหรือวัดจำนวนเทสต์ UI ใหม่
ไม่ได้วัด RAM เวลาเปิด หรือขนาดคู่แข่งใหม่ Linux เป็นการคอมไพล์ข้ามสถาปัตยกรรม

ไฟล์ Store อยู่ที่ `C:\Users\phrms\Downloads\Aetox-v1.9.5\aetox-windows-amd64.msix` ให้เจ้าของอัปผ่าน
Partner Center ยังไม่ได้ส่งเข้ารับรอง และการตั้งส่ง Store อัตโนมัติพักไว้ตามคำเจ้าของ
การส่ง zip และตัวติดตั้งให้ Microsoft Defender ตรวจยังค้าง ไม่ถือว่าผ่านการรับรองจาก Microsoft
