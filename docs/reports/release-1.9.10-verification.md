# ตรวจการปล่อยและแพ็กเกจ Aetox v1.9.10 — 6 ต.ค. 2026

## ด่านปล่อยและไฟล์จริง

- workflow `37363680606` ครั้งที่ 2 ผ่าน รวม `go vet ./...` และ `go test -timeout 20m ./...` บน Windows
  รอบแรกไม่มี hosted runner รับงาน จึงยกเลิกก่อนเริ่มขั้นใด ไม่ได้ข้ามเทสต์หรือเปลี่ยนแท็กเพื่อรันซ้ำ
- [Release v1.9.10](https://github.com/Mikedev115/Aetox/releases/tag/v1.9.10) เผยแพร่แล้ว ไม่ใช่ draft และตั้งเป็น Latest
  ซอร์สอยู่ private ส่วนแท็กสาธารณะสร้างบน `main` ของเอกสาร
- ตรวจ ed25519 ของ `checksums.txt` ด้วยกุญแจสาธารณะที่แอปใช้ผ่าน
- ตรวจ SHA-256 ของไฟล์บิ้ว 6 รายการและ exe ใน portable ผ่าน จากนั้นเทียบ digest ของ assets สาธารณะครบ 8 รายการ
  รวม checksums และลายเซ็น ตรงทุกไฟล์
- portable มี `aetox.exe` กับ `aetox-engine.exe` เท่านั้น CLI แยก zip และมี engine hash เดียวกัน
- Store ถือรุ่น `1.9.10.0` มี exe hash ตรงกับ portable ทั้งคู่ พร้อม `resources.pri`
  และไอคอน targetsize-24 unplated / targetsize-32
- checksums ที่ดาวน์โหลดจาก Release หลังเผยแพร่ตรงกับไฟล์ที่ตรวจลายเซ็น ใช้เติม Scoop ทั้งแอปและ CLI

## ขนาด

MiB = bytes / 1,048,576 วัดจากไฟล์ของ workflow ไม่ได้บิ้วหรือเปิดแอป production ในเครื่อง

| ไฟล์ | bytes | MiB |
|---|---:|---:|
| `aetox.exe` | 68,852,736 | 65.7 |
| `aetox-engine.exe` | 48,850,432 | 46.6 |
| รวม exe ของแอป | 117,703,168 | 112.3 |
| `aetox-amd64-installer.exe` | 45,726,545 | 43.6 |
| `aetox-windows-amd64-portable.zip` | 44,762,255 | 42.7 |
| `aetox-cli-setup.exe` | 29,563,626 | 28.2 |
| `aetox-cli-windows-amd64.zip` | 40,288,160 | 38.4 |
| MSIX ของ Store | 46,325,507 | 44.2 |

ตัวคูณขนาดใน BENCHMARK หารด้วย 112.25048828125 MiB ใหม่ทั้งคอลัมน์
ไม่ได้วัด RAM เวลาเปิด จำนวนเทสต์ UI หรือขนาดคู่แข่งใหม่ และไม่ได้ทดลองแก้ PowerPoint จริง

## SHA-256 ที่ใช้กับ Scoop และ Store

| ไฟล์ | SHA-256 |
|---|---|
| portable ของแอป | `60F136E242DF5932647B6FF260CD363F8ADE7CA0FDD8F3CDC55B7EE9F9A23C76` |
| portable ของ CLI | `29755B9807428ABC2329AC59689F5B6EB5C24B7993EE90939936E61DA658A2AC` |
| MSIX ของ Store | `C404076112F5EC453B48725A6D41F9B7B62F672971FA99D779C1D9539CFBA178` |

## ค้างให้เจ้าของทำ

แพ็กเกจอยู่ที่ `C:\Users\phrms\Downloads\aetox-v1.9.10-windows-amd64.msix`
ยังไม่ได้ส่ง Partner Center เพราะไม่มี secret ของ Store
ส่ง `.exe` ใหม่ให้ Defender (WDSI) ยังไม่ได้ทำ
ตัวเลขขนาดบนรีโป landing ยังไม่ได้อัปเดต: ทรีมีงานเจ้าของค้างและ branch ต่างจาก remote จึงไม่พุชทับ
