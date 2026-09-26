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
const { isExisted, getOffsetByLocator } = require('@web-bench/test-util')
const { starData, density } = require('./util/util')

test.beforeEach(async ({ page }) => {
  await page.goto('/index.html')
})

test('data', async ({ page }) => {
  await expect(starData).toMatchObject({ r: 8, color: '#ffff00' })
})

test('sun', async ({ page }) => {
  await expect(page.locator('circle.sun')).toBeVisible()
})

test('sun attributes', async ({ page }) => {
  await expect(page.locator('.sun')).toHaveAttribute('r', `${starData.r}`)
  await expect(page.locator('.sun')).toHaveAttribute('fill', starData.color)
})

test('sun layout', async ({ page }) => {
  const offset = await getOffsetByLocator(page.locator('.sun'))
  await expect(offset).toMatchObject({ centerX: 80 * density, centerY: 80 * density })
})
