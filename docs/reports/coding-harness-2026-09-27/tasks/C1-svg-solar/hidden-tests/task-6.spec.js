// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// 
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//     http://www.apache.org/licenses/LICENSE-2.0
// 
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

const { test, expect } = require('@playwright/test')
const { isExisted, expectTolerance, getOffsetByLocator } = require('@web-bench/test-util')
const { starData, density } = require('./util/util')

test.beforeEach(async ({ page }) => {
  await page.goto('/index.html')
})

test('planet revolutions', async ({ page }) => {
  await expect((await page.locator('animateMotion').count()) ?? 0).toBeGreaterThanOrEqual(8)
})

test('planet revolution params', async ({ page }) => {
  const rev = page.locator('animateMotion').nth(0)
  await expect(rev).toHaveAttribute('dur', `${starData.bodies[0].dur}s`)
  await expect(rev.locator('mpath')).toBeAttached()
  try {
    await expect(rev.locator('mpath')).toHaveAttribute('href', /#/i)
  } catch {
    await expect(rev.locator('mpath')).toHaveAttribute('xlink:href', /#/i)
  }
})

// svg performance problem
// test('planet revolution animation', async ({ page }) => {
//   await page.mouse.move(0, 0)
//   const planet = page.locator('.planet').nth(0)
//   const offset0 = await getOffsetByLocator(planet)
//   console.log({ offset0 })
//   // await page.waitForTimeout(3000)
//   // const offset1 = await getOffsetByLocator(planet)
//   // await expect(offset0.centerX).not.toBe(offset1.centerX)
//   // await expect(offset0.centerY).not.toBe(offset1.centerY)

//   let moved = false
//   for (let i = 0; i < 20; i++) {
//     await page.waitForTimeout(200) // Wait to observe movement
//     const offset1 = await getOffsetByLocator(planet)
//     console.log(i, offset1)
//     if (offset1.centerX !== offset0.centerX || offset1.centerY !== offset0.centerY) {
//       moved = true
//       break
//     }
//   }

//   await expect(moved).toBeTruthy()
// })

test('planet revolution pause', async ({ page }) => {
  const planet = page.locator('.planet').nth(2)
  const offset = await getOffsetByLocator(planet)
  await page.mouse.move(offset.centerX, offset.centerY)

  const offset0 = await getOffsetByLocator(planet)
  await page.waitForTimeout(500)
  const offset1 = await getOffsetByLocator(planet)
  await expectTolerance(offset0.centerX, offset1.centerX)
  await expectTolerance(offset0.centerY, offset1.centerY)
})
