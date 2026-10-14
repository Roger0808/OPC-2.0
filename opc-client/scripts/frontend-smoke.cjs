// Browser QA against an explicitly configured development relay.
// Credentials are read from a private external file and are never printed.
const fs = require('node:fs')
const path = require('node:path')
const assert = require('node:assert/strict')
const { chromium } = require(path.resolve(__dirname, '../../website-docs/node_modules/playwright-core'))

async function main() {
  if (!process.env.OPC_SMOKE_ACCOUNT_FILE) throw new Error('Set OPC_SMOKE_ACCOUNT_FILE to a private local test account JSON')
  const account = JSON.parse(fs.readFileSync(process.env.OPC_SMOKE_ACCOUNT_FILE, 'utf8'))
  const browser = await chromium.launch({ executablePath: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', headless: true })
  try {
    const page = await browser.newPage({ locale: 'zh-CN', viewport: { width: 1280, height: 800 } })
    const errors = [], apiRequests = []
    page.on('pageerror', e => errors.push(e.message))
    page.on('request', r => { if (new URL(r.url()).pathname.startsWith('/api/v1/')) apiRequests.push({ url: r.url(), headers: r.headers() }) })
    await page.goto('http://127.0.0.1:5174/login')
    await page.locator('.form-card input[type="text"]').first().fill(account.email)
    await page.locator('.form-card input[type="password"]').fill(account.password)
    await page.locator('.form-card button[type="submit"]').click()
    await page.waitForURL('**/platform/knowledge-bases', { timeout: 30000 })
    await page.getByRole('button', { name: '服务端连接', exact: true }).click()
    await page.getByRole('button', { name: '检测当前连接', exact: true }).click()
    await page.getByText('当前服务端连接正常').waitFor()
    const state = await page.evaluate(() => {
      const keys = Object.keys(localStorage)
      return {
        publicOrigin: window.__OPC_CLIENT_RUNTIME__.publicOrigin,
        transport: window.__OPC_CLIENT_RUNTIME__.transportBase,
        tokenInMemory: !!localStorage.getItem('weknora_token'),
        persistedToken: keys.some(k => k.endsWith('weknora_token') || k.endsWith('weknora_refresh_token')),
        lite: localStorage.getItem('weknora_lite_mode'),
      }
    })
    assert.equal(state.publicOrigin, 'http://127.0.0.1:8080')
    assert.equal(state.tokenInMemory, true)
    assert.equal(state.persistedToken, false)
    assert.equal(state.lite, null)
    assert.equal(apiRequests.some(r => r.url.includes('/auth/auto-setup')), false)
    assert.ok(apiRequests.some(r => r.url.includes('/auth/me')))
    for (const r of apiRequests) {
      assert.equal(new URL(r.url).origin, state.transport)
      assert.ok(r.headers['x-opc-client-session'])
    }
    assert.deepEqual(errors, [])
    // Same-tab reload should lose memory-only web-preview auth. A native build restores from its OS store.
    await page.reload()
    await page.waitForURL('**/login', { timeout: 30000 })
    console.log(JSON.stringify({ passed: true, checks: ['password login', 'auth/me', 'knowledge-base navigation', 'connection probe', 'all APIs through real relay', 'no plaintext tokens', 'no Lite auto-setup', 'memory-only development reload'], apiRequests: apiRequests.length, pageErrors: errors.length }))
  } finally { await browser.close() }
}
main().catch(e => { console.error(e.message); process.exitCode = 1 })
