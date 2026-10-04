# ตรวจการปล่อยและแพ็กเกจ Aetox v1.9.4 — 30 ก.ย. 2026

## รุ่นและด่านปล่อย

- ซอร์ส: `71f7666f683053a1d1563e24d351324d77abb2ee` ในรีโป private
- workflow ที่สร้างไฟล์: `36727940092` ผ่าน รวม `go vet ./...` และ `go test -timeout 20m ./...` บน Windows
- รุ่นเผยแพร่: <https://github.com/Mikedev115/Aetox/releases/tag/v1.9.4> เป็น Latest และไม่ใช่ draft
- แท็กสาธารณะชี้คอมมิตเอกสาร `fc5e4646f3b68239cd0359b90d7fdc839aea5fb1` ซอร์สอยู่ private ตามเดิม
- ไฟล์ Release สาธารณะครบ 8 รายการ ตรวจ state ว่าอัปโหลดแล้วและ digest ตรงกับไฟล์ที่ตรวจในเครื่อง

รอบแรก `36721540704` ล้มที่ `TestGitWorktreesListAddAndBranchMark`, `TestEveryExecSiteHidesTheConsole`
และ `TestEverySpawnCanBeStopped` แก้การเทียบพาธ Windows ชื่อเต็ม/8.3 ด้วยตัวตนไฟล์จริง และบอกอายุโปรเซส
เซิร์ฟเวอร์โมเดลที่จุดสร้างคำสั่ง พร้อมซ่อนหน้าต่าง Ollama ที่จุดนั้น สามเทสต์ผ่านบนซอร์สสะอาด
ก่อนย้ายแท็ก private ตามคำอนุญาตเจ้าของ และด่านเต็มบน workflow รอบใหม่ผ่าน

## ตรวจไฟล์จริง

- ลายเซ็น ed25519 ของ `checksums.txt` ผ่านกับกุญแจสาธารณะที่ฝังในแอป
- SHA-256 ของไฟล์ดาวน์โหลดทุกตัวตรงกับ `checksums.txt` และ digest ที่ GitHub รายงาน
- portable มี exe สองตัว และ CLI เป็น zip แยกของตัวเอง
- MSIX ถือเลข `1.9.4.0` มี exe สองตัวที่ hash ตรงกับ portable และมี `resources.pri` กับไอคอนขนาดที่ต้องใช้
- hash ของ Scoop ทั้งแอปและ CLI เติมจาก `checksums.txt` ที่ดาวน์โหลดจาก Release สาธารณะหลังเผยแพร่

## ขนาด

MiB = bytes / 1,048,576 ขนาดวัดจากไฟล์ที่ workflow สร้าง ไม่ได้บิ้วแอปแยกในเครื่อง

| ไฟล์ | bytes | MiB |
|---|---:|---:|
| `aetox.exe` | 58,240,000 | 55.5 |
| `aetox-engine.exe` | 37,425,152 | 35.7 |
| รวม exe ของแอป | 95,665,152 | 91.2 |
| `aetox-amd64-installer.exe` | 38,775,372 | 37.0 |
| `aetox-windows-amd64-portable.zip` | 37,732,066 | 36.0 |
| `aetox-cli-setup.exe` | 23,717,797 | 22.6 |
| `aetox-cli-windows-amd64.zip` | 32,629,492 | 31.1 |
| `aetox-windows-amd64.msix` | 39,098,295 | 37.3 |

## SHA-256

| ไฟล์ | SHA-256 |
|---|---|
| `aetox-amd64-installer.exe` | `5DB7789D481572FCF413A72A91EFB893EF31E4EB407B7210AE3C54F4F5FC1457` |
| `aetox-windows-amd64-portable.zip` | `953F99A1281DB32AD64B2064C06FE0C32BA0CF32797596BA7DB484AEEA158D15` |
| `aetox-cli-windows-amd64.zip` | `B0173986699C286544E41484E12AF08422BB5780489F122CFCA09121454465A7` |
| `aetox-cli-setup.exe` | `F4CE7DF154AD933316EA3C5B2832A7D6A41FF8867302D9384A4D1DC941F76A3E` |
| `aetox.exe` | `807815CE96996998DBD10BAD1FFBDF126B7401267A2AE7A12B92393F9F60C0B9` |
| `aetox-engine.exe` | `2B99959CC54F69F9E3B27D935F3570242ADB5351F63EAEB07C370F07AECF36E5` |
| `aetox-engine-linux-amd64` | `665DF0CECFAB2A13B1FA75E25E437530FCF2E3345863311784B5E7CA2848E06E` |
| `aetox-engine-linux-arm64` | `19879BBD99E6B7A25B3306C61DA3453E9BDDA60D233013BF1B29E57B0E5F4CBA` |
| MSIX ของ Store | `763F124F4EF0A15FCE093E857FC3A97FC040415D8CB98A6E482750839CF69879` |

## ขอบเขตผล

workflow รุ่นนี้ตรวจชุด Go และสร้าง frontend จริง การตรวจนี้ไม่ได้รันชุด Vitest เต็มหรือวัดจำนวนเทสต์ UI ใหม่
ไม่ได้เปิดแอป production เพื่อวัด RAM หรือเวลาเปิด และ Linux เป็นการตรวจคอมไพล์ข้ามสถาปัตยกรรม
ผลทดลองคุณภาพงานโค้ด 120 รันอยู่ใน รายงานแยก (เอกสารภายใน ไม่เผยแพร่แล้ว) และไม่ใช่การวัดทั้งรุ่นนี้

## Microsoft Store

จัดไฟล์ไว้ใน `Downloads/Aetox-v1.9.4/aetox-windows-amd64.msix` ให้เจ้าของอัปผ่าน Partner Center
ยังไม่ได้ส่งเข้า Store หรือเริ่มกระบวนการรับรอง เพราะเจ้าของเลือกอัปเองและยังไม่มีค่าการเชื่อมต่ออัตโนมัติ
