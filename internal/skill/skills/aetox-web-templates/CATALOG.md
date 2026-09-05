# แคตตาล็อกอ้างอิง — ของฟรีทั้งหมดในตลาด เทียบกับคลังของเรา

สำรวจ 4 ก.ย. 2026 · 20 เว็บ · **นับจากทะเบียนจริงของแต่ละเว็บ** (`registry.json`,
`sitemap.xml`, GitHub tree) ไม่ใช่นับจากหน้าจอ · **ตัดของที่เสียเงินออกหมดแล้ว**
(รายการที่ข้ามอยู่ท้ายไฟล์)

เอกสารนี้ไม่ใช่รายการสิ่งที่ต้องทำทั้งหมด มันคือแผนที่ว่าโลกข้างนอกมีอะไร จัดหมวดยังไง
และคลังเราขาดตรงไหน ทุกชิ้นที่จะรับเข้าคลังยังต้องผ่าน `STANDARD.md` ข้อ 0 ก่อนเสมอ —
**ถ้าโมเดลเจนเองได้ผลไม่ต่าง ไฟล์นั้นไม่ควรมีอยู่** รายชื่อยาวข้างล่างนี้จึงเป็นเมนู
ไม่ใช่ใบสั่ง

> `STANDARD.md` ข้อ 8 ยังบังคับอยู่: **ห้ามคัดลอกโค้ดหรือไฟล์ของเว็บพวกนี้มาทั้งชิ้น**
> ศึกษากลไกแล้วเขียนใหม่เท่านั้น และห้ามลิงก์ asset จากเซิร์ฟเวอร์ของเขา
> เอกสารนี้เก็บแต่ *ชื่อ หมวด และจำนวน* ซึ่งเป็นข้อเท็จจริง ไม่ใช่งานสร้างสรรค์ของใคร

---

## สรุปตัวเลข

| กลุ่ม | เว็บ | ของฟรีที่นับได้ | สัญญาอนุญาต |
|---|---|---:|---|
| 1 · เอฟเฟกต์/การเคลื่อนไหว | Aceternity UI | 108 | ฟรี (Pro แยก) |
| | Cult UI | 78 | MIT |
| | Magic UI | 77 | MIT |
| | KokonutUI | 46 | MIT |
| | Motion Primitives | 33 | MIT |
| 2 · ตัวควบคุมพื้นฐาน | coss ui (เดิม Origin UI) | 55 ฐาน + 507 แบบย่อย | โอเพนซอร์ส (ไม่ประกาศชนิดบนหน้าเว็บ) |
| | Preline UI | 84 + 27 ปลั๊กอิน JS | ฟรี (Pro แยก) |
| | HyperUI | 73 หมวด / 318 ตัวอย่าง | MIT |
| | daisyUI | 68 | MIT |
| | Flowbite | 68 | MIT |
| | shadcn/ui | 63 | MIT |
| 3 · หน้าเต็ม | HTML5 UP | 43 | CC BY 3.0 |
| | Start Bootstrap | 16 ธีมฟรี | MIT |
| | Cruip | 5 | ฟรี (พรีเมียมแยก) |
| | Vercel Templates | หลายร้อย starter | ตามแต่ละ repo |
| 4 · Svelte | shadcn-svelte | 57 | MIT · **ต้องมี Tailwind** |
| | Skeleton v5 | 11 + 32 | MIT · **ต้องมี Tailwind** |
| | Flowbite Svelte | ~60 | MIT · **ต้องมี Tailwind** |
| | Melt UI (next-gen) | 18 builder | MIT · headless ไม่ต้องมี Tailwind |

**คลังเราตอนนี้: 104 ส่วน + 6 หน้าสำเร็จรูป** ทั้งหมดเป็นชั้น A

อัปเดต 4 ก.ย. 2569 หลังรื้อรอบที่สอง — เพิ่มทิศทาง `gl` (กระจก) ครบชุดหน้าขาย
SaaS 16 ส่วน · ทิศทาง `wm` (อบอุ่น) สำหรับงานที่ความงามอยู่ที่ภาพถ่าย 8 ส่วน
ทิศทาง `st` (สตูดิโอ) ขาว–ดำล้วนสำหรับเอเจนซีและพอร์ตโฟลิโอ 6 ส่วน
ชุดร้านอาหารอีก 4 ส่วนที่ใช้ทิศทาง `wm` ร่วมกับชุดที่พัก
และทิศทาง `pa` (กระดาษ) สำหรับงานที่ยืมภาษาสิ่งพิมพ์ 6 ส่วน
รวมสี่เชลล์ ห้าทิศทางศิลป์
พร้อมเชลล์อีกสองใบ (`page-shell-warm.html`, `page-shell-studio.html`)
และโฟลเดอร์ `pages/` ซึ่งเป็น
**หน้าที่เสร็จทั้งหน้า** ให้เอเจนต์หยิบไปแก้คำแก้ภาพ ไม่ต้องประกอบเองทุกครั้ง

---

## หมวด A · ตัวควบคุมพื้นฐาน

ใช้หมวดของ **daisyUI** เป็นโครง เพราะเป็นตัวที่แบ่งตามหน้าที่ได้สะอาดที่สุดในเจ็ดเจ้า
(daisyUI/shadcn/Preline/Flowbite/HyperUI/coss/Skeleton แทบทับกันหมด ต่างกันแค่ชื่อหมวด)
คอลัมน์ขวาคือสถานะของคลังเรา

### A1 · การกระทำ (Actions)

| ชิ้น | มีที่ | เรา |
|---|---|---|
| Button | ทุกเจ้า | ✓ `page-shell` (`.w-btn`, `.w-btn--ghost`) |
| Button group | shadcn, HyperUI(5), coss, Flowbite | ✗ |
| Split button | HyperUI, coss | ✓ `split-buttons` |
| Dropdown / Menu | ทุกเจ้า | ✗ |
| Context menu | shadcn, coss, Preline | ✗ |
| Modal / Dialog | ทุกเจ้า | ✓ `dialog` |
| Drawer / Sheet / Offcanvas | shadcn, coss, daisy, Preline, Flowbite | ✗ |
| Swap / Theme controller | daisyUI, Preline | ✓ `theme-toggle` |
| FAB / Speed dial | daisyUI, Flowbite | ✗ |
| Toolbar | coss, Motion Primitives, Cult UI | ✗ |
| Command palette | shadcn, coss, KokonutUI (`action-search-bar`) | ✗ |

### A2 · แสดงข้อมูล (Data display)

| ชิ้น | มีที่ | เรา |
|---|---|---|
| Accordion / Collapse / Disclosure | ทุกเจ้า | ✓ `faq` |
| Avatar + Avatar group | ทุกเจ้า | ✗ |
| Badge / Chip | ทุกเจ้า | ✗ |
| Card | ทุกเจ้า | ✓ กระจายอยู่ในหลายส่วน |
| Card stack / fan | Aceternity, KokonutUI, Cult UI | ✓ `card-stack` |
| Carousel | ทุกเจ้า | ✗ |
| Chat bubble | daisy, Preline, Flowbite, Aceternity | ✗ |
| Countdown / Timer | daisyUI, Cult UI | ✗ |
| Diff / Compare slider | daisyUI, Aceternity, Motion Primitives | ✗ |
| Kbd | shadcn, coss, daisy, Preline, Flowbite | ✗ |
| List group | daisy, Preline, Flowbite | ✗ |
| Stat | daisy, HyperUI(6) | ✓ `stats`, `stat-cards`, `stat-sparkline-row` |
| Status dot | daisyUI | ✗ (ทำได้ใน `alert`) |
| Table / Data table | ทุกเจ้า | ✓ `data-table` |
| Timeline | daisy, Preline, Flowbite, Aceternity, HyperUI(3) | ✗ |
| Code block | Preline, Aceternity, Cult UI | ✗ |
| Tree view | Preline, Melt UI, Magic UI (`file-tree`) | ✗ |
| QR code | Flowbite, Skeleton | ✗ |
| Rating | daisy, Preline, Flowbite, Skeleton | ✗ |
| Tweet / social card | Magic UI, Cult UI, KokonutUI | ✗ |

### A3 · การนำทาง (Navigation)

| ชิ้น | มีที่ | เรา |
|---|---|---|
| Navbar / Header | ทุกเจ้า | ✓ `nav` |
| Navbar หลายทรง (pill, underline, mega, floating, resizable) | Aceternity(5), Preline, Flowbite | ✗ — เรามีทรงเดียว |
| Sidebar / Vertical menu | shadcn, coss, Preline, HyperUI(8), Aceternity | ✓ `dashboard-shell`, `docs-page` |
| Dock | daisy, Magic UI, Motion Primitives, Cult UI, Aceternity | ✗ |
| Breadcrumb | ทุกเจ้า | ✓ `breadcrumb` + `breadcrumbs` (แบบยุบกลาง) |
| Pagination | ทุกเจ้า | ✗ |
| Steps / Stepper | daisy, Preline, Flowbite, HyperUI(5), Skeleton | ✗ |
| Tabs | ทุกเจ้า | ✗ |
| Scrollspy / on-this-page | Preline | ✓ `docs-page` |
| Sticky banner / Announcement | Aceternity, HyperUI(6), Flowbite | ✗ |

### A4 · ผลตอบกลับ (Feedback)

| ชิ้น | มีที่ | เรา |
|---|---|---|
| Alert | ทุกเจ้า | ✓ `alert` |
| Toast | ทุกเจ้า | ✓ `toast` |
| Tooltip | ทุกเจ้า | ✓ `tooltips` |
| Popover / Hover card / Preview card | shadcn, coss, Preline, Motion Primitives | ✗ |
| Progress (แถบ + วง) | ทุกเจ้า | ✓ `progress` |
| Skeleton loading | shadcn, coss, Preline, Flowbite | ✗ |
| Spinner / Loader | ทุกเจ้า + HyperUI(7) | ✗ |
| Multi-step loader | Aceternity | ✗ |
| Empty state | shadcn, coss, HyperUI(5+5) | ✓ `empty-state` |
| Meter | coss | ✗ |

### A5 · รับข้อมูล (Data input)

| ชิ้น | มีที่ | เรา |
|---|---|---|
| Text / Textarea / Select / Checkbox / Radio / Range / File / Date | ทุกเจ้า | ✓ `form-controls` |
| Switch (ลากได้ มีสปริง) | ทุกเจ้า | ✓ `switch` |
| Input group / addon | shadcn, coss(29), Preline | ✗ |
| Number input / Quantity | coss, Preline, HyperUI(4) | ✗ |
| OTP / PIN input | shadcn, coss(9), Preline, Melt UI | ✗ |
| Combobox / Autocomplete | shadcn, coss(20+16), Preline, Melt, Skeleton | ✗ |
| Search box | Preline, KokonutUI | ✓ `search-box` |
| Filter | daisy, HyperUI(2) | ✗ |
| File upload (ลากวาง) | shadcn, Preline, Aceternity, Melt, KokonutUI | ✗ |
| Color picker | Preline, Cult UI | ✗ |
| Tags input | Skeleton | ✗ |
| Toggle password / Strong password | Preline | ✗ |
| Rich text editor | Preline, Flowbite | ✗ (นอกขอบเขต) |
| Calendar / Date picker | shadcn, coss(25+9), Preline, Skeleton | ✗ |
| Slider (หลายหัว, มีป้าย) | coss(23), Preline, Melt | ✓ บางส่วนใน `form-controls` |
| Form card / Auth | HyperUI, Aceternity | ✓ `form-card` |

### A6 · โครงหน้า (Layout)

| ชิ้น | มีที่ | เรา |
|---|---|---|
| Container / Grid / Columns | Preline, HyperUI(10) | ✓ `page-shell` (`.w-wrap`) |
| Divider | daisy, Preline, HyperUI(6) | ✗ |
| Hero | daisyUI, ทุกเจ้าในกลุ่ม 3 | ✓ `hero` |
| Footer | daisy, HyperUI(12) | ✓ `footer` |
| Indicator / Legend | daisy, Preline | ✗ |
| Join (ต่อขอบกัน) | daisyUI | ✗ |
| Mask / Corner shapes | daisyUI, Skeleton | ✗ |
| Stack | daisyUI | ✗ |
| **Bento grid** | Aceternity, Magic UI, KokonutUI | ✗ **ทุกเจ้ามี เราไม่มี** |
| Layout splitter / Resizable | shadcn, coss, Preline | ✗ |
| Scroll area | shadcn, coss(5) | ✗ |
| Custom scrollbar | Preline | ✗ |

### A7 · กรอบจำลอง (Mockup)

| ชิ้น | มีที่ | เรา |
|---|---|---|
| Browser window | daisyUI, Cult UI | ✗ |
| Code window | daisyUI, Aceternity, Preline | ✗ |
| Terminal | daisyUI, Magic UI, Aceternity, Cult UI | ✗ |
| Phone / Tablet / Laptop | daisy, Magic UI (`iphone`,`safari`,`android`), Preline | ✗ |
| Keyboard | Aceternity | ✗ |

---

## หมวด B · บล็อกหน้าเว็บ

ใช้หมวดของ **HyperUI** เป็นโครง (Application / Marketing) เพราะเป็นเจ้าเดียวที่แยก
"หน้าจอในแอป" ออกจาก "หน้าขาย" อย่างชัดเจน และนับจำนวนแบบต่อหมวดไว้ให้ด้วย
ตัวเลขในวงเล็บคือจำนวนแบบที่ HyperUI มีจริง

### B1 · Marketing (HyperUI 23 หมวด / 106 แบบ)

| หมวด | HyperUI | เรา |
|---|---:|---|
| Headers | 4 | ✓ `nav` |
| Announcements | 6 | ✗ |
| Banners | 3 | ✗ |
| Sections (hero) | 4 | ✓ `hero` — **1 ทรง** |
| Feature grids | 4 | ✓ `features` — **1 ทรง** |
| CTAs | 4 | ✓ `cta-band` |
| Pricing | 2 | ✓ `pricing` — **1 ทรง** |
| FAQs | 3 | ✓ `faq` |
| Testimonials | 3 | ✓ `testimonial` |
| Team sections | 3 | ✓ `team`, `profile-spotlight` |
| Stats | 3 | ✓ `stats` |
| Logo clouds | 4 | ✓ `logo-bar` |
| Footers | 12 | ✓ `footer` — **1 ทรง** |
| Newsletter signup | 2 | ✓ `newsletter` |
| Contact forms | 5 | ✗ (มี `newsletter` แบบสั้น) |
| Blog cards | 7 | ✓ `article-index` |
| Cards | 9 | ✓ กระจาย |
| Buttons | 5 | ✓ `page-shell` |
| Empty content | 5 | ✓ `empty-state` |
| Polls | 3 | ✗ (Cult UI มี 5 ชิ้น) |
| Product cards / collections / carts | 8+4+3 | ✗ (อีคอมเมิร์ซ — นอกขอบเขต) |

### B2 · Application (HyperUI 34 หมวด / 163 แบบ)

ส่วนใหญ่ทับกับหมวด A ข้างบน ที่เพิ่มมาเป็นระดับ "หน้าจอ" ไม่ใช่ "ตัวควบคุม":

| หมวด | HyperUI | เรา |
|---|---:|---|
| Charts | 11 | ✓ 6 ชนิด (`chart-*`) |
| Grids | 10 | ✓ `media-grid` |
| Media | 8 | ✓ `media-grid` |
| Vertical menu | 8 | ✓ `dashboard-shell` |
| Details list | 4 | ✗ |
| Side menu | 2 | ✓ `dashboard-shell` |
| Skip links | 3 | ✓ `page-shell` |

### B3 · บล็อกที่มีเฉพาะเจ้าเอฟเฟกต์ (ฟรี)

| ชิ้น | มีที่ |
|---|---|
| Onboarding / intro disclosure | Cult UI |
| Chat conversation | Aceternity (ตัวฟรี) |
| Feature carousel / voting / poll widget | Cult UI |
| Prompt library / AI instructions | Cult UI |
| AI prompt input, AI voice, AI loading | KokonutUI |
| Apple activity ring card | KokonutUI |
| Currency transfer / team selector | KokonutUI |
| World map / dotted map | Aceternity, Magic UI |
| Globe (3D) | Aceternity, Magic UI |
| Hero video dialog | Magic UI |
| Code comparison | Magic UI |

---

## หมวด C · เอฟเฟกต์และการเคลื่อนไหว

ห้าเจ้าในกลุ่ม 1 ไม่มีใครจัดหมวดไว้เลย (Aceternity เรียงเป็นแถวเดียว 111 ชิ้น)
หมวดข้างล่างนี้จึงจัดเอง โดยเรียงตาม **ลำดับที่ Aceternity วางไว้ในหน้า** ซึ่งเรียงตาม
ตระกูลเอฟเฟกต์อยู่แล้ว แล้วยุบของ Magic UI / Cult UI / KokonutUI / Motion Primitives
เข้าไปในตระกูลเดียวกัน

คอลัมน์ **ชั้น** คือประเมินว่าย้ายเข้าคลังเราได้ที่ชั้นไหน ตาม `STANDARD.md` ข้อ 1
— `A` = CSS/SVG ล้วน ไม่มีสคริปต์ · `A+js` = ต้องมีสคริปต์สั้น ๆ · `A+canvas` =
ต้องมี canvas/WebGL (ยังเป็นไฟล์เดียวได้) · `B` = ต้องมีภาพถ่ายหรือฟอนต์จริง

### C1 · พื้นหลัง

| เอฟเฟกต์ | มีที่ | ชั้น | เรา |
|---|---|---|---|
| จุด / ตาราง / เส้นทแยง / ลายเส้น | Magic UI(6), Aceternity, HyperUI | A | ✓ `backgrounds` (8 ชั้น) |
| หกเหลี่ยม / ตารางกะพริบ / ตารางโต้ตอบ | Magic UI | A | ✗ |
| Aurora / หมอกสีไหล | Aceternity, Magic UI, Cult UI | A | ✓ `backgrounds` |
| Retro grid / perspective grid | Magic UI, Aceternity | A | ✗ |
| Meteors / ดาวตก / ดาวพื้นหลัง | Magic UI, Aceternity | A | ✗ |
| Ripple / วงกระเพื่อม | Magic UI, Aceternity | A | ✗ |
| Light rays / lamp / spotlight | Magic UI, Aceternity, Motion Primitives | A | ✗ |
| Warp / vortex / wavy | Magic UI, Aceternity | A+canvas | ✗ |
| Noise texture | Magic UI, Aceternity, Cult UI | A (feTurbulence) | ✓ `backgrounds` |
| Particles / beams / background boxes | Magic UI, Aceternity | A+canvas | ✗ |
| Fractal dot grid / lightboard / heatmap | Cult UI | A+canvas | ✗ |
| Dither / cloud / liquid metal shader | Aceternity, Cult UI | A+canvas | ✗ |
| Stripe-style guide lines | Cult UI | A | ✗ |
| Flow field | KokonutUI | A+canvas | ✗ |

### C2 · ตัวอักษร

| เอฟเฟกต์ | มีที่ | ชั้น | เรา |
|---|---|---|---|
| Typewriter / typing | Magic UI, Aceternity, Cult UI, KokonutUI | A | ✗ |
| Word rotate / flip words / text loop | Magic UI, Aceternity, Motion Primitives | A | ✗ |
| Text reveal ตอนสกอลล์ | Magic UI, Aceternity | A (`animation-timeline: view()`) | ✗ |
| Blur fade / text animate | Magic UI, Motion Primitives | A | ✗ |
| Shimmer / shiny / gradient text | Magic UI, Cult UI, Motion Primitives | A | ✗ |
| Aurora text / line shadow / comic / 3D flip | Magic UI | A | ✗ |
| Morphing / scramble / encrypted / glitch / matrix | Magic UI, Motion Primitives, Aceternity, KokonutUI | A+js | ✗ |
| Sparkles text / highlighter / pointer highlight | Magic UI, Aceternity | A | ✗ |
| Spinning text (วงกลม) | Magic UI, Motion Primitives, Cult UI | A (SVG textPath) | ✗ |
| Scroll-based velocity | Magic UI | A | ✗ |
| Pixel heading / word / paragraph | Cult UI | A | ✗ |
| Sliced / swoosh / squiggly / colourful | KokonutUI, Aceternity | A | ✗ |
| Text flipping board (แบบป้ายสนามบิน) | Aceternity | A+js | ✗ |
| Gradient heading | Cult UI | A | ✗ |

### C3 · การ์ดและพื้นผิว

| เอฟเฟกต์ | มีที่ | ชั้น | เรา |
|---|---|---|---|
| Spotlight ตามเมาส์ | Aceternity(2), Magic UI, KokonutUI | A+js (`@property`) | ✗ |
| Glare / shine / glow border | Magic UI(3), Aceternity, Cult UI | A | ✗ |
| Border beam / moving border | Magic UI, Aceternity, Cult UI | A (conic + mask) | ✗ |
| 3D tilt / wobble / comet | Aceternity(3), Motion Primitives, Cult UI | A+js | ✗ |
| Card flip / shift / cutout / minimal | KokonutUI, Cult UI | A | ✗ |
| Liquid glass / distorted glass / frosted | KokonutUI, Cult UI | A | บางส่วนใน `analytics-overview` |
| Neon / metal / neumorph | Magic UI, Cult UI | A | ✗ |
| Texture card / overlay | Cult UI | A | ✗ |
| Direction-aware hover | Aceternity, Cult UI | A+js | ✗ |
| Evervault / canvas reveal | Aceternity | A+canvas | ✗ |
| Progressive blur / edge blur | Magic UI, Motion Primitives, Cult UI | A (mask + backdrop) | ✗ |
| Lens (แว่นขยาย) | Magic UI, Aceternity | A+js | ✗ |
| Chromatic / pixelated / dither image | Aceternity, Cult UI | A+canvas | ✗ |

### C4 · การเคลื่อนไหวตามสกอลล์

| เอฟเฟกต์ | มีที่ | ชั้น | เรา |
|---|---|---|---|
| Scroll progress bar | Magic UI, Motion Primitives | A (`animation-timeline: scroll()`) | ✗ |
| Sticky scroll reveal | Aceternity | A | ✗ |
| Parallax scroll / hero parallax | Aceternity(3) | A+js | ✗ |
| Container scroll / macbook scroll | Aceternity(2) | A+js | ✗ |
| Tracing beam | Aceternity | A | ✗ |
| In-view / animated group | Motion Primitives | A (`view()`) | ✗ |
| Google Gemini effect | Aceternity | A (SVG `draw`) | ✗ |

### C5 · ปุ่มและตัวชี้

| เอฟเฟกต์ | มีที่ | ชั้น | เรา |
|---|---|---|---|
| Shimmer / rainbow / pulsating / ripple button | Magic UI(4) | A | ✗ |
| Particle / attract / hold / magnetize button | KokonutUI(4) | A+js | ✗ |
| Magnetic | Aceternity, Motion Primitives | A+js | ✗ |
| Hover border gradient / interactive hover | Magic UI, Aceternity | A | ✗ |
| Stateful button (loading → done) | Aceternity | A+js | ✗ |
| Cosmic / texture / metal / bg-animate button | Cult UI(4) | A | ✗ |
| Smooth cursor / following pointer / custom cursor | Magic UI, Aceternity, Motion Primitives | A+js | ✗ |
| Confetti / cool mode | Magic UI, Preline | A+canvas | ✗ |

### C6 · ตัวเลขและวงกลม

| เอฟเฟกต์ | มีที่ | ชั้น | เรา |
|---|---|---|---|
| Number ticker / sliding number | Magic UI, Motion Primitives, Cult UI | **A ล้วน** (`@property` + `counter`) | ✗ |
| Animated circular progress | Magic UI | A (`@property` + dasharray) | ✓ บางส่วนใน `progress` |
| Orbiting circles | Magic UI | A (`offset-path`) | ✗ |
| Icon cloud / avatar circles | Magic UI | A+js | ✗ |
| Apple activity rings | KokonutUI | A | ✗ |

### C7 · แถบวิ่งและรายการ

| เอฟเฟกต์ | มีที่ | ชั้น | เรา |
|---|---|---|---|
| Marquee (แนวนอน/ตั้ง/3D) | Magic UI, Preline, Skeleton, Aceternity | A | ✗ — `logo-bar` ยังนิ่ง |
| Infinite slider / moving cards | Aceternity, Motion Primitives, Cult UI | A | ✗ |
| Animated list (เข้าทีละอัน) | Magic UI | A | ✗ |
| Logo carousel (สลับโลโก้) | Cult UI | A | ✗ |
| Animated beam (เส้นเชื่อมกล่อง) | Magic UI | A (SVG `draw`) | ✗ |
| Sortable list / draggable card | Cult UI, Aceternity | A+js | ✗ |
| 3D carousel / apple cards carousel | Cult UI, Aceternity | A+js | ✗ |

### C8 · แผงและ overlay ที่ขยับทั้งรูปทรง

| เอฟเฟกต์ | มีที่ | ชั้น | เรา |
|---|---|---|---|
| Morphing dialog / popover | Motion Primitives(2) | A+js (`view-transition`) | ✗ |
| Dynamic island / family drawer / expandable | Cult UI(4), Aceternity | A+js | ✗ |
| Floating panel / side panel | Cult UI(2), Skeleton | A+js | ✗ |
| Transition panel / direction-aware tabs | Motion Primitives, Cult UI | A+js | ✗ |
| Smooth drawer | KokonutUI | A+js | ✗ |
| Animated theme toggler (7 ทรง) | Magic UI | A+js (`view-transition`) | ✓ `theme-toggle` แบบธรรมดา |

---

## หมวด D · หน้าเต็ม

| เว็บ | ของฟรี | สัญญาอนุญาต | เอาไปใช้ยังไง |
|---|---|---|---|
| **HTML5 UP** | 43 เทมเพลต — Paradigm Shift, Massively, Ethereal, Story, Dimension, Editorial, Forty, Stellar, Multiverse, Phantom, Hyperspace, Future Imperfect, Solid State, Lens, Fractal, Eventually, Spectral, Photon, Highlights, Landed, Strata, Read Only, Alpha, Directive, Aerial, Twenty, Big Picture, Tessellate, Prologue, Helios, Telephasic, Strongly Typed, Parallelism, Escape Velocity, Astral, Striped, Dopetrope, Miniport, TXT, Verti, Zerofour, Arcana, Halcyonic, Minimaxing | CC BY 3.0 (ต้องให้เครดิต) | ดูองค์ประกอบทั้งหน้า โดยเฉพาะ Editorial/Massively ที่เป็นแนว `ed` ของเรา |
| **Start Bootstrap** | 16 ธีมฟรี — SB Admin 2, Personal, Freelancer, Agency, Creative, Grayscale, Clean Blog (+Angular, +Jekyll), Resume, Landing Page, Stylish Portfolio, Coming Soon, One Page Wonder, Business Casual, New Age | MIT | โครงหน้าแบบดั้งเดิม ไม่มีอะไรใหม่ทางเทคนิค |
| **Cruip** | 5 — Simple Light, Open, Mosaic Lite (React/Vue/Laravel) | ฟรี | Mosaic Lite = แดชบอร์ด เทียบกับ `analytics-overview` ของเราได้ |
| **Vercel Templates** | starter หลายร้อยตัว ครอบคลุม Next/React/Vue/Nuxt/Svelte/Astro/Remix/Hono/FastAPI/Go | ตามแต่ละ repo | เป็นโครงโปรเจกต์ ไม่ใช่ชิ้นส่วน — ตรงกับชั้น C ของเรา |

---

## หมวด E · Svelte (สำหรับ frontend ของ Aetox เอง ไม่ใช่คลังเทมเพลต)

`desktop/frontend` เป็น **Svelte 5 + Vite + CSS เปล่า ไม่มี Tailwind**
ข้อเท็จจริงนี้ตัดสามในสี่ตัวออกไปเลย

| ตัว | จำนวน | ต้องมี Tailwind | ใช้กับ Aetox ได้ไหม |
|---|---:|---|---|
| shadcn-svelte | 57 | **ใช่** | ต้องรับ Tailwind เข้ามาทั้งชุดก่อน |
| Skeleton v5 | 11 Tailwind + 32 framework (บน Zag.js) | **ใช่** | เหมือนกัน |
| Flowbite Svelte | ~60 | **ใช่** | เหมือนกัน |
| **Melt UI (next-gen)** | 18 builder — accordion, avatar, collapsible, combobox, dialog, file-upload, pin-input, popover, progress, radio-group, select, slider, spatialmenu, tabs, toaster, toggle, tooltip, tree | **ไม่** — headless ล้วน | **ตัวเดียวที่หย่อนเข้าไปได้ตรง ๆ** ให้พฤติกรรมกับการเข้าถึง แล้วเราทาสีเองด้วย CSS ที่มีอยู่ |

---

## ที่ข้ามไปเพราะเสียเงิน

| เว็บ | ของที่เสียเงิน | ราคา |
|---|---|---|
| Aceternity UI | บล็อกพรีเมียม 200+ ชิ้น + เทมเพลต 12 ชุด (ชื่อพวก `hero-section-with-*`, `pricing-with-switch`, `footer-with-*` ที่อยู่ใน `registry.json` แต่ไม่อยู่ในหน้า components) | $169/ปี · $199 ตลอดชีพ · $1,590 ทีม |
| Cult UI | pro.cult-ui.com — บล็อก AI, เทมเพลตฟูลสแตก | เสียเงิน |
| Magic UI | Magic UI Pro (pro.magicui.design) | เสียเงิน |
| Motion Primitives | Motion Primitives Pro | เสียเงิน |
| Preline | Preline Pro | เสียเงิน |
| Flowbite | Flowbite Pro — 450+ section | เสียเงิน |
| Start Bootstrap | SB Admin Pro, SB UI Kit Pro (+Angular/Vue) และบันเดิล | เสียเงิน |
| Cruip | ทุกอย่างนอกเหนือ 5 ตัวข้างบน | เสียเงิน |
| 21st.dev | ปุ่ม Copy prompt ต้องเป็นสมาชิก Builder | $6/เดือน |

---

## บันทึกจากการสำรวจ

**หลักฐานว่าชั้น A ไปได้ไกลกว่าที่คลังรุ่นแรกคิด** —
`zerasoftwarestudio.com/build` เป็นหน้าขายคอร์ส ไม่ใช่เว็บแจกเทมเพลต แต่งัดดูแล้ว
ทั้งหน้าไม่มี canvas ไม่มี WebGL ไม่มีวิดีโอสักตัว มีแต่ `animation-timeline`,
`@property`, `mask-image`, `clip-path`, `color-mix`, `backdrop-filter`, `100svh`
และ keyframes 28 ชุด — ทุกตัวอยู่ในรายการ "ต้องมีทางลง" ของ `STANDARD.md` ข้อ 2
อยู่แล้ว แปลว่าหน้าตาระดับนั้นทำได้ในชั้น A โดยไม่ต้องแตะไลบรารีสักตัว
(หน้านั้นมี lorem ipsum ค้างอยู่ในการ์ดหนึ่งด้วย — ข้อ 8 ของเราห้ามไว้)

**สิ่งที่ทุกเจ้าทำแล้วเราไม่ได้ทำ** — หนึ่งบทบาทมีหลายแบบ coss ui ชัดที่สุด:
ฐาน 55 ชิ้น แต่มีแบบย่อยให้เลือก 507 แบบ (`p-button-1` ถึง `p-button-41`)
HyperUI ก็เหมือนกัน หนึ่งหมวดมี 2–12 แบบ ของเราคือหนึ่งบทบาทหนึ่งไฟล์
ซึ่งตรงกับที่ `STANDARD.md` ข้อ 3 สั่งไว้ว่าต้องมีอย่างน้อยสามทิศทาง แต่ยังไม่ได้ทำ

**สิ่งที่ไม่มีใครทำแล้วเราทำอยู่** — กราฟที่วาดด้วย SVG ล้วนพร้อมตารางตัวเลขใน
`<details>` และ `<title>` ต่อ mark ทุกอัน HyperUI มีกราฟ 11 แบบแต่เป็นภาพนิ่ง
Preline/Flowbite โยนไปที่ ApexCharts ซึ่งเป็น CDN ไม่มีเจ้าไหนให้ทางออกเป็นตัวเลข
ตรงนี้คลังเราดีกว่าตลาด อย่าไปแก้
