import assert from 'node:assert/strict'
import test from 'node:test'
import worker from './_worker.js'

const origin = 'https://browser.lemon.vin'
const repo = '/repos/lemon-casino/trace-browser-release/releases'

function releaseFixture(version, id) {
  // Each architecture uploads its own packages and checksum file.
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
    `TraceBrowser-${version}-windows-amd64.sha256.txt`,
    `TraceBrowser-${version}-windows-arm64.sha256.txt`,
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

const targets = [
  ['windows', 'amd64', 'installer', 'TraceBrowser-Setup-{version}-win-x64.exe'],
  ['windows', 'arm64', 'installer', 'TraceBrowser-Setup-{version}-win-arm64.exe'],
  ['windows', 'amd64', 'selfupdate', 'TraceBrowser-SelfUpdate-{version}-windows-amd64.zip'],
  ['windows', 'arm64', 'selfupdate', 'TraceBrowser-SelfUpdate-{version}-windows-arm64.zip'],
  ['macos', 'amd64', 'installer', 'TraceBrowser-{version}-macos-amd64.dmg'],
  ['macos', 'arm64', 'installer', 'TraceBrowser-{version}-macos-arm64.dmg'],
  ['linux', 'amd64', 'portable', 'TraceBrowser-{version}-linux-amd64.tar.gz'],
  ['linux', 'arm64', 'portable', 'TraceBrowser-{version}-linux-arm64.tar.gz'],
  ['linux', 'amd64', 'installer', 'trace-browser_{version}_amd64.deb'],
  ['linux', 'arm64', 'installer', 'trace-browser_{version}_arm64.deb'],
]

function queryFor([platform, arch, kind]) {
  return `platform=${platform}&arch=${arch}&package=${kind}`
}

function onlyTarget(release, [platform, arch, , template]) {
  const version = release.tag_name.slice(1)
  const names = [template.replace('{version}', version), `TraceBrowser-${version}-${platform}-${arch}.sha256.txt`]
  return { ...release, assets: release.assets.filter(asset => names.includes(asset.name)) }
}

test('every platform and architecture can update as soon as its own package uploads', async t => {
  for (const target of targets) {
    await t.test(queryFor(target), async t => {
      const latest = onlyTarget(releaseFixture('2.1.60', 60), target)
      const calls = githubFixture(t, { latest })
      const response = await request(`/api/trace-browser/latest?${queryFor(target)}`)
      assert.equal(response.status, 200)
      const payload = await response.json()
      assert.equal(payload.version, '2.1.60')
      assert.equal(payload.ready, true)
      assert.equal(payload.assets.length, 2)
      assert.ok(payload.assets.some(asset => asset.name === target[3].replace('{version}', '2.1.60')))
      assert.ok(payload.assets.every(asset => asset.downloadUrl.includes('/release/60/asset/')))
      assert.equal(calls.length, 1)
    })
  }
})

test('only the unfinished target falls back; completed targets use the new version', async t => {
  const latest = onlyTarget(releaseFixture('2.1.60', 60), targets[0])
  const previous = releaseFixture('2.1.59', 59)
  githubFixture(t, { latest, releases: [latest, previous] })
  for (const target of [targets[0], targets[1], targets[4], targets[7]]) {
    const response = await request(`/api/trace-browser/latest?${queryFor(target)}`)
    assert.equal(response.status, 200)
    assert.equal((await response.json()).version, target === targets[0] ? '2.1.60' : '2.1.59')
  }
})

test('installer availability does not advertise an unfinished self-update package', async t => {
  const latest = onlyTarget(releaseFixture('2.1.60', 60), targets[0])
  const previous = releaseFixture('2.1.59', 59)
  githubFixture(t, { latest, releases: [latest, previous] })
  const response = await request(`/api/trace-browser/latest?${queryFor(targets[2])}`)
  assert.equal((await response.json()).version, '2.1.59')
})

test('no published package for the requested target is a normal no-update response', async t => {
  const latest = onlyTarget(releaseFixture('2.1.60', 60), targets[0])
  githubFixture(t, { latest, releases: [latest] })
  const response = await request(`/api/trace-browser/latest?${queryFor(targets[1])}`)
  assert.equal(response.status, 200)
  const payload = await response.json()
  assert.equal(payload.ok, true)
  assert.equal(payload.ready, false)
  assert.deepEqual(payload.assets, [])
  assert.equal(payload.version, undefined)
})

test('only the requested package and its checksum must finish uploading', async t => {
  const latest = onlyTarget(releaseFixture('2.1.60', 60), targets[0])
  const previous = releaseFixture('2.1.59', 59)
  latest.assets[1].state = 'starter'
  githubFixture(t, { latest, releases: [latest, previous] })
  const path = `/api/trace-browser/latest?${queryFor(targets[0])}`
  const before = await request(path)
  assert.equal((await before.json()).version, '2.1.59')
  latest.assets[1].state = 'uploaded'
  const after = await request(path, { 'If-None-Match': before.headers.get('ETag') })
  assert.equal(after.status, 200)
  assert.equal((await after.json()).version, '2.1.60')
})

test('wrong-version, zero-byte, missing checksum and unfinished assets stay hidden', async t => {
  for (const [name, change] of [
    ['wrong version', release => { release.assets[0].name = release.assets[0].name.replace('2.1.60', '2.1.59') }],
    ['zero bytes', release => { release.assets[0].size = 0 }],
    ['uploading', release => { release.assets[0].state = 'starter' }],
    ['missing state', release => { delete release.assets[0].state }],
    ['missing download URL', release => { delete release.assets[0].browser_download_url }],
    ['checksum missing', release => { release.assets.pop() }],
    ['draft', release => { release.draft = true }],
    ['prerelease', release => { release.prerelease = true }],
  ]) {
    await t.test(name, async t => {
      const latest = onlyTarget(releaseFixture('2.1.60', 60), targets[0])
      change(latest)
      const previous = releaseFixture('2.1.59', 59)
      githubFixture(t, { latest, releases: [latest, previous] })
      const response = await request(`/api/trace-browser/latest?${queryFor(targets[0])}`)
      assert.equal((await response.json()).version, '2.1.59')
    })
  }
})

test('legacy unscoped clients see a partially published release with available packages', async t => {
  const latest = onlyTarget(releaseFixture('2.1.60', 60), targets[0])
  githubFixture(t, { latest })
  const response = await request('/api/trace-browser/latest')
  assert.equal(response.status, 200)
  const payload = await response.json()
  assert.equal(payload.version, '2.1.60')
  assert.equal(payload.assets.length, 2)
})

test('ETag changes when another architecture uploads under the same tag', async t => {
  const full = releaseFixture('2.1.60', 60)
  const latest = onlyTarget(full, targets[0])
  githubFixture(t, { latest })
  const before = await request('/api/trace-browser/latest')
  const etag = before.headers.get('ETag')
  assert.equal((await request('/api/trace-browser/latest', { 'If-None-Match': `W/${etag}` })).status, 304)
  latest.assets.push(...onlyTarget(full, targets[1]).assets)
  const after = await request('/api/trace-browser/latest', { 'If-None-Match': etag })
  assert.equal(after.status, 200)
  assert.equal((await after.json()).assets.length, 4)
})

test('different platform queries cannot share a not-modified response', async t => {
  githubFixture(t, { latest: releaseFixture('2.1.60', 60) })
  const before = await request(`/api/trace-browser/latest?${queryFor(targets[0])}`)
  const after = await request(`/api/trace-browser/latest?${queryFor(targets[1])}`, { 'If-None-Match': before.headers.get('ETag') })
  assert.equal(after.status, 200)
})

test('version lists filter by the requested target before applying the limit', async t => {
  const latest = onlyTarget(releaseFixture('2.1.60', 60), targets[0])
  const previous = releaseFixture('2.1.59', 59)
  githubFixture(t, { latest, releases: [{ ...releaseFixture('2.1.61', 61), draft: true }, latest, previous] })
  for (const target of [targets[0], targets[1]]) {
    const response = await request(`/api/trace-browser/releases?limit=1&${queryFor(target)}`)
    const payload = await response.json()
    assert.equal(payload.count, 1)
    assert.equal(payload.releases[0].version, target === targets[0] ? '2.1.60' : '2.1.59')
  }
})

test('stable downloads independently choose the latest package for each architecture', async t => {
  const latest = onlyTarget(releaseFixture('2.1.60', 60), targets[0])
  const previous = releaseFixture('2.1.59', 59)
  const calls = githubFixture(t, { latest, releases: [latest, previous] })
  for (const [arch, version, suffix] of [['amd64', '2.1.60', 'win-x64'], ['arm64', '2.1.59', 'win-arm64']]) {
    const response = await request(`/download/trace-browser/windows/${arch}/installer`)
    assert.equal(response.status, 200)
    assert.equal(await response.text(), 'package')
    assert.ok(calls.at(-1).url.pathname.includes(`/v${version}/TraceBrowser-Setup-${version}-${suffix}.exe`))
  }
})

test('release-specific assets download without waiting for other platforms', async t => {
  const latest = onlyTarget(releaseFixture('2.1.60', 60), targets[0])
  const previous = releaseFixture('2.1.59', 59)
  const calls = githubFixture(t, { latest, releases: [latest, previous] })
  const response = await request('/download/trace-browser/release/60/asset/TraceBrowser-2.1.60-windows-amd64.sha256.txt')
  assert.equal(response.status, 200)
  assert.ok(calls.at(-1).url.pathname.includes('/v2.1.60/'))
  const missing = await request('/download/trace-browser/release/60/asset/TraceBrowser-Setup-2.1.60-win-arm64.exe')
  assert.equal(missing.status, 404)
})

test('legacy named downloads locate their pinned asset in an older release', async t => {
  const latest = onlyTarget(releaseFixture('2.1.60', 60), targets[0])
  const previous = releaseFixture('2.1.59', 59)
  const calls = githubFixture(t, { latest, releases: [latest, previous] })
  const response = await request('/download/trace-browser/asset/TraceBrowser-Setup-2.1.59-win-arm64.exe')
  assert.equal(response.status, 200)
  assert.ok(calls.at(-1).url.pathname.includes('/v2.1.59/'))
})

test('legacy Windows SHA256SUMS remains supported without other architecture files', async t => {
  const latest = onlyTarget(releaseFixture('2.1.60', 60), targets[0])
  latest.assets[1].name = 'SHA256SUMS'
  githubFixture(t, { latest })
  const response = await request(`/api/trace-browser/latest?${queryFor(targets[0])}`)
  assert.equal((await response.json()).version, '2.1.60')
})

test('Windows self-update can use the full portable archive when that is available', async t => {
  const latest = releaseFixture('2.1.60', 60)
  latest.assets = latest.assets.filter(asset => asset.name === 'TraceBrowser-Portable-2.1.60-win-x64.zip' || asset.name === 'TraceBrowser-2.1.60-windows-amd64.sha256.txt')
  githubFixture(t, { latest })
  const response = await request(`/api/trace-browser/latest?${queryFor(targets[2])}`)
  const payload = await response.json()
  assert.equal(payload.version, '2.1.60')
  assert.ok(payload.assets.some(asset => asset.name === 'TraceBrowser-Portable-2.1.60-win-x64.zip'))
})

test('invalid target queries are rejected instead of selecting another architecture', async t => {
  const calls = githubFixture(t, { latest: releaseFixture('2.1.60', 60) })
  for (const query of ['platform=windows', 'platform=windows&arch=mips', 'platform=linux&arch=amd64&package=selfupdate']) {
    assert.equal((await request(`/api/trace-browser/latest?${query}`)).status, 400)
  }
  assert.equal(calls.length, 0)
})

test('darwin and x64 request aliases resolve the matching macOS build', async t => {
  githubFixture(t, { latest: onlyTarget(releaseFixture('2.1.60', 60), targets[4]) })
  const response = await request('/api/trace-browser/latest?platform=darwin&arch=x64')
  assert.equal((await response.json()).version, '2.1.60')
})

test('target-specific fallback looks through historical pages and remains bounded', async t => {
  const latest = onlyTarget(releaseFixture('2.1.60', 60), targets[0])
  const previous = releaseFixture('2.1.59', 59)
  const state = { latest, pages: [Array(100).fill(latest), [previous]] }
  const calls = githubFixture(t, state)
  const response = await request(`/api/trace-browser/latest?${queryFor(targets[1])}`)
  assert.equal((await response.json()).version, '2.1.59')
  assert.equal(calls.at(-1).url.searchParams.get('page'), '2')
  delete state.pages
  state.releases = Array(100).fill(latest)
  calls.length = 0
  const unavailable = await request(`/api/trace-browser/latest?${queryFor(targets[1])}`)
  assert.equal((await unavailable.json()).ready, false)
  assert.equal(calls.length, 6)
})

test('old-version assets cannot make a different architecture appear ready', async t => {
  const latest = onlyTarget(releaseFixture('2.1.60', 60), targets[0])
  latest.assets.push(...onlyTarget(releaseFixture('2.1.59', 59), targets[1]).assets)
  githubFixture(t, { latest, releases: [latest] })
  const response = await request(`/api/trace-browser/latest?${queryFor(targets[1])}`)
  assert.equal((await response.json()).ready, false)
})

test('GitHub errors remain visible', async t => {
  githubFixture(t, { latest: { message: 'API unavailable' }, latestStatus: 503 })
  assert.equal((await request('/api/trace-browser/latest')).status, 502)
})

test('Chromium keeps its independent download rules', async t => {
  const latest = { id: 99, tag_name: 'v140.0.0', assets: [{ name: 'chromium-windows-amd64.zip', size: 100 }] }
  const calls = githubFixture(t, { latest })
  const response = await request('/api/chromium/latest')
  assert.equal(response.status, 200)
  assert.equal((await response.json()).version, '140.0.0')
  assert.equal(calls.length, 1)
})
