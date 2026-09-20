import assert from 'node:assert/strict'
import test from 'node:test'
import worker from './_worker.js'

const origin = 'https://browser.lemon.vin'
const repo = '/repos/lemon-casino/trace-browser-release/releases'

function releaseFixture(version, id) {
  // The published Windows, macOS and Linux workflows produce these 17 files.
  const names = [
    `TraceBrowser-Setup-${version}-win-x64.exe`,
    `TraceBrowser-Setup-${version}-win-arm64.exe`,
    `TraceBrowser-Portable-${version}-win-x64.zip`,
    `TraceBrowser-Portable-${version}-win-arm64.zip`,
    `TraceBrowser-SelfUpdate-${version}-windows-amd64.zip`,
    `TraceBrowser-SelfUpdate-${version}-windows-arm64.zip`,
    `TraceBrowser-${version}-macos-amd64.dmg`,
    `TraceBrowser-${version}-macos-arm64.dmg`,
    `trace-browser_${version}_amd64.deb`,
    `trace-browser_${version}_arm64.deb`,
    `TraceBrowser-${version}-linux-amd64.tar.gz`,
    `TraceBrowser-${version}-linux-arm64.tar.gz`,
    'SHA256SUMS',
    `TraceBrowser-${version}-macos-amd64.sha256.txt`,
    `TraceBrowser-${version}-macos-arm64.sha256.txt`,
    `TraceBrowser-${version}-linux-amd64.sha256.txt`,
    `TraceBrowser-${version}-linux-arm64.sha256.txt`,
  ]
  return {
    id, tag_name: `v${version}`, name: `Trace Browser ${version}`,
    draft: false, prerelease: false, published_at: '2026-09-20T00:00:00Z',
    assets: names.map((name, index) => ({
      id: id * 100 + index, name, size: 1024, state: 'uploaded',
      browser_download_url: `https://github.com/lemon-casino/trace-browser-release/releases/download/v${version}/${name}`,
    })),
  }
}

function githubFixture(t, state) {
  const calls = []
  t.mock.method(globalThis, 'fetch', async (input, init = {}) => {
    const url = new URL(typeof input === 'string' ? input : input.url)
    calls.push({ url, init })
    if (url.hostname === 'github.com') return new Response('package', { headers: { 'Content-Type': 'application/octet-stream' } })
    assert.equal(url.hostname, 'api.github.com')
    if (url.pathname.endsWith('/latest')) {
      return Response.json(state.latest, { status: state.latestStatus || 200 })
    }
    if (url.pathname === repo) {
      return Response.json(state.pages?.[Number(url.searchParams.get('page')) - 1] || state.releases || [])
    }
    const release = [state.latest, ...(state.releases || [])].find(item => url.pathname === `${repo}/${item.id}`)
    assert.ok(release, `Unexpected upstream request: ${url}`)
    return Response.json(release)
  })
  return calls
}

function request(path, headers = {}) {
  return worker.fetch(new Request(origin + path, { headers }), {})
}

test('latest stays on the last complete release while any build is unfinished', async t => {
  const current = releaseFixture('2.1.60', 60)
  const previous = releaseFixture('2.1.59', 59)
  current.assets.pop()
  const calls = githubFixture(t, { latest: current, releases: [current, previous] })
  const response = await request('/api/trace-browser/latest')
  assert.equal(response.status, 200)
  const payload = await response.json()
  assert.equal(payload.version, '2.1.59')
  assert.equal(payload.releaseId, 59)
  assert.ok(payload.assets.every(asset => asset.downloadUrl.startsWith(`${origin}/download/trace-browser/release/59/asset/`)))
  assert.equal(calls.length, 2)
})

test('new version becomes visible only after the last asset finishes uploading', async t => {
  const latest = releaseFixture('2.1.60', 60)
  const previous = releaseFixture('2.1.59', 59)
  const finalAsset = latest.assets.at(-1)
  finalAsset.state = 'starter'
  githubFixture(t, { latest, releases: [latest, previous] })
  const before = await request('/api/trace-browser/latest')
  assert.equal(before.status, 200)
  assert.equal((await before.json()).version, '2.1.59')
  const etag = before.headers.get('ETag')
  const unchanged = await request('/api/trace-browser/latest', { 'If-None-Match': `W/${etag}` })
  assert.equal(unchanged.status, 304)
  finalAsset.state = 'uploaded'
  const after = await request('/api/trace-browser/latest', { 'If-None-Match': etag })
  assert.equal(after.status, 200)
  assert.equal((await after.json()).version, '2.1.60')
  assert.notEqual(after.headers.get('ETag'), etag)
})

test('every package and checksum is required, even when this platform is ready', async t => {
  const complete = releaseFixture('2.1.60', 60)
  for (const asset of complete.assets) {
    await t.test(asset.name, async t => {
      const latest = structuredClone(complete)
      latest.assets = latest.assets.filter(item => item.name !== asset.name)
      const previous = releaseFixture('2.1.59', 59)
      githubFixture(t, { latest, releases: [latest, previous] })
      const response = await request('/api/trace-browser/latest')
      assert.equal(response.status, 200)
      assert.equal((await response.json()).version, '2.1.59')
    })
  }
})

test('wrong-version, empty and unfinished assets cannot satisfy release readiness', async t => {
  for (const [name, change] of [
    ['wrong version', asset => { asset.name = asset.name.replace('2.1.60', '2.1.59') }],
    ['zero bytes', asset => { asset.size = 0 }],
    ['uploading', asset => { asset.state = 'starter' }],
    ['missing state', asset => { delete asset.state }],
    ['missing download URL', asset => { delete asset.browser_download_url }],
  ]) {
    await t.test(name, async t => {
      const latest = releaseFixture('2.1.60', 60)
      change(latest.assets[0])
      const previous = releaseFixture('2.1.59', 59)
      githubFixture(t, { latest, releases: [latest, previous] })
      const response = await request('/api/trace-browser/latest')
      assert.equal(response.status, 200)
      assert.equal((await response.json()).version, '2.1.59')
    })
  }
})

test('release list filters unfinished, draft and prerelease builds before applying the limit', async t => {
  const latest = releaseFixture('2.1.60', 60)
  latest.assets = latest.assets.slice(0, 1)
  const draft = { ...releaseFixture('2.1.62', 62), draft: true }
  const prerelease = { ...releaseFixture('2.1.61', 61), prerelease: true }
  const previous = releaseFixture('2.1.59', 59)
  githubFixture(t, { latest, releases: [draft, prerelease, latest, previous] })
  const response = await request('/api/trace-browser/releases?limit=1')
  assert.equal(response.status, 200)
  const payload = await response.json()
  assert.equal(payload.count, 1)
  assert.deepEqual(payload.releases.map(release => release.version), ['2.1.59'])
})

test('latest fallback skips draft and prerelease builds', async t => {
  const latest = releaseFixture('2.1.60', 60)
  latest.assets = []
  const previous = releaseFixture('2.1.59', 59)
  githubFixture(t, { latest, releases: [
    { ...releaseFixture('2.1.62', 62), draft: true },
    { ...releaseFixture('2.1.61', 61), prerelease: true },
    latest, previous,
  ] })
  const response = await request('/api/trace-browser/latest')
  assert.equal(response.status, 200)
  assert.equal((await response.json()).version, '2.1.59')
})

test('stable download uses the same complete release as update checks', async t => {
  const latest = releaseFixture('2.1.60', 60)
  latest.assets.pop()
  const previous = releaseFixture('2.1.59', 59)
  const calls = githubFixture(t, { latest, releases: [latest, previous] })
  const response = await request('/download/trace-browser/windows/amd64/installer')
  assert.equal(response.status, 200)
  assert.equal(await response.text(), 'package')
  assert.ok(calls.at(-1).url.pathname.includes('/v2.1.59/TraceBrowser-Setup-2.1.59-win-x64.exe'))
})

test('named downloads stay pinned to the selected complete release', async t => {
  const latest = releaseFixture('2.1.60', 60)
  latest.assets.pop()
  const previous = releaseFixture('2.1.59', 59)
  const calls = githubFixture(t, { latest, releases: [latest, previous] })
  const response = await request('/download/trace-browser/release/59/asset/SHA256SUMS')
  assert.equal(response.status, 200)
  assert.equal(await response.text(), 'package')
  assert.equal(calls[0].url.pathname, `${repo}/59`)
  assert.ok(calls.at(-1).url.pathname.includes('/v2.1.59/SHA256SUMS'))
  const incomplete = await request('/download/trace-browser/release/60/asset/SHA256SUMS')
  assert.equal(incomplete.status, 503)
})

test('legacy named download resolves the last complete release', async t => {
  const latest = releaseFixture('2.1.60', 60)
  latest.assets.pop()
  const previous = releaseFixture('2.1.59', 59)
  const calls = githubFixture(t, { latest, releases: [latest, previous] })
  const response = await request('/download/trace-browser/asset/SHA256SUMS')
  assert.equal(response.status, 200)
  assert.ok(calls.at(-1).url.pathname.includes('/v2.1.59/SHA256SUMS'))
})

test('no complete release does not advertise a partially built version', async t => {
  const latest = releaseFixture('2.1.60', 60)
  latest.assets = []
  githubFixture(t, { latest, releases: [latest] })
  const response = await request('/api/trace-browser/latest')
  assert.equal(response.status, 503)
  assert.equal(response.headers.get('Cache-Control'), 'no-store')
  const payload = await response.json()
  assert.equal(payload.ok, false)
  assert.equal(payload.version, undefined)
})

test('GitHub errors remain visible and are not mistaken for a completed release', async t => {
  githubFixture(t, { latest: { message: 'API unavailable' }, latestStatus: 503 })
  const response = await request('/api/trace-browser/latest')
  assert.equal(response.status, 502)
  assert.equal((await response.json()).ok, false)
})

test('Chromium release lookup retains its independent asset requirements', async t => {
  const latest = { id: 99, tag_name: 'v140.0.0', assets: [{ name: 'chromium-windows-amd64.zip', size: 100 }] }
  const calls = githubFixture(t, { latest })
  const response = await request('/api/chromium/latest')
  assert.equal(response.status, 200)
  assert.equal((await response.json()).version, '140.0.0')
  assert.equal(calls.length, 1)
})

test('historical lookup continues past an incomplete first page', async t => {
  const latest = releaseFixture('2.1.60', 60)
  latest.assets = []
  const previous = releaseFixture('2.1.59', 59)
  const calls = githubFixture(t, { latest, pages: [Array(100).fill(latest), [previous]] })
  const response = await request('/api/trace-browser/latest')
  assert.equal(response.status, 200)
  assert.equal((await response.json()).version, '2.1.59')
  assert.equal(calls.at(-1).url.searchParams.get('page'), '2')
})

test('historical lookup is bounded when every release is incomplete', async t => {
  const latest = releaseFixture('2.1.60', 60)
  latest.assets = []
  const calls = githubFixture(t, { latest, releases: Array(100).fill(latest) })
  const response = await request('/api/trace-browser/latest')
  assert.equal(response.status, 503)
  assert.equal(calls.length, 6)
})

test('old assets attached to a complete release never replace its current packages', async t => {
  const latest = releaseFixture('2.1.60', 60)
  latest.assets.unshift(releaseFixture('2.1.59', 59).assets[0])
  const calls = githubFixture(t, { latest })
  const response = await request('/api/trace-browser/latest')
  assert.equal(response.status, 200)
  const payload = await response.json()
  assert.equal(payload.version, '2.1.60')
  assert.equal(payload.assets.length, 17)
  assert.ok(payload.assets.every(asset => !asset.name.includes('2.1.59')))
  const download = await request('/download/trace-browser/windows/amd64/installer')
  assert.equal(download.status, 200)
  assert.ok(calls.at(-1).url.pathname.includes('/v2.1.60/TraceBrowser-Setup-2.1.60-win-x64.exe'))
})

test('complete release is returned immediately without a historical lookup', async t => {
  const calls = githubFixture(t, { latest: releaseFixture('2.1.60', 60) })
  const response = await request('/api/trace-browser/latest')
  assert.equal(response.status, 200)
  assert.equal((await response.json()).version, '2.1.60')
  assert.equal(calls.length, 1)
})
