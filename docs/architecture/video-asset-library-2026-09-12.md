# คลังวัตถุดิบวิดีโอ — ของ 30GB ที่ยัดทีเดียวไม่ไหว จึงต้องเป็นตัวเลือก

**12 ก.ย. 2026** · สถานะ: ชั้น A ลงแล้ว (§5.1–5.4) · หน้ารอบสองอยู่ที่ [studio-shelf-page-2026-09-12.md](studio-shelf-page-2026-09-12.md) · ชั้น B ยัง Proposed

**เจ้าของ, 12 ก.ย.:** *"ผมจะพามาอัปเกรดเอเจนสร้างวิดีโอ ผมไปเจอขุมทรัพย์มา แต่ยัดทีเดียวไม่น่าไหวเพราะมันใหญ่มาก
เราจะใส่เป็นตัวเลือกให้ผู้ใช้โหลดเอาล่ะกัน"* — และให้ไปอ่าน
[ii23 Edit Kit](https://edit-kit.ii23.dev/manual?os=win) เผื่อได้ไอเดียมาพัฒนาเอเจนตัดต่อ

## 1. ของที่เจอ

### 1.1 ขุมทรัพย์: โฟลเดอร์ Drive "30GB+ Video Editing Assets"

โฟลเดอร์ "แชร์กับฉัน" บน Google Drive ของเจ้าของ (ที่มาไม่ทราบ — **ไม่ใช่** ลิงก์ในคู่มือ ii23
ซึ่งชี้ไปโฟลเดอร์ "peen's SFX" กับ "SFX Sound Effects" คนละอันกัน ตรวจ 12 ก.ย.) 17 zip / 33GB
39 โฟลเดอร์ย่อย จัดตามชนิดของวัตถุดิบ:

| กลุ่ม | โฟลเดอร์ (ตัวอย่าง) | เอเจนที่ใช้ |
|---|---|---|
| เสียง | 71 SFX PACK, MONEY SFX | ทั้งสอง |
| ซ้อนทับ (overlay, มักมี alpha) | overlays, FILM BURN, GREEN SCREEN EFFECTS, Animated Textures, Glowing Arrows | ทั้งสอง |
| ฉากหลังเคลื่อนไหว | MOTION BACKGROUNDS, GRID BACKGROUNDS, CLOUD ASSETS, FIRE CLOUD ANIMATIONS | video (สร้าง) |
| ไอคอน/ตัวอักษร/โลโก้เคลื่อนไหว | Animated Icons ×2, GLOWING ICONS, Animated letters, ANIMATED LOGOS ×2, ANIMATED EMOJIS, AI GENERATED EMOJIS | video (สร้าง) |
| เอฟเฟกต์เฉพาะ | MONEY EFFECT, Paper Animation, PLANT ANIMATIONS 3D, Hand Motion Graphics, CinePacks Title Card FX | ทั้งสอง |
| ภาพนิ่ง/มีม | 101 VIRAL MEMES, Human still images for… | editor (ตัดต่อ) |

นี่คือ **วัตถุดิบ ไม่ใช่โค้ดหรือเครื่องมือ** — คนละชนิดกับทุกอย่างที่ `internal/capability`
เคยโหลดมา (tesseract, poppler, ffmpeg, hyperframes, kinocut, whisper ล้วนเป็นโปรแกรม)

### 1.2 ii23 Edit Kit: สิ่งที่เขาทำ และสิ่งที่เราหยิบมาได้

Kit เป็นสินค้าขายขาด (฿790) — ชุด skill 7 ตัวสำหรับ Claude Code / Codex / Antigravity /
OpenCode ตัดทะลุเข้า CapCut และมี panel Premiere Pro แยก ต้องมี key ElevenLabs (ซับไทย)
กับ Gemini (ภาพ + วิเคราะห์คลิป) เขาเองก็เริ่มจาก HyperFrames ตัวเดียวกับที่เราใช้

สกิลทั้งเจ็ด กับที่ Aetox มีอยู่ตอนนี้:

| ของเขา | ทำอะไร | Aetox วันนี้ | ช่องว่าง |
|---|---|---|---|
| `ii23-analyze` | ถอดคำ หาช่วงเงียบ จุดเปลี่ยนฉาก เทคที่ดีสุด | `audio_transcribe` + `video_ocr` + kinocut | ยังไม่มี "ช่วงเงียบ" เป็นข้อมูลออกมาตรง ๆ |
| `ii23-transcribe` | ซับไทย 2 pass แบ่งคำถูก | whisper tiny q5_1 | **ช่องว่างจริง** — tiny ผิดคำไทยบ่อย และไม่มีการแบ่งคำไทย |
| `ii23-clean-cut` | ตัดเทคพูดผิด + ช่วงเงียบ ปรับเสียง | kinocut ตัด/ต่อได้ | ต้องดูว่า kinocut มี silence-detect ไหม |
| `ii23-capcut` | เขียนการตัดลง CapCut | — | นอกขอบเขต เราเลือก kinocut เป็นห้องตัดแล้ว (video-edit-tool-2026-08-25) |
| `ii23-video-handoff` | จำสไตล์/จังหวะจากคลิปที่เสร็จ | ระบบ memory ต่อโต๊ะ | หยิบได้ทันที: ให้เอเจนวิดีโอบันทึก "สไตล์ของบ้านนี้" ลง memory ของโต๊ะ |
| `ii23-cutout` | ลบพื้นหลัง | — | ทีหลัง (rembg เป็น Python + โมเดล ~170MB) |
| `ii23-find-assets` | หาเสียง/เพลง/มีม/คลิปแทรก ไม่มีก็ให้ AI สร้าง (บอกราคาก่อน) | `media_fetch` ดึงจาก URL ได้ แต่ **ไม่มีคลังให้ค้น** | **นี่คืองานที่เอกสารนี้ตอบ** |

สิ่งที่หยิบ: แนวคิด "หา asset ได้จากในห้อง ไม่ต้องสลับแท็บ" กับ "จำสไตล์ไว้ใช้ซ้ำ"
สิ่งที่ต่างโดยตั้งใจ: ของเขาค้นเว็บแล้วเสนอสร้างด้วย API เสียเงิน ของเราค้น **คลังในเครื่อง** ที่ผู้ใช้เลือกเอง
ทำงาน offline ทั้งหมด ตามหลักเดียวกับ renderer ("a render happens offline or it does not happen")

## 2. สิ่งที่มีอยู่แล้ว และรับงานนี้ได้แค่ไหน

- **`internal/capability`** — ของเสริมโหลดทีหลัง ปักหมุด URL+SHA256 คู่กัน ลงที่ `<DataRoot>/tools/…`
  แสดงขนาด/ลิขสิทธิ์/หน้าบ้านก่อนกด มี Probe/Marker ตรวจทุกครั้งที่อ่านสถานะ
  นี่คือกลไก "ตัวเลือกให้ผู้ใช้โหลด" ที่เจ้าของพูดถึง และมันดีอยู่แล้ว
- **`desktop/videotooling.go` → `VideoReadiness`** — แถวความพร้อมต่อเอเจน (`editor`, `templates`,
  `h264`, `gsap`) แสดงใน `VideoReady.svelte` ปุ่มเดียวโหลดสิ่งที่ขาดของการ์ดนั้น
- **สกิล `video-templates`** ([internal/subagent/profiles/agents/video/skills/video-templates/SKILL.md](../../internal/subagent/profiles/agents/video/skills/video-templates/SKILL.md))
  — ตาราง 75 ฉากพร้อมความยาว/กรอบ เอเจนอ่านแล้วเรียก `video new <name>`
  แบบแผน "ชั้นวางของที่มีสารบัญ" นี้คือต้นแบบให้คลังวัตถุดิบ
- **เครื่องมือ `video`** (`desktop/video_tool.go`) — `new` / `check` / `render`
  `video new` ก๊อปไฟล์เสียงมาไว้ข้างโปรเจกต์อยู่แล้ว แปลว่า "วัตถุดิบต้องอยู่ข้างฉาก" เป็นสัญญาที่ renderer มีอยู่
- **ffprobe** มากับ ffmpeg ทุกแบบที่ติดตั้ง (`videoEditorNeeds`) — พอสำหรับอ่านความยาว ขนาด และ alpha ของไฟล์

## 3. ทำไม "โฮสต์ทั้งก้อนแล้วให้กดโหลด" ทำไม่ได้ — สามข้อ

### 3.1 ขนาด: ตัวโหลดวันนี้ไม่รองรับ

`Component.download()` เขียนลง `os.CreateTemp("", …)` ไฟล์เดียว ไม่มี Range/resume
หลุดที่ 20GB คือเริ่มใหม่ และ `%TEMP%` อยู่บนไดรฟ์ C ของคนที่อาจไม่มีที่ 30GB
GitHub Release asset ยังจำกัดไฟล์ละ 2GB ซึ่งเป็นที่โฮสต์เดียวที่เราใช้อยู่ (tesseract, hyperframes, kinocut)

**ผล:** ถ้าจะโหลดผ่าน capability ต้อง (ก) แตกเป็นแพ็กตามหมวด ≤2GB ต่อไฟล์ (ข) เพิ่ม resume
(ค) เขียน `.part` ลง `<DataRoot>` ไม่ใช่ `%TEMP%` — สามข้อนี้เป็นงานของตัวโหลด ไม่ใช่ของแพ็ก

### 3.2 ที่มาและลิขสิทธิ์: นี่คือข้อที่ต้องพูดตรง ๆ

ทุก `Component` ใน manifest มี `License` กับ `Homepage` เพราะเราเอาของคนอื่นไปวางบนเครื่องผู้ใช้
และเราตอบได้ว่าของใคร ใช้ตามเงื่อนไขอะไร โฟลเดอร์ Drive นี้ตอบไม่ได้:

- เป็น "แชร์กับฉัน" จากบัญชีที่ไม่รู้ว่าเป็นใคร ไม่มี LICENSE ไม่มีที่มาต่อไฟล์
- **`CinePacks Title Card FX`** เป็นสินค้าขายของ CinePacks — ไม่ใช่ของแจก
- **`101 VIRAL MEMES`** เป็นภาพที่มีเจ้าของลิขสิทธิ์แน่นอน
- ที่เหลือ (Animated Icons, MOTION BACKGROUNDS …) ไม่รู้ว่าดึงมาจากสต็อกที่ห้าม redistribute หรือไม่
- แม้แต่คู่มือของ ii23 เองยังเขียนกำกับว่า *"เช็คสิทธิ์การใช้งานเชิงพาณิชย์เองก่อนเอาไปใช้กับงานขาย"*

ถ้า Aetox เอาไฟล์เหล่านี้ขึ้น `Mikedev115/Aetox/releases` แล้วให้แอปกดโหลด นั่นคือ **Aetox เป็นผู้เผยแพร่ซ้ำ**
ในชื่อของเรา บนโดเมนของเรา — เป็นความเสี่ยงต่อ repo ทั้งตัว (DMCA takedown บน GitHub ไม่ได้เอาแค่ asset เดียว)
ไม่ใช่การตัดสินที่ควรทำเงียบ ๆ ตามเหตุผลเดียวกับที่เราไม่ยัด GSAP ลง binary

### 3.3 เอเจนมองไม่เห็นของ 30GB

โฟลเดอร์ไฟล์ mp4/mov/wav หลายพันไฟล์ไม่มีประโยชน์กับเอเจนที่ฟังเสียงไม่ได้และดูคลิปทีละไฟล์ไม่ไหว
`video-templates` ใช้งานได้เพราะมี **ตาราง** — ชื่อ ความยาว กรอบ สิ่งที่ต้องรู้ก่อนหยิบ
คลังวัตถุดิบต้องมีสารบัญแบบเดียวกัน แต่ใหญ่กว่า 75 แถวมาก จึงยัดลงสกิลไม่ได้
(บทเรียน มุ่งเป้า: ของยาวในบริบทถูกส่งซ้ำทุกรอบ) — ต้องเป็น **เครื่องมือค้น** ไม่ใช่รายการในสกิล

## 4. ตัดสิน (เสนอ): สองชั้น

### ชั้น A — นำเข้าคลังของผู้ใช้เอง (ทำก่อน ไม่มีปัญหาลิขสิทธิ์)

ผู้ใช้โหลด Drive เอง (อย่างที่เจ้าของกำลังทำอยู่) หรือมีโฟลเดอร์ asset ของตัวเองอยู่แล้ว
แล้ว **ชี้โฟลเดอร์ให้ Aetox** Aetox สแกน ทำสารบัญ แล้วเอเจนทั้งสองค้นได้

ของอยู่ที่เดิมของผู้ใช้ Aetox ไม่ก๊อป 30GB ซ้ำ (บันทึกแค่ path + สารบัญ) ก๊อปเฉพาะไฟล์ที่ถูกหยิบเข้าโปรเจกต์
ตอนใช้จริง — ตามสัญญา "วัตถุดิบอยู่ข้างฉาก" ที่ `video new` ทำอยู่แล้ว

ความรับผิดชอบเรื่องสิทธิ์อยู่กับผู้ใช้ เหมือนไฟล์ทุกไฟล์ที่เขาแนบเข้าแชท และ UI ต้องพูดประโยคนี้ตรง ๆ หนึ่งบรรทัด

### ชั้น B — แพ็กที่กดโหลดได้ (เฉพาะที่ตอบเรื่องสิทธิ์ได้)

การ์ดใน manifest เหมือน tesseract: `.github/workflows/tools.yml` แพ็กจากต้นทางที่ระบุลิขสิทธิ์ชัด
แยกตามหมวด ไฟล์ละ ≤2GB ผู้ใช้ติ๊กเลือกหมวด ต้องมี resume ก่อน (§3.1)

ผู้สมัครที่ต้องตรวจก่อนปักหมุด (ยังไม่ตัดสินสักตัว):
- Kenney (kenney.nl) — SFX/UI sounds CC0 ✔ redistribute ได้
- freesound.org — เฉพาะที่ CC0 คัดเป็นชุดเอง
- Mixkit — licence ของตัวเอง **ห้าม redistribute เป็นชุด** ✘ ต้องให้ผู้ใช้โหลดเอง (ชั้น A)
- Pixabay/Pexels — licence ห้ามแจกซ้ำเป็นคอลเลกชัน ✘ เช่นกัน

ชั้น B จึงน่าจะได้แค่ "เสียง" กับ overlay ที่ CC0 จริง ๆ — เล็กกว่า 30GB มาก และนั่นถูกแล้ว

## 5. โครงที่เตรียมไว้ — และที่ลงจริง

**เจ้าของ, 12 ก.ย. (รอบสอง):** *"หน้าที่คุณคือคิดแค่ 2 เอเจนนี้เท่านั้น สร้างวิดีโอ และ ตัดวิดีโอ ... เราจะไม่ฝังไปในตัวติดตั้ง
แต่จะให้ผู้ใช้กดโหลดเอง เราควรมีคลังสำหรับสตูดิโอนะ ทำไว้ที่หน้าตั้งค่าดีไหม"* — ชื่อห้องจึงเป็น **คลังสตูดิโอ**
อยู่ในตั้งค่า กลุ่มเครื่องมือ ข้าง เสียง/สร้างภาพ

ที่ลงจริงต่างจากโครงด้านล่างสองจุด: (1) ไม่มีสกิล `video-assets` — คำแนะนำไปอยู่ใน `Guidance()` ของเครื่องมือ
(ส่งครั้งเดียวตอนเรียกแรก ตาม guidance.go) กับย่อหน้าเดียวใน AGENT.md ของทั้งสองเอเจน เพราะสกิลต้องผูกกับโฟลเดอร์ของเอเจนคนเดียว
ส่วน Guidance ไปถึงทั้งคู่โดยไม่ต้องมีสองสำเนา (2) "ผู้ใช้กดโหลดเอง" = ปุ่มเปิดหน้าดาวน์โหลดของแหล่งนั้นในเบราว์เซอร์
(`studioSources.ts`) ไม่ใช่ Aetox โหลด — ตาม §3.2

ไฟล์: `internal/assetlib/` (assetlib.go, probe.go, ทดสอบ) · `desktop/studio_library.go` (binding + เครื่องมือ `asset_find` + แถว readiness)
· `desktop/screen_doors.go` (`AddStudioLibrary` ไดอะล็อก) · `Settings.svelte` หมวด `studio` · `studioSources.ts` · locale สามภาษา
· `VideoReady.svelte` แถว `assets` · `category.go` (`asset_find` → deliverables) · AGENT.md ของ video กับ editor


### 5.1 แพ็กเกจใหม่ `internal/assetlib`

```go
// Asset คือวัตถุดิบหนึ่งไฟล์ในสารบัญ อ่านจาก ffprobe ครั้งเดียวตอนสแกน
type Asset struct {
    ID       string   // sha1 ของ path สัมพัทธ์ — คงที่แม้ย้ายรากคลัง
    Path     string   // สัมพัทธ์กับ Library.Root
    Kind     Kind     // sfx | music | overlay | background | icon | image | clip
    Category string   // ชื่อโฟลเดอร์ชั้นแรก ตามที่ผู้ใช้จัดมา ไม่แปล
    Duration float64  // วินาที; 0 สำหรับภาพนิ่ง
    Width, Height int
    HasAlpha bool     // mov/webm ที่ pix_fmt มี a — คือ "overlay ได้" หรือไม่
    Bytes    int64
    Tags     []string // จากชื่อไฟล์แยกคำ + ชื่อโฟลเดอร์ทุกชั้น ตัวพิมพ์เล็ก
}

// Library คือคลังหนึ่งราก ผู้ใช้มีได้หลายราก
type Library struct {
    ID, Root string
    Scanned  time.Time
    Assets   []Asset
}

func Scan(ctx, root string, probe ProbeFunc, onProgress func(done, total int)) (*Library, error)
func (l *Library) Search(q Query) []Asset   // Query{Text, Kind, MaxDuration, NeedAlpha, Limit}
func Load()/Save() // <DataRoot>/assets/libraries.json
```

- `Kind` อนุมานจาก ext + ffprobe: `.wav/.mp3` → sfx (music ถ้า >30s), `.mov/.webm` มี alpha → overlay,
  `.mp4` ไม่มี alpha → background ถ้าชื่อโฟลเดอร์มี background/texture/cloud ไม่งั้น clip, `.png/.jpg/.gif` → image/icon
  ผิดได้ — จึงเก็บ `Category` ดิบไว้ด้วยเสมอ ให้เอเจนอ่านเองได้
- **ไม่แตะ `internal/capability`** สำหรับชั้น A: มันเป็นเรื่องโปรแกรมที่ปักหมุดได้ คลังของผู้ใช้ไม่ใช่
- สแกน 30GB = ffprobe หลายพันครั้ง ต้องมี progress และยกเลิกได้ รันเป็นงานเบื้องหลังแบบเดียวกับ `runCapabilityInstall`

### 5.2 เครื่องมือ `asset_find` (builtin ให้ทั้ง `video` และ `editor`)

เครื่องมือแยก ไม่ใช่ action ใหม่ของ `video` — เพราะ `editor` ไม่มีเครื่องมือ `video` (ของเขามาจาก kinocut ทั้งหมด)

```
asset_find  query="whoosh"  kind=sfx  max_seconds=2        → ตาราง ≤20 แถว: id, ชื่อ, หมวด, ความยาว, alpha
asset_find  id=<id>  into=<project dir>                     → ก๊อปเข้าโปรเจกต์ คืน path สัมพัทธ์ให้เขียนใน <video src> / ส่ง kinocut
asset_find  summary=true                                     → นับต่อหมวด — สิ่งเดียวที่สกิลอ้างถึง
```

คลังว่าง → ตอบว่าว่างและบอกว่าเพิ่มได้ที่ งานวิดีโอ → คลังวัตถุดิบ (ประโยคเดียวกับที่ renderer ตอบเมื่อยังไม่ติดตั้ง)

### 5.3 สกิล `video-assets` (สั้น ไม่ใช่ตาราง)

อยู่ข้าง `video-templates` แต่ต่างกันโดยตั้งใจ: ไม่มีรายการไฟล์ มีแค่
- คลังคืออะไร ใครเป็นคนเลือกของเข้ามา และสิทธิ์เป็นของผู้ใช้
- ใช้ `asset_find` อย่างไร และ **หยิบตอนที่รู้แล้วว่าฉากต้องการอะไร** (ลำดับเดียวกับ AGENT.md ของ video: ตัดสินก่อน แล้วค่อยไปดูชั้นวาง)
- overlay ต้องเป็น mov/webm ที่มี alpha; ใน hyperframes วางเป็น `<video>` ที่มี `data-*` เวลา เหมือน media อื่น (ตรวจกับสกิล `media-use` ของ renderer ว่าต้องเป็นเจ้าของไฟล์อย่างไร ก่อนเขียนสกิลนี้)
- SFX ใน kinocut: ส่ง path สัมบูรณ์ตรง ๆ ได้

แถว `templates` ใน `VideoReadiness` เพิ่มเพื่อน: `assets` (state: none / ok, Where: จำนวนไฟล์ + ขนาด)

### 5.4 UI

- **VideoReady** — แถว "คลังวัตถุดิบ" พร้อมปุ่ม "เพิ่มโฟลเดอร์" → dialog เลือกโฟลเดอร์ (มีอยู่แล้วใน desktop หลัง bbbe4384)
  → progress สแกน (รูปทรงเดียวกับ `CapabilityProgress`) → แถวขึ้นจำนวน
- **ตั้งค่า → คลังวัตถุดิบ** — รายชื่อราก, สแกนใหม่, เอาออก (เอาออก = ลบสารบัญ ไม่ลบไฟล์ของผู้ใช้ — ต้องเขียนบนปุ่ม)
- คำใน locale ทั้งสาม, ไม่มี emoji (memory-split-by-desk)
- ชั้น B ค่อยเป็นการ์ดใน Onboarding/ตั้งค่าตามแบบ capability เดิม เมื่อมีแพ็กที่ผ่าน §4

### 5.5 งานตัวโหลด (ต้องมีก่อนชั้น B แม้แพ็กจะเล็ก)

- `download()` เขียน `<DataRoot>/tools/.part/<id>` แทน `%TEMP%`; ส่ง `Range: bytes=<size>-` ถ้ามี `.part` อยู่
  ตรวจ SHA256 ตอนจบเหมือนเดิม (ต้องคำนวณต่อจากไบต์เดิม — อ่านไฟล์ที่มีก่อน หรือเก็บ state ของ hasher)
- `Component` เพิ่ม `Parts []Part{URL, SHA256}` สำหรับแพ็กที่เกิน 2GB — หรือปล่อยเป็นหลาย Component ใต้ Capability เดียว
  ซึ่งกลไก "หลาย Component หนึ่ง Capability" มีอยู่แล้ว (speech = whisper + model) **เลือกอย่างหลัง** ไม่ต้องแตะ type

## 6. ลำดับงาน

1. `internal/assetlib` — Scan / Search / Load / Save + ทดสอบด้วยโฟลเดอร์จริงที่เจ้าของโหลดมา
2. `asset_find` tool + สกิล `video-assets` + แถว `assets` ใน readiness — เอเจนใช้ได้จริงจากแชท
3. UI เพิ่มโฟลเดอร์ / ตั้งค่า
4. ให้ `video` บันทึก "สไตล์บ้านนี้" ลง memory โต๊ะ (หยิบจาก `ii23-video-handoff` — งานเล็ก แยก commit)
5. ตัวโหลด resume + `.part` ใน DataRoot (เปิดทางชั้น B และแก้จุดอ่อนของ hyperframes 157MB ไปพร้อมกัน)
6. ตรวจ licence ผู้สมัคร §4 ทีละตัว แล้วค่อยเพิ่ม `tools.yml` job + การ์ด

## 7. คำถามที่ต้องให้เจ้าของตอบ

1. **ยืนยันว่าไม่โฮสต์ Drive pack เอง** (§3.2) — ถ้ายืนยัน ชั้น A คือคำตอบของ "ตัวเลือกให้โหลด" สำหรับก้อนนี้
   คือผู้ใช้ไปโหลดจากลิงก์ที่เราชี้ให้ แล้วนำเข้า
2. ชื่อไทยของห้องนี้: "คลังวัตถุดิบ" / "คลังของตกแต่ง" / อื่น
3. ช่องว่างซับไทย (§1.2) เป็นงานคนละเรื่อง แต่ใหญ่พอควรเปิดเอกสารของตัวเอง — จะเปิดไหม
