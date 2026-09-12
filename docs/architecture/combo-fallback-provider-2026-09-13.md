# สลับ provider ข้ามบัญชีอัตโนมัติเมื่อโควต้าหมด — Combo Fallback Provider (2026-09-13)

Direction document, เขียนก่อนลงมือ — ยังไม่มีบรรทัดโค้ดของฟีเจอร์นี้อยู่ในต้นไม้ ตัดสินใจ
เก็บไว้ทำ **เวอร์ชันถัดไป** ตามที่เจ้าของสั่ง ("จดเป็น doc ไว้เราจะทำในเวอร์ชั่นถัดไป",
13 ก.ย. 2026) หลังจากไปศึกษา [9Router](https://github.com/decolua/9router) (ตัวมัน
เป็น proxy server แยกต่างหาก, Node/Next.js, MIT) แล้วเทียบว่า Aetox ขาดอะไรจริง ๆ

Status: ทุกอย่างในเอกสารนี้เป็น `Proposed` ล้วน — ยังไม่มี Phase ใดเริ่ม §7 ไว้บันทึก
เมื่อเริ่มทำจริง

Live code ที่เกี่ยวข้อง (อ่านแล้วตอนร่างเอกสารนี้ ทุกอย่าง `Direct`):
[`internal/model/types.go`](../../internal/model/types.go) ·
[`internal/model/bootstrap.go`](../../internal/model/bootstrap.go) ·
[`internal/model/httpclient.go`](../../internal/model/httpclient.go) ·
[`internal/model/quota.go`](../../internal/model/quota.go) ·
[`internal/bootstrap/bootstrap.go`](../../internal/bootstrap/bootstrap.go) ·
[`internal/config/config.go`](../../internal/config/config.go) ·
[`internal/cognitive/agent.go`](../../internal/cognitive/agent.go) ·
[`desktop/provider_for.go`](../../desktop/provider_for.go) ·
[`desktop/provider_forward.go`](../../desktop/provider_forward.go) ·
[`desktop/screen.go`](../../desktop/screen.go) ·
[`desktop/app.go`](../../desktop/app.go)

---

## 0. ทำไม

เจ้าของถามให้ไปดู 9Router ว่ามีประโยชน์กับ Aetox ไหม จุดขายหลักของมันที่ Aetox ยังไม่มี
จริง ๆ คือ **combo** — ตั้งลำดับ tier ของ provider แล้วสลับอัตโนมัติเมื่อ tier บนสุด
โควต้าหมด (subscription หลัก → provider ถูก → free tier) ส่วนอย่างอื่นของ 9Router
(แปลงฟอร์แมต OpenAI/Anthropic/Gemini, quota/balance tracking, OAuth หลายเจ้า) Aetox
ทำเองอยู่แล้วใน [internal/model](../../internal/model/README.md) เทียบเท่าหรือดีกว่า
(ฝังในแอปเดียว ไม่ต้องรัน proxy แยก)

ส่วนที่ 9Router ทำแต่**ไม่ควรลอก**: เอา OAuth ของ subscription หนึ่ง (เช่น Claude Code
Pro) ไปแชร์ให้เครื่องมืออื่นยิงผ่าน proxy — เข้าข่ายขัด ToS ของผู้ให้บริการ ไม่เอามาทำ

## 1. ขอบเขตที่ตัดสินใจแล้ว

**"ข้ามบัญชี" ในที่นี้คือข้าม provider ไม่ใช่หลายบัญชีของ provider เดียวกัน** — ระบบเก็บ
credential ปัจจุบันเก็บได้บัญชีเดียวต่อ provider (`SetAPIKey`/oauth sign-in ทับของเก่า
ทันที, ดู [forgetQuotas](../../desktop/app.go) comment เรื่องทำไมต้องลบโควต้าเก่าไม่ใช่
เก็บเป็นค่าว่าง) การรองรับสองคีย์ของ provider เดียวกันต้องขยาย credential storage เป็น
list ก่อน — เป็นงานคนละก้อน ไม่อยู่ในดีไซน์นี้

Chain ที่ออกแบบคือ**เรียงลำดับชื่อ provider** (เช่น `["anthropic", "z-ai", "kiro"]`)
แต่ละตัวใช้ credential/model/baseURL ที่ตั้งไว้ของ provider นั้นอยู่แล้ว ไม่มีการเก็บคีย์
ซ้ำที่ไหนใหม่

## 2. ของที่มีอยู่แล้วพอดี — ใช้ต่อได้เลยไม่ต้องเขียนใหม่

- [`desktop/provider_for.go:29`](../../desktop/provider_for.go) —
  `App.providerFor(cfg)` สร้าง `model.Provider` จากชื่อ provider ใดก็ได้แบบครบ
  (key/oauth/baseURL/transport resolve ให้หมด) อยู่แล้ว ปัจจุบันใช้กับ sub-agent ที่ระบุ
  `provider:` ของตัวเองใน AGENT.md — เรียกซ้ำทีละ tier ได้ทันที
- [`internal/model/types.go`](../../internal/model/types.go) `Provider` interface มีแค่
  `Name()` + `Complete()` (`StreamingProvider` แยกต่างหาก) → ห่อเป็น wrapper ใหม่ได้โดย
  ไม่กระทบ [`internal/cognitive/agent.go`](../../internal/cognitive/agent.go) เลย เพราะ
  agent เห็นแค่ `model.Provider` ตัวเดียว ไม่รู้ว่าข้างในเป็น chain
- [`internal/model/httpclient.go`](../../internal/model/httpclient.go) —
  `outOfCreditsError`/`providerDownError` แยก "หมดเครดิต" vs "provider ล่ม" vs error
  อื่นอยู่แล้ว แค่ต้องทำให้ chain แยกแยะได้จาก error value ที่คืนออกมา
- [`internal/model/quota.go`](../../internal/model/quota.go) — `Quota`/`quotaObserver`
  ที่มีอยู่แล้วใช้โชว์สถานะบนจอได้ทันที ไม่ต้องเดินสายใหม่

## 3. ชิ้นที่ต้องสร้างใหม่

### 3.1 `ChainProvider` — ไฟล์ใหม่ `internal/model/chain.go`

ห่อ `[]tier{name string, provider Provider}` เรียงลำดับ implement `Provider` +
`StreamingProvider` โดย delegate ไปทีละ tier:

- ก่อนเรียก tier ใด เช็ค `bannedUntil[tierName]` (map ในหน่วยความจำล้วน ไม่ persist —
  ธรรมชาติเดียวกับ `Quota` ที่ "ไม่เคย fetch มีแต่สังเกต") ถ้ายังไม่พ้นเวลาแบน ข้าม tier
  นั้นไปเลย
- เรียก tier ปัจจุบันด้วย **request เดิม** (`agent.go` เรียก `Complete`/`StreamComplete`
  ด้วย `req` ตัวเดียวอยู่แล้ว ไม่ต้อง build ใหม่)
- error เป็น "หมดโควต้า/บัญชีใช้ไม่ได้" (ดู §3.2) → แบน tier นี้ (cooldown แบบ backoff เช่น
  5 → 15 → 30 นาที) แล้วลอง tier ถัดไปด้วย request เดิมทันที
- error เป็นอย่างอื่น (tool schema ผิด, context เกิน ฯลฯ) → **ไม่สลับ** ปล่อย error
  ออกไปตามเดิม กันบั๊กจริงถูกกลบด้วยการเด้งไป tier อื่นแล้วเปลืองโควต้าที่นั่นตามไปอีก
- **กฎ streaming**: สลับ tier ได้เฉพาะตอนที่ยังไม่มี `onChunk` ไหนถูกเรียกเลยในความ
  พยายามนี้ (แทร็ก flag `started` ต่อการลองแต่ละครั้ง) — ถ้า error โผล่หลังจากเนื้อความ
  เริ่มไหลออกจอแล้ว **ห้ามสลับ** ต้องรายงาน error ตรง ๆ เหมือนตอนนี้ หลักการเดียวกับ
  [`tool_truncation.go`](../../internal/model/tool_truncation.go) — "รันต่อไปทั้งที่
  ผิดคือผลลัพธ์ที่แย่ที่สุด"
- ทุก tier ถูกแบนหมด → คืน error สุดท้าย (พฤติกรรมเดิมของ single-provider ตอนพัง)
- tier ที่พ้นเวลาแบนแล้วถูกลองก่อนเสมอ (ลำดับความสำคัญคงเดิม ไม่ sticky อยู่ tier ต่ำ
  ตลอดไป — เมื่อ tier 1 กลับมาใช้ได้ ให้กลับไปใช้ tier 1 ก่อน)

### 3.2 Sentinel error ให้ chain แยกแยะได้

แก้ [`httpclient.go`](../../internal/model/httpclient.go) เพิ่ม
`var ErrAccountExhausted = errors.New(...)` แล้วห่อด้วย `%w` ใน `outOfCreditsError`
และเคส 5xx ที่หมด retry แล้วใน `providerDownError` — chain เช็คด้วย
`errors.Is(err, model.ErrAccountExhausted)`

### 3.3 Config field

[`internal/config/config.go`](../../internal/config/config.go) เพิ่ม
`ModelFallbackChain []string` — แค่ชื่อ provider เรียงลำดับ **ไม่มีคีย์ในนี้เลย** ตรงกับ
กติกา §248 A4 ที่คอมเมนต์ในไฟล์นี้บอกไว้แล้วว่า `Config` ห้ามถือ key (เหตุผลเดียวกับที่
ทำให้ remote-engine ปลอดภัย — ดู
[remote-engine-2026-09-11.md](remote-engine-2026-09-11.md) §1 ข้อ 3)

### 3.4 จุดประกอบร่าง

[`internal/bootstrap/bootstrap.go`](../../internal/bootstrap/bootstrap.go) เพิ่ม
`Options.Provider model.Provider` (ออปชันแทนของเดิม) — ถ้าตั้งมา `Engine()` ใช้ตัวนี้
แทนการเรียก `model.BootstrapProvider` เอง (บรรทัดเดิมที่เรียก
`model.BootstrapProvider` ยังอยู่เหมือนเดิมสำหรับ path ปกติที่ไม่ตั้ง chain)

ฝั่ง desktop เพิ่มฟังก์ชันเล็ก ๆ ก่อนเรียก `bootstrap.Engine` (ใกล้จุดที่
[`desktop/app.go`](../../desktop/app.go) ตั้ง `ProviderTransport` วันนี้):

```go
if len(cfg.ModelFallbackChain) > 1 {
    opts.Provider = a.buildComboProvider(cfg) // nil ถ้าสร้างไม่ได้แม้แต่ tier เดียว
}
```

`buildComboProvider` วนเรียก `a.providerFor(cfg)(name)` ทีละชื่อใน
`ModelFallbackChain` เก็บที่สร้างสำเร็จ ห่อเป็น `model.NewChainProvider(...)` —
**ผู้ใช้ที่ไม่ตั้งค่านี้ไม่กระทบอะไรเลย** เพราะ path เดิมยังวิ่งตรงเหมือนเดิม 100%

### 3.5 โชว์บนจอ

เก็บ tier ที่ active ไว้ใน `ChainProvider` (mutex-guarded) แล้ว expose ผ่านเมธอดใหม่ให้
`GetModelInfo()`/`ModelStatus()` เติมบรรทัดเช่น "ใช้ tier 2/3 (Claude หมดโควต้า
รีเซ็ต 12 นาที)" — **โชว์ที่จอเท่านั้น ไม่ยัดเข้า context ของโมเดล** ทุกรอบ (ตรงกับที่
เคยสรุปไว้ว่าการ narrate ทุกรอบมันแพงแบบ quadratic — [goal-mode-reports-little])

### 3.6 UI ตั้งค่า

Settings.svelte เพิ่ม section เล็ก ๆ ให้ลาก/เรียงลำดับจาก provider ที่ "พร้อมใช้" อยู่
แล้ว (`EnabledProviders()`) เป็น chain — ไม่ต้องมี UI ใหม่สำหรับคีย์ เพราะยืมของเดิม
ทั้งหมด

## 4. คำถามที่ยังไม่ตัดสินใจ — ต้องเคาะก่อนเริ่ม Phase 1

1. **ตอน silent fallback ไป tier 2 แล้ว ควรเปลี่ยนชื่อโมเดลที่โชว์บน TopBar/avatar
   ทันทีไหม** — เอียงไปทาง "ควร" (สอดคล้องกับปรัชญาที่ไม่โชว์อะไรไม่จริงเกี่ยวกับบัญชีที่
   ใช้อยู่) แต่จะกระทบ dropdown thinking-level/vision ที่ populate จาก capability ของ
   tier 1 ตอน dial — ถ้า tier 2 ความสามารถไม่เท่ากัน UI อาจค้างปุ่มที่กดไม่ได้จริงจนกว่า
   จะเริ่ม session ใหม่ ต้องเลือกว่ารับข้อจำกัดนี้ไปก่อนใน v1 หรือรอ solve เต็มรูปแบบ
   (populate ใหม่ทุกครั้งที่ tier เปลี่ยน)
2. **cooldown ตายตัว (5/15/30 นาที) พอไหม หรืออยากอ่าน `ResetAt` จริงจาก response มา
   คำนวณ** — แบบหลังแม่นกว่าแต่ต้องเดินสาย `Quota` เข้า `ChainProvider` เพิ่มอีกชั้น
   (ข้าม package boundary ที่ตอนนี้ `internal/model` ไม่รู้จัก `a.quotas` ของ desktop)
3. **pre-flight skip จากโควต้าที่เคยเห็นตอนต้น session** — v1 ที่ออกแบบไว้ไม่เช็ค
   `a.quotas` ก่อนเรียก tier 1 ตอน dial (chain รู้จักแบนเฉพาะจากความล้มเหลวที่เกิดใน
   session นี้เอง) แปลว่าข้อความแรกหลังเปิดแอปใหม่อาจเสียเวลาลอง tier ที่รู้อยู่แล้วว่า
   หมดโควต้าหนึ่งครั้ง — เป็น gap เล็ก ยอมรับได้ใน v1 หรืออยากปิดตั้งแต่แรกด้วย callback
   `QuotaHint` ที่ desktop ส่งเข้ามา

## 5. ลำดับงานที่แนะนำเมื่อเริ่มทำ

1. `ChainProvider` + sentinel error ก่อน — ทดสอบแยกได้ทั้งหมดด้วย unit test ใน
   `internal/model` โดยไม่ต้องแตะ UI หรือ desktop เลย
2. `Config.ModelFallbackChain` + จุดประกอบร่างใน `bootstrap.Engine`/`desktop/app.go`
   — ยังไม่มี UI ตั้งค่า ทดสอบด้วยการเซ็ต config ตรง ๆ ก่อน
3. โชว์สถานะบนจอ (`GetModelInfo`/`ModelStatus`)
4. UI ตั้งค่าใน Settings.svelte เป็นลำดับสุดท้าย

## 6. บันทึกความคืบหน้า

ยังไม่เริ่ม — เอกสารนี้เขียนไว้ล่วงหน้าเพื่อทำในเวอร์ชันถัดไปตามที่เจ้าของขอ
