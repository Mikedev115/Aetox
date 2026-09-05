---
name: aetox-web-templates
before: writing a web page or a screen as an .html file that is not a slide deck
description: กติกาของหน้าเว็บที่ Aetox ยอมส่งออกไป — สัญญาที่ทุกหน้าต้องผ่าน (ไฟล์เดียวจบ · responsive โดยไม่ต้องจดเบรกพอยต์ · ไม่ใช้สคริปต์ถ้าไม่จำเป็นต่อความหมาย · landmark จริง · โทเคนสีสำหรับสว่างและมืด · กราฟเป็น SVG ในไฟล์ · เข้าถึงได้ตั้งแต่โครง) พร้อมมาตรฐานใน STANDARD.md และหน้าตัวอย่างเต็มหน้าอีก 6 ใบใน pages/ **นี่คือกติกา ไม่ใช่คลังโค้ดให้แปะ** ออกแบบและเขียนเอง แล้วเอากติกามาตรวจ
---

# Web templates

**กติกา ไม่ใช่มาร์กอัปให้คัดลอก**

คลังส่วนย่อย 104 ชิ้นกับ kit ถูกถอดออกเมื่อ 5 ก.ย. 2569 ด้วยเหตุผลที่ `STANDARD.md`
เปิดหัวไว้เอง: **เทมเพลตที่เข้ารหัสค่ากลางไม่ได้ช่วยโมเดลที่เก่งอยู่แล้ว มันตรึงให้อยู่ที่ค่ากลาง**
ตัวเลขที่ตัดสิน — ส่วนย่อยเฉลี่ย 11.7 KB ต่อไฟล์ เทียบชั้นวางสไลด์ที่ 2.2 KB
ที่ 2 KB โมเดลอ่านแล้วเขียนเอง ที่ 11.7 KB โมเดลแปะ และสิ่งที่แปะคือค่าเฉลี่ยของเมื่อวาน
เทสที่คุมคลังนั้นก็สารภาพเรื่องเดียวกัน: มันวัด responsive มั้ย มี `<nav>` มั้ย
ตั้งชื่อคลาสมีคำนำหน้ามั้ย — ซึ่ง**หน้าที่เขียนสดจากศูนย์ผ่านทุกข้อ** ตามข้อ 0 ไฟล์พวกนั้นจึงไม่ควรมีอยู่

สิ่งที่เหลืออยู่คือสิ่งที่เจนสดไม่ได้จริง ๆ: **ข้อกำหนดที่ตรวจได้** กับ **ค่าที่วัดมาแล้ว**

## วิธีใช้สกิลนี้

ออกแบบหน้าเอง เขียนเอง แล้วเอาสัญญาข้างล่างมาตรวจก่อนส่ง
สกิลนี้ไม่ตัดสินว่าหน้าตาควรเป็นยังไง — `aetox-frontend-design` ตัดสินเรื่องนั้น
และ `aetox-design-system` ถือชั้นโทเคนถ้าหน้านั้นต้องมีพาเลตต์ของตัวเอง

## หน้าตัวอย่างเต็มหน้า — `pages/` 6 ใบ

| ไฟล์ | สำหรับ | ขนาด |
|---|---|---:|
| `pages/fintech-saas.html` | SaaS · แอปการเงิน | 301 KB |
| `pages/homestay.html` | ที่พัก · โฮมสเตย์ | 192 KB |
| `pages/restaurant.html` | ร้านอาหาร · คาเฟ่ | 181 KB |
| `pages/studio-agency.html` | เอเจนซี · สตูดิโอ | 166 KB |
| `pages/portfolio.html` | พอร์ตโฟลิโอ · ฟรีแลนซ์ | 149 KB |
| `pages/fashion-editorial.html` | แฟชั่น · แบรนด์งานฝีมือ | 118 KB |

**ทุกใบใหญ่กว่าที่ `skill_view` ส่งได้ในครั้งเดียว (64 KB)** ผลลัพธ์ที่ถูกตัดจะบอก
`offset` ที่ต้องอ่านต่อ แต่ก่อนจะเปิด ให้ชั่งก่อนว่าคุ้มไหม — ใบเดียวกินคอนเท็กซ์
เท่ากับงานทั้งวัน และเปิดมาแล้วยังเขียนกลับในรอบเดียวไม่ได้อยู่ดี

**ใช้มันแบบตัวอย่าง ไม่ใช่แบบโครงที่ต้องแก้** อ่านส่วนที่อยากรู้ว่าเขาแก้ปัญหานั้นยังไง
แล้วเขียนของตัวเอง อย่าเปิดทั้งใบเพื่อแก้คำสามคำ

## มาตรฐาน

`STANDARD.md` ในโฟลเดอร์นี้คือเส้นที่ทุกอย่างต้องผ่าน **อ่านก่อนเขียนหน้า** ไม่ใช่อ่านเฉพาะ
ตอนจะเพิ่มไฟล์กลับเข้ามา — ข้อ 4 ถือคลังคำศัพท์การเคลื่อนไหว ข้อ 5 ถือกับดักที่พังจริง
สองข้อนั้นคือของที่หน้าเขียนสดต้องใช้ ไม่ใช่ของที่คนดูแลโฟลเดอร์ใช้
ย่อเหลือประโยคเดียว: **เทมเพลตที่เข้ารหัสค่ากลางไม่ได้ช่วยโมเดลที่เก่งอยู่แล้ว มันตรึงให้อยู่ที่ค่ากลาง**
ไฟล์ที่จะกลับเข้ามาได้ต้องถือของที่เจนสดไม่ได้ — ค่าที่วัดมาแล้ว เทคนิคที่ต้องลองสิบรอบ
หรือทิศทางศิลป์ที่เป็นการเลือกจริง ไม่ใช่ค่าเริ่มต้น ในนั้นยังมีคลังคำศัพท์การเคลื่อนไหว
ทิศทางศิลป์สี่แบบ และบันทึกกับดักที่พังจริงในเบราว์เซอร์นี้

## สัญญาที่ทุกหน้าต้องผ่าน

- **One self-contained file.** No framework, no CDN, no build step, no
  `node_modules`. The page opens from disk, from a USB stick, from an email
  attachment. Same reason a deck is one file.
- **Responsive without breakpoint bookkeeping.** `clamp()` for type and space,
  `repeat(auto-fit,minmax(min(280px,100%),1fr))` for grids, so the column count
  follows the content and the viewport rather than a list of widths somebody has
  to maintain. Media queries appear only where the layout genuinely changes
  shape, such as the dashboard sidebar becoming a top scroller.
- **No script unless the semantics cannot be had without it.** The FAQ is
  `<details>`, the mobile menu is a checkbox. Both are keyboard operable and
  survive a JS error, which a scripted version of either does not. The one
  exception worth making is a theme toggle: a theme choice that resets on every
  reload has no script-free equivalent worth calling a feature, so it carries a
  small inline script rather than faking one.
- **Real landmarks and real elements.** `<nav>`, `<main>`, `<footer>`,
  `<blockquote>` with `<cite>`, `<ol>` when the order is the meaning, `<table>`
  with `scope` on every header. This is not decoration; it is what a screen
  reader navigates by.
- **Every class is namespaced `w-`.** `aetox-frontend-design` warns that
  generated CSS most often breaks when a class selector and an element selector
  fight over the same padding. Two templates pasted into one page must not be
  able to reach into each other.
- **Light and dark from tokens**, defined once in the shell. No template hard
  codes a colour.
- **Charts are SVG and CSS in the file.** No charting library, no CDN, for the
  same reason as everything else here: a page whose picture arrives from
  somewhere else is a page with a hole in it when that somewhere else is down.
  Series colours come from `--viz-1` … `--viz-6` in the shell, in that fixed
  order, never cycled; the ramp `--viz-s1` … `--viz-s5` carries magnitude. Those
  six ran through the `dataviz` skill's validator in both modes — lightness
  band, chroma floor, colour-blind separation of adjacent pairs, contrast
  against the surface. Change a value and run it again rather than eyeballing.
  Every mark carries its own `<title>`, so the browser draws the tooltip and a
  screen reader reads the number without a line of script, and every chart ships
  the same numbers as a table inside `<details>`.
- **Accessible by construction.** Visible focus ring, a skip link, labels that
  stay visible when the field is filled, `prefers-reduced-motion` respected, and
  status never carried by colour alone.
- **The page moves at least once.** A section that sits perfectly still needs a
  written reason; stillness with no reason is a page that reads as a printout.
  The numbers here are measured, not chosen — 338 entrance transitions across the
  24 published originals in the reference folder. **94% are springs** (318 of
  338), and **73% ride zero bounce** (248), so the default entrance is a spring
  that arrives without overshooting; keep a small bounce for one hero moment, not
  for a grid of cards. Duration clusters at **0.9s** (0.6s for something small,
  1.5s for a hero). Siblings **stagger 100ms apart**, in the order somebody reads
  them. What moves is **opacity and `translateY` only** — 331 of 338 keep
  `scale:1`, so a card that enters rises, it does not grow. Offsets are 50px for
  a block, 10–20px for a line of text. Note the gap against `STANDARD.md` ข้อ 4,
  which says 0.45–0.7s and 60–90ms: those are the older numbers and the source is
  slower and calmer than the rule.
- **Never start an entrance at `opacity:0`.** The originals begin at `.001` in
  316 of 338 entrances, which keeps the element composited so the first animated
  frame does not pop. This is also the cheap way past the trap in `STANDARD.md`
  ข้อ 5: `opacity:0` plus `fill-mode:both` plus a delay renders a blank first
  frame, and on a hero that blank frame is the page's own thumbnail. Whatever the
  start value, the rest state must be the finished page — someone who turns
  motion off sees everything a person with it on sees, just not moving.

## เมื่อแก้ของที่มีอยู่แล้ว

**ออกแบบใหม่ให้ของที่มีอยู่** อ่านหน้าจริงหรือคอมโพเนนต์จริงก่อนเสมอ — เปิดไฟล์ที่
เรนเดอร์บนเส้นทางนั้นจริง ๆ ไม่ใช่ไฟล์ข้าง ๆ ที่หน้าตาคล้ายกัน แล้วยกสิ่งที่มันพูดและ
ทำอยู่มาด้วย หน้าที่เขียนจากการเดาว่าของเดิมมีอะไร ไม่ใช่การออกแบบใหม่ มันคือหน้าคนละหน้า
ที่บังเอิญอยู่ URL เดียวกัน

**แก้ต่อจากที่เพิ่งทำ** จับคู่การแก้ให้ตรงกับสิ่งที่ถูกขอ คำผิดหรือตัวเลขผิดคือการแก้
บรรทัดเดียว ไม่ใช่เขียนส่วนนั้นใหม่ทั้งก้อน คำขอที่อยากได้ทิศทางอื่น ("ลองฮีโร่ที่กล้ากว่านี้")
คุ้มที่จะทำสองสามแบบวางเทียบกัน เพราะเทียบถูกกว่าเดาแล้วเจนใหม่ คำขอที่อยากขัดเกลา
ของเดิมต่อ ยังเป็นการแก้เล็ก ๆ ตรงจุดในมาร์กอัปเดิม สับสนสามอย่างนี้เสียมากกว่าที่ประหยัด —
เขียนทั้งส่วนใหม่เพื่อแก้คำผิดคือการทิ้งทุกอย่างที่มันทำถูกอยู่แล้ว

ไม่ว่าจะแก้แบบไหน ระบบโทเคนที่เลือกไว้ตอนต้นยังบังคับอยู่ การแก้รอบหลังที่หยิบสีหรือ
ฟอนต์นอกระบบนั้น คือการไหลกลับไปหาค่าเริ่มต้นทั่ว ๆ ไปที่ `aetox-frontend-design` เตือนไว้
ถ้าพาเลตต์ต้องเปลี่ยนจริง นั่นคือการแก้แผน — ตั้งใจและเรียกชื่อมัน ไม่ใช่ปล่อยให้ค่าหนึ่ง
เล็ดลอดเข้ามาตอนแก้ครั้งที่สาม

## Not in here

Video, because Aetox produces none; `video_ocr` reads one. Slides, which are
`aetox-slide-templates` and a different contract entirely. Documents and sheets,
which leave through `doc_write` and `sheet_write` as OOXML and are not HTML at
all. If any of those earn templates, they earn a skill of their own rather than
a folder in here: a template nobody can tell the medium of is a template that
gets pasted into the wrong file.
