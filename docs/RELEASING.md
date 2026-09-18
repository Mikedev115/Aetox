# ปล่อยรุ่นใหม่ — ลำดับที่ต้องเดินให้ครบ

> เอกสารนี้ตอบคำถามเดียว: **"บั๊มป์เวอร์ชันแล้วต้องทำอะไรต่อบ้าง กว่าจะเรียกว่าปล่อยเสร็จ"**
>
> ตัวเลขที่ต้องวัดใหม่หลังปล่อยอยู่ที่ [PUBLISHED-NUMBERS.md](PUBLISHED-NUMBERS.md) ·
> เหตุผลว่าทำไม scoop hash ถึงค้างหนึ่งรุ่นอยู่ที่ [ARCHITECTURE.md](../ARCHITECTURE.md) §58 ·
> ไฟล์ไหนถือเวอร์ชันบ้างอยู่ในคอมเมนต์ของ [internal/version/version.go](../internal/version/version.go)
> ที่นี่ไม่คัดของพวกนั้นมาเขียนซ้ำ

**เขียนขึ้นเพราะขั้นที่ 7 ไม่เคยถูกเขียนไว้ที่ไหนเลย** — งานบิ้วจบ ป้าย Latest ขึ้นแล้ว
ทุกอย่างดูเหมือนเสร็จ แต่คนที่ลงผ่าน Microsoft Store ยังได้รุ่นเก่าอยู่ และไม่มีอะไรบนหน้าจอ
บอกว่าขาดอะไรไป ช่องทาง Store เป็นช่องทางเดียวที่ลงแล้วไม่เจอจอเตือน SmartScreen เลย
เพราะไมโครซอฟท์เซ็นแพ็กเกจให้เอง ([บันทึกรุ่น v1.5.7](release-notes/v1.5.7.md))
การลืมมันคือการทิ้งผู้ใช้กลุ่มที่เจออุปสรรคน้อยที่สุดไว้ที่รุ่นเก่า

---

## หนึ่งรุ่น = สองไฟล์ exe เสมอ

ตั้งแต่ §248 (13 ก.ย. 2026) โปรแกรมบน Windows คือสองไฟล์ที่อยู่ข้างกัน:
`aetox.exe` (หน้าต่าง) และ `aetox-engine.exe` (สมอง — โมเดล เครื่องมือ ฐานข้อมูล)
ทั้งคู่บิ้วจาก **คอมมิตเดียวกัน ถือเลขรุ่นเดียวกัน** และคุยกันผ่านซ็อกเก็ตที่ไม่สัญญาว่า
รุ่นต่างกันจะเข้าใจกัน — ดังนั้น **ไม่มีทางอัปแค่ไฟล์เดียว** และคุณไม่ต้องอัปแยกเอง:

**กฎของโฟลเดอร์ `desktop/build/bin`: มี executable ตัวโปรแกรมจริงแค่สองชื่อด้านบนเท่านั้น**
ห้ามบิ้วไฟล์ทดลองหรือไฟล์วินิจฉัยเพิ่มด้วยชื่อใหม่ เช่น `aetox-divider-test.exe` เพราะทำให้สับสนว่า
ตัวไหนคือของจริง ถ้าจะทดสอบหน้าจอให้ใช้ `wails dev`; ถ้าจำเป็นต้องตรวจ production bundle ให้บิ้วทับ
`aetox.exe` ชื่อมาตรฐาน แล้วลบผลบิ้วในเครื่องเมื่อจบ

ไฟล์ติดตั้งให้ workflow รุ่นจริงสร้างเพียงสองช่องทางมาตรฐาน: NSIS สำหรับ GitHub และ MSIX สำหรับ
Microsoft Partner Center ห้ามสร้าง installer variant เพิ่มในเครื่อง เว้นแต่ผู้ใช้สั่งขอรูปแบบใหม่โดยตรง

| ช่องทาง | ใครวางสองไฟล์ |
|---|---|
| ตัวติดตั้ง (NSIS) | ตัวติดตั้งวางทั้งคู่ใต้ Program Files ในครั้งเดียว |
| แบบพกพา (zip) | zip มีทั้งคู่ และตัวอัปเดตในแอปสลับ **ทั้งคู่** ด้วยการ rename ทีละไฟล์ ([internal/update/apply.go](../internal/update/apply.go) `swapPortable`) — แอปที่รันอยู่ยังเป็นรุ่นเก่าจนเปิดใหม่ แล้วค่อยเป็นรุ่นใหม่พร้อมกัน |
| Microsoft Store (.msix) | แพ็กเกจเดียวมีทั้งคู่ Windows สลับทั้งแพ็กเกจ |
| Scoop | manifest ชี้ zip เดียวกัน |

สิ่งที่ workflow ตรวจให้ก่อนจะปล่อย: `aetox-engine.exe` ต้องมี version block ที่บอก
`Aetox <ver>` ตรงกับ `aetox.exe` (ขั้น "the .syso did not link" ใน release.yml ตั้งแต่ 1.7.2)
และ `checksums.txt` มีแฮชของ **แต่ละ exe แยกบรรทัด** ไม่ใช่แค่ของ zip — เพื่อให้ไฟล์ที่ผู้ใช้
กู้คืนจาก Defender ตรวจกับบรรทัดที่เซ็นแล้วได้

เอนจินสำหรับเครื่องระยะไกล (`aetox-engine-linux-amd64` / `-arm64`) เป็นไฟล์ที่สามและสี่
ของรุ่นเดียวกัน แต่ **ไม่ได้อยู่ในแพ็กเกจ** — แอปดึงจากหน้า release ตาม tag ของรุ่นตัวเอง
ตอนที่ผู้ใช้ตั้งเครื่องระยะไกล ([internal/update/engine.go](../internal/update/engine.go))
ถ้ารุ่นถูกลบหรือ asset สองตัวนี้หาย ฟีเจอร์เครื่องระยะไกลของรุ่นนั้นจะตายทั้งที่แอปยังใช้ได้

---

## ลำดับ

### 1. ทรีต้องเขียวก่อน ไม่ใช่หลัง

```bash
bash ./verify.sh
```

ทุกด่านต้องเขียว ถ้า stage `test` ขึ้น FAIL แบบไม่พิมพ์ชื่อแพ็กเกจ ให้เช็คก่อนว่า
`wails.exe` รันอยู่ไหม — มันเขียน `desktop/aetox-res.syso` ทับ แล้วแพ็กเกจ `desktop`
บิ้วไม่ผ่านชั่วขณะ นั่นคือการชนกับตัวบิ้ว ไม่ใช่โค้ดพัง รันซ้ำตอนมันหยุดแล้ว

### 2. บั๊มป์เวอร์ชัน

แก้ `Current` ใน `internal/version/version.go` ก่อนที่เดียว แล้วรัน

```bash
go test ./internal/version/
```

มันจะบอกเองว่าไฟล์ไหนยังไม่ตรง (หกไฟล์ถูกบังคับ) ส่วนไฟล์ที่เจ็ด —
`docs/release-notes/v<ver>.md` — ไม่มีเทสต์จับ ต้องเขียนเอง

`scoop/aetox.json` ช่อง `hash` **ยังไม่ต้องแก้ตอนนี้** ปล่อยให้ค้างไว้ก่อน (ข้อ 8)

### 3. คอมมิตขึ้น `main` ให้เรียบร้อยก่อนจะแตะ tag

รุ่นตัดบน `main` เท่านั้น และ workflow เช็คว่า tag กับซอร์สตรงกันก่อนบิ้ว
**tag ที่ถูกพุชก่อนคอมมิตคือ tag ที่ทำให้ workflow ล้ม**

### 4. พุช main ก่อน แล้วค่อยพุช tag

```bash
git push origin main
git push origin v<ver>
```

การพุช tag คือสิ่งที่จุดชนวน `.github/workflows/release.yml`

### 5. รอ workflow จนจบ

```bash
gh run watch <run-id> --exit-status
```

ประมาณ 7-8 นาที ขั้นที่ควรเห็นเขียวคือ "Both artifacts exist",
"The suite must pass on what is about to ship" และ "Sign checksums with the release key"

### 6. เอารุ่นออกจาก draft — งานบิ้วเขียวไม่ได้แปลว่าปล่อยแล้ว

workflow สร้าง GitHub Release เป็น **draft** เสมอ (`draft: true`)
ถ้าหยุดตรงนี้ ป้าย Latest จะยังค้างอยู่ที่รุ่นก่อน และตัวเช็คอัปเดตในแอปจะมองไม่เห็นอะไรเลย

```bash
gh release edit v<ver> --draft=false --latest
```

### 7. อัปแพ็กเกจขึ้น Microsoft Store — ขั้นที่ลืมง่ายที่สุด

workflow ไม่ได้ส่งไฟล์นี้ขึ้น Store ให้ มันแค่เก็บไว้เป็น artifact
(ขั้น "Keep the Store package for Partner Center") การอัปเป็นงานมือ เพราะการกดส่ง
คือการเผยแพร่ในชื่อเจ้าของบัญชี

```bash
gh run download <run-id> -n msix-store-package -D ~/Downloads
```

แล้วเข้า https://partner.microsoft.com/dashboard →
**Apps and games** → Aetox → **Submissions** → **Packages** → ลาก `.msix` เข้าไป → Submit

เช็คก่อนอัปได้ด้วยคำสั่งเดียว ถ้าเลขไม่ตรง Partner Center จะตีกลับตอนอัปอยู่ดี
แต่รู้ตั้งแต่ตรงนี้ถูกกว่า:

```bash
unzip -p aetox-windows-amd64.msix AppxManifest.xml | grep -o 'Version="[^"]*"'
```

สามอย่างที่ต้องมีในแพ็กเกจ ไม่งั้นไอคอนบน taskbar จะเบลอ ([DECISIONS.md](DECISIONS.md) §207):
`resources.pri` ต้องอยู่ในไฟล์ · ไฟล์โลโก้ต้องครบทุกสเกล · `Identity` ต้องเป็นของ
Partner Center ห้ามแก้เอง

**artifact มีอายุ** ถ้าปล่อยรุ่นไว้นานแล้วเพิ่งนึกได้ว่ายังไม่ได้อัป Store
ให้เช็คก่อนว่ามันหมดอายุหรือยัง — ถ้าหมด ต้องรัน workflow ใหม่:

```bash
gh api repos/Mikedev115/Aetox/actions/runs/<run-id>/artifacts --jq '.artifacts[] | {name,expired}'
```

### 8. เติม scoop hash

zip เพิ่งจะมีอยู่จริงหลังข้อ 6 ค่านี้จึงมาจากรุ่นที่ปล่อยแล้วเท่านั้น
ไม่ใช่จากไฟล์ที่บิ้วเองในเครื่อง

```bash
gh release download v<ver> -p checksums.txt -D /tmp/rel
```

เอาบรรทัดของ `aetox-windows-amd64-portable.zip` ไปใส่ช่อง `hash` ใน `scoop/aetox.json`
แล้วคอมมิตตามหนึ่งก้อน

### 9. วัดตัวเลขใหม่

ตอนนี้ zip มีอยู่จริงแล้ว ลำดับกับกติกาอยู่ที่ [PUBLISHED-NUMBERS.md](PUBLISHED-NUMBERS.md)
ข้อที่พลาดมาแล้วสองรอบคือ **ถ้าขนาดเปลี่ยน ต้องหารตัวคูณ "เล็กกว่า X กี่เท่า" ใหม่ทุกตัว**
ไม่ใช่แก้แค่ตัวตั้ง

### 10. ส่งสอง exe ให้ Defender ตรวจ — ก่อนผู้ใช้จะเป็นคนเจอ

ทำได้ทันทีที่ workflow เขียว (ไม่ต้องรอข้อ 6–9) แต่ต้องทำ **ทุกรุ่น** เพราะคำตัดสินของ
Defender ผูกกับแฮชของไฟล์ รุ่นถัดไปคือไฟล์ใหม่ที่โมเดลตัดสินใหม่ได้ โดนมาแล้วสองครั้ง:
ตัวติดตั้ง (`Program:Win32/Wacapew.C!ml`, 20 ส.ค. 2026) และ `aetox-engine.exe` ของ 1.7.1
(`Trojan:Script/Wacatac.C!ml`, 15 ก.ย. 2026 — กักกลางเซสชันของผู้ใช้ รายการ provider ว่างทั้งแอป)
ครั้งหลังไมโครซอฟท์ตอบว่าตรวจผิดภายในวันเดียว แต่ผู้ใช้เป็นคนพบก่อน

1. เปิด https://www.microsoft.com/wdsi/filesubmission ด้วยบัญชี Microsoft ของเจ้าของ เลือก
   **Software developer**
2. ส่ง `aetox-windows-amd64-portable.zip` ของรุ่นจริงจากหน้า release (มีทั้ง `aetox.exe` และ
   `aetox-engine.exe`) และ `aetox-amd64-installer.exe` — ระบุว่า Incorrect detection
   และแปะลิงก์ release กับ README หัวข้อ Windows Defender
3. จด Submission ID ไว้ใน `docs/release-notes/v<ver>.md` (บรรทัดเดียว) — เวลามีคนแจ้งว่า
   "provider ไม่แสดง / ไม่พบ aetox-engine.exe" จะได้ตอบด้วย ID แทนการเริ่มใหม่

ถ้าไมโครซอฟท์ตอบว่าถอด detection แล้วแต่ยังมีคนโดนอยู่ — เครื่องนั้นแคชคำตัดสินเดิมไว้
ให้เขารัน (cmd แบบผู้ดูแล ใน `C:\Program Files\Windows Defender`)
`MpCmdRun.exe -removedefinitions -dynamicsignatures` แล้ว `MpCmdRun.exe -SignatureUpdate`
จากนั้นกู้คืนไฟล์จาก Protection history ตามที่ README เขียนไว้

---

## เช็กลิสต์สั้น

- [ ] `verify.sh` เขียวทุกด่าน
- [ ] `go test ./internal/version/` ผ่าน (หกไฟล์ตรงกัน)
- [ ] `docs/release-notes/v<ver>.md` เขียนแล้ว
- [ ] คอมมิตอยู่บน `main` แล้ว **ก่อน** สร้าง tag
- [ ] พุช main แล้วค่อยพุช tag
- [ ] workflow `release` เขียว
- [ ] `gh release edit v<ver> --draft=false --latest`
- [ ] **อัป `.msix` ขึ้น Partner Center**
- [ ] เติม scoop `hash` จาก `checksums.txt` ของรุ่นจริง
- [ ] วัดขนาดใหม่ และหารตัวคูณใหม่ถ้าขนาดเปลี่ยน
- [ ] ส่ง zip + installer เข้า WDSI (Software developer) และจด Submission ID ลง release notes
