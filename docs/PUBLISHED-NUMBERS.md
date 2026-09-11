# ตัวเลขที่เผยแพร่ — อยู่ที่ไหน และวัดใหม่ยังไง

> เอกสารนี้ไม่เก็บค่าตัวเลข มันเก็บ **ที่อยู่** กับ **คำสั่งที่วัดค่านั้นใหม่**
> ถ้าเขียนค่าไว้ที่นี่ด้วย มันก็จะกลายเป็นที่ที่ต้องตามแก้อีกที่หนึ่ง
>
> กติกาว่าอะไรวัดได้/วัดไม่ได้อยู่ที่ [BENCHMARK.md](../BENCHMARK.md) · ที่นี่ตอบคำถามเดียว
> คือ *"ถ้าตัวเลขนี้เปลี่ยน ต้องไปแก้กี่ที่"*

## ระดับการบังคับ

| | แปลว่า |
|:---|:---|
| 🔒 **เทสต์บังคับ** | แก้โค้ดแล้วลืมแก้เอกสาร เทสต์แดง |
| 📦 **ตอนออกรุ่น** | เทสต์ตรวจไม่ได้ ต้องมี artifact ก่อน — เช็กตอนปล่อยรุ่น |
| ✋ **มือล้วน** | ไม่มีอะไรจับได้เลย ต้องจำเอง |

---

## ตาราง

> `aetox-landing` = รีโป [Mikedev115/aetox-landing](https://github.com/Mikedev115/aetox-landing) ปล่อยที่
> <https://mikedev115.github.io/aetox-landing/> (ย้ายออกจาก `docs/index.html` เมื่อ 11 ก.ย. 2026 —
> ที่นี่เหลือหน้า redirect กับ `privacy.html` ซึ่ง URL จดไว้กับ Microsoft Store จึงย้ายไม่ได้)
>
> ตัวเลขในรีโปนั้นอยู่ 3 ที่: `lib/i18n/en.ts` + `th.ts` (ข้อความ) · `components/Weight.tsx`
> (`KPI_VALUES` กับตารางขนาดคู่แข่ง) · `components/UseCases.tsx` (ตัวเลขของเคส ไม่ใช่ของแอป)
> ส่วน `lib/version.ts` **ไม่มีสำเนาเลขรุ่นเลย** และ `lib/site.ts` เก็บแค่ที่อยู่ลิงก์ขาออก ไม่มีตัวเลข

| ตัวเลข | อยู่ที่ | วัดใหม่ด้วย | บังคับ |
|:---|:---|:---|:---|
| **เวอร์ชัน** | `internal/version` · `desktop/wails.json` · `scoop/aetox.json` · README ทั้งสอง (หัวข้อสถานะ) | `go test ./internal/version` · แลนดิ้งเพจไม่มีสำเนาให้ตรวจ (`lib/version.ts` อ่าน GitHub Releases ตอน build) | 🔒 |
| **จำนวนเครื่องมือ + ตารางเครื่องมือ** | README ทั้งสอง (§มันทำอะไรได้บ้างทั้งหมด) · `aetox-landing` (`components/Weight.tsx` — `KPI_VALUES` · การ์ดใน `lib/i18n`) · `ARCHITECTURE.md` (ไดอะแกรม) | `go test ./desktop -run TestPrintReadmeToolTable -v` — พิมพ์ตารางออกมาให้ก๊อป | ✋ |
| **โทเคนของบล็อกเครื่องมือ + เพดาน** | README ทั้งสอง · `aetox-landing` (`lib/i18n/en.ts` + `th.ts`) | `go test ./desktop -run TestTheToolBlockStaysWithinItsBudget -v` | ✋ |
| **จำนวนเทสต์** | README ทั้งสอง (badge + ตาราง "วัดมา ไม่ใช่อ้าง") · `aetox-landing` (การ์ดสถิติ) | `go test ./... -count=1` · `cd desktop/frontend && npx vitest run` | ✋ |
| **จำนวนผู้ให้บริการ + รายชื่อ** | README ทั้งสอง (จุดเด่น + §ผู้ให้บริการ) · `aetox-landing` (`lib/i18n` — ข้อความ "19 providers") | `internal/provider/catalog.go` — `canonicalOrder` | ✋ |
| **จำนวนเอเจน + ซับเอเจน** | README ทั้งสอง (§ทีมงาน) — **รีโป landing ไม่มีตัวเลขนี้** | นับ `internal/subagent/profiles/agents/*/` และ `internal/subagent/profiles/subagents/*.md` · `go test ./internal/subagent` ผูกจำนวนไว้ที่ตัวโปรไฟล์ — **แต่ยังไม่ตรวจ README** ต่างจาก `go test ./internal/version` ที่อ่าน README จริง | ✋ |
| **ขนาด `aetox.exe`** | README ทั้งสอง (4 จุด: ย่อหน้าเปิด · ตารางวัด · ตารางเทียบ Zed · ย่อหน้าวิธีวัด) · `BENCHMARK.md` §4 · `PLATFORM-SUPPORT.md` · `ROADMAP.md` · `aetox-landing` **3 ที่** ในรีโปนั้น (`components/Weight.tsx` ตารางบวก `KPI_VALUES` · `lib/i18n` การ์ดสถิติและ FAQ) | โหลด portable zip ของรุ่นนั้น แตกออก อ่านขนาดไฟล์เดียวข้างใน | 📦 |
| **ขนาดตัวติดตั้ง** | README ทั้งสอง (§ติดตั้ง + ตารางวัด) · `BENCHMARK.md` §4 · `aetox-landing` (การ์ดสถิติ · ขั้นตอนติดตั้ง) | `gh release view --json assets` | 📦 |
| **ตัวคูณ "เล็กกว่า X กี่เท่า"** | `BENCHMARK.md` §4 · `PLATFORM-SUPPORT.md` · `aetox-landing` (`lib/i18n/en.ts` + `th.ts` — ข้อความท้ายกราฟ) | หารใหม่ทุกครั้งที่ขนาดเปลี่ยน — **นี่คือช่องที่พลาดมาแล้วสองรอบ** | ✋ |
| **ขนาดคู่แข่ง** | `BENCHMARK.md` §4 · `aetox-landing` (`components/Weight.tsx` — ค่า `mb` กับความกว้าง `w`) · README ทั้งสอง (ตาราง Zed) | ลงโปรแกรมจริงแล้ววัดจากโฟลเดอร์ติดตั้ง ([BENCHMARK.md](../BENCHMARK.md) ข้อ 5 บอกว่าห้ามนับอะไร) | ✋ |
| **RAM · เวลาเปิด · จำนวน process** | README ทั้งสอง (แถว ⁽ᵈ⁾) · `BENCHMARK.md` · `aetox-landing` | รีบูตก่อน แล้ว `.\bench.ps1 -Start` | ✋ |
| **ประกอบหนึ่งเทิร์น** | README ทั้งสอง (ตารางวัด) | `.\bench.ps1 -Engine` | ✋ |

---

## ลำดับตอนออกรุ่นใหม่

> ครึ่งที่เป็นตัวเลขอยู่ข้างล่างนี้ · ครึ่งที่เป็นขั้นตอนการปล่อยรุ่น (tag, เอาออกจาก draft,
> **อัป .msix ขึ้น Partner Center**, เติม scoop hash) อยู่ที่ [RELEASING.md](RELEASING.md)

1. `go test ./...` — เวอร์ชันในหกไฟล์ตรงกันหรือยัง
2. รอ CI ปล่อย artifact แล้ว `gh release view --json assets` → ขนาดตัวติดตั้ง
3. โหลด zip แตกออก อ่านขนาด `aetox.exe`
4. **ถ้าขนาดเปลี่ยน หารตัวคูณใหม่ทุกตัว** ไม่ใช่แค่แก้ตัวตั้ง
5. ลงผลใน [BENCHMARK.md](../BENCHMARK.md) ข้อ 13 ก่อน แล้วค่อยยกขึ้น README กับเว็บ
6. **เว็บไม่อัปเดตเอง** — `aetox-landing/.github/workflows/deploy.yml` ฟัง `repository_dispatch`
   ชนิด `aetox-release` อยู่จริง แต่ **รีโปนี้ไม่เคยยิงมัน** (`grep repository_dispatch` ใน
   `.github/workflows/` ของที่นี่ = 0) ป้ายเวอร์ชันบนเว็บจึงรอ push ครั้งถัดไปของรีโปนั้น
   หรือสั่ง deploy มือ · เลขตัวอื่นบนเว็บไม่ขยับตามอยู่แล้วเพราะอยู่ในโค้ด

ข้อ 4 คือข้อที่พลาดมาแล้วสองรอบ ทั้งสองรอบตัวตั้งถูกแก้ ตัวคูณไม่ถูกแก้
และทั้งสองรอบตัวคูณที่ค้างอยู่เข้าข้างเราทุกตัว

ข้อ 5 คือลำดับที่ [BENCHMARK.md](../BENCHMARK.md) ข้อ 12 เขียนไว้อยู่แล้ว รอบ 22 ส.ค. เดินย้อนทาง
ตัวเลขขึ้นเว็บก่อนโดยไม่ผ่าน BENCHMARK ผลคือเว็บกับ README ตอบคนละค่าอยู่สามวัน

---

## ที่ยังไม่มีใครบังคับ และควรบังคับ

จำนวนเครื่องมือ ตารางเครื่องมือ และโทเคนของบล็อก — ทั้งสามอย่างนี้โปรแกรมตอบเองได้ทั้งหมด
แต่ยังต้องก๊อปด้วยมือ เทสต์ที่อ่าน README แล้วเทียบกับทะเบียนจริงจะปิดช่องนี้ได้ทั้งช่อง
และเป็นเทสต์ที่เขียนได้จริง ต่างจากขนาดไฟล์ที่ต้องรอ artifact

**และตั้งแต่ 11 ก.ย. 2026 สำเนามือมีสามที่ ไม่ใช่สอง** — ที่ที่สามอยู่ในอีกรีโปหนึ่ง (`aetox-landing`)
เทสต์ในรีโปนี้จึงอ่านไปไม่ถึง และปิดได้แค่ README กับ `ARCHITECTURE.md` · ส่วนเว็บต้องมีเทสต์
ของตัวเองในรีโปนั้นหรือไม่มีเลย สองทางนั้นต่างกันที่ตัวเลขจะค้างโดยไม่มีใครรู้ หรือค้างแล้วมีคนรู้

**ที่ค้างอยู่จริงตอนนี้ — ตรวจ 11 ก.ย. 2026** — `TestPrintReadmeToolTable` กับ
`TestTheToolBlockStaysWithinItsBudget` ตอบ **34 tools / ~9,882 tokens** ขณะที่ README เขียน
28 / ~7,527 และเว็บเขียน 31 / 8,477 · สามที่สามค่า และไม่มีอะไรเทียบให้ตรงกันได้เลย
