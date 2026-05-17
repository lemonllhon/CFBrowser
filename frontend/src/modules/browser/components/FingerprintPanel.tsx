import { useEffect, useState } from 'react'
import { ChevronDown, ChevronUp, RefreshCw, Wand2 } from 'lucide-react'
import { Alert, ConfirmModal, FormItem, Input, Select, Textarea } from '../../../shared/components'
import {
  type FingerprintConfig,
  FINGERPRINT_PRESETS,
  PRESET_RESOLUTIONS,
  deserialize,
  getSystemTimezone,
  randomFingerprintSeed,
  serialize,
} from '../utils/fingerprintSerializer'
import { REGION_PRESETS, findRegionPresetByLocale, regionTimezones } from '../config/regionPresets'

interface FingerprintPanelProps {
  value: string[]
  onChange: (args: string[]) => void
}

const BRAND_OPTIONS = [
  { value: '', label: '自动随机' },
  { value: 'Chrome', label: 'Chrome' },
  { value: 'Edge', label: 'Edge' },
  { value: 'Firefox', label: 'Firefox' },
  { value: 'Safari', label: 'Safari' },
]

const PLATFORM_OPTIONS = [
  { value: '', label: '自动随机' },
  { value: 'windows', label: 'Windows' },
  { value: 'mac', label: 'macOS' },
  { value: 'linux', label: 'Linux' },
]

const LANG_OPTIONS = [
  { value: '', label: '自动/由地区国家联动' },
  { value: 'zh-CN', label: '中文 (zh-CN)' },
  { value: 'en-US', label: 'English (en-US)' },
  { value: 'en-GB', label: 'English (en-GB)' },
  { value: 'ja-JP', label: '日本語 (ja-JP)' },
  { value: 'ko-KR', label: '한국어 (ko-KR)' },
  { value: 'fr-FR', label: 'Français (fr-FR)' },
  { value: 'de-DE', label: 'Deutsch (de-DE)' },
]

const TIMEZONE_OPTIONS = [
  { value: '', label: '自动/由地区国家联动' },
  { value: 'system', label: '跟随系统时区' },
  // 亚洲
  { value: 'Asia/Shanghai', label: 'Asia/Shanghai (UTC+8)' },
  { value: 'Asia/Tokyo', label: 'Asia/Tokyo (UTC+9)' },
  { value: 'Asia/Seoul', label: 'Asia/Seoul (UTC+9)' },
  { value: 'Asia/Singapore', label: 'Asia/Singapore (UTC+8)' },
  { value: 'Asia/Hong_Kong', label: 'Asia/Hong_Kong (UTC+8)' },
  { value: 'Asia/Dubai', label: 'Asia/Dubai (UTC+4)' },
  { value: 'Asia/Kolkata', label: 'Asia/Kolkata (UTC+5:30)' },
  // 美洲
  { value: 'America/New_York', label: 'America/New_York (UTC-5)' },
  { value: 'America/Los_Angeles', label: 'America/Los_Angeles (UTC-8)' },
  { value: 'America/Chicago', label: 'America/Chicago (UTC-6)' },
  { value: 'America/Denver', label: 'America/Denver (UTC-7)' },
  { value: 'America/Toronto', label: 'America/Toronto (UTC-5)' },
  { value: 'America/Sao_Paulo', label: 'America/Sao_Paulo (UTC-3)' },
  // 欧洲
  { value: 'Europe/London', label: 'Europe/London (UTC+0)' },
  { value: 'Europe/Paris', label: 'Europe/Paris (UTC+1)' },
  { value: 'Europe/Berlin', label: 'Europe/Berlin (UTC+1)' },
  { value: 'Europe/Moscow', label: 'Europe/Moscow (UTC+3)' },
  // 大洋洲
  { value: 'Australia/Sydney', label: 'Australia/Sydney (UTC+10)' },
  { value: 'Pacific/Auckland', label: 'Pacific/Auckland (UTC+12)' },
]

const REGION_LANG_OPTIONS = Array.from(new Set(REGION_PRESETS.map(item => item.lang)))
  .filter(lang => !LANG_OPTIONS.some(opt => opt.value === lang))
  .sort((a, b) => a.localeCompare(b))
  .map(lang => ({ value: lang, label: lang }))

const REGION_TIMEZONE_OPTIONS = Array.from(new Set(REGION_PRESETS.flatMap(item => regionTimezones(item))))
  .filter(timezone => !TIMEZONE_OPTIONS.some(opt => opt.value === timezone))
  .sort((a, b) => a.localeCompare(b))
  .map(timezone => ({ value: timezone, label: timezone }))

const ALL_LANG_OPTIONS = [...LANG_OPTIONS, ...REGION_LANG_OPTIONS]
const ALL_TIMEZONE_OPTIONS = [...TIMEZONE_OPTIONS, ...REGION_TIMEZONE_OPTIONS]

const RESOLUTION_OPTIONS = [
  { value: '', label: '自动随机' },
  ...PRESET_RESOLUTIONS.map(r => ({ value: r, label: r })),
  { value: 'custom', label: '自定义...' },
]

const WEBGL_VENDOR_OPTIONS = [
  { value: '', label: '自动随机' },
  { value: 'Intel', label: 'Intel' },
  { value: 'NVIDIA', label: 'NVIDIA' },
  { value: 'AMD', label: 'AMD' },
  { value: 'Apple', label: 'Apple' },
]

const WEBGL_RENDERER_OPTIONS: Record<string, { value: string; label: string }[]> = {
  Intel: [
    { value: '', label: '跟随供应商随机' },
    { value: 'Intel(R) UHD Graphics 630', label: 'UHD Graphics 630' },
    { value: 'Intel(R) UHD Graphics 620', label: 'UHD Graphics 620' },
    { value: 'Intel(R) HD Graphics 520', label: 'HD Graphics 520' },
    { value: 'Intel(R) Iris(R) Xe Graphics', label: 'Iris Xe Graphics' },
    { value: 'custom', label: '自定义...' },
  ],
  NVIDIA: [
    { value: '', label: '跟随供应商随机' },
    { value: 'NVIDIA GeForce RTX 3080', label: 'GeForce RTX 3080' },
    { value: 'NVIDIA GeForce RTX 3060', label: 'GeForce RTX 3060' },
    { value: 'NVIDIA GeForce GTX 1660', label: 'GeForce GTX 1660' },
    { value: 'NVIDIA GeForce GTX 1080 Ti', label: 'GeForce GTX 1080 Ti' },
    { value: 'custom', label: '自定义...' },
  ],
  AMD: [
    { value: '', label: '跟随供应商随机' },
    { value: 'AMD Radeon RX 6600', label: 'Radeon RX 6600' },
    { value: 'AMD Radeon RX 580', label: 'Radeon RX 580' },
    { value: 'AMD Radeon Vega 8', label: 'Radeon Vega 8' },
    { value: 'custom', label: '自定义...' },
  ],
  Apple: [
    { value: '', label: '跟随供应商随机' },
    { value: 'Apple M1', label: 'Apple M1' },
    { value: 'Apple M2', label: 'Apple M2' },
    { value: 'Apple M3', label: 'Apple M3' },
    { value: 'custom', label: '自定义...' },
  ],
}

const BOOL_OPTIONS = [
  { value: '', label: '自动随机' },
  { value: 'true', label: '启用' },
  { value: 'false', label: '禁用' },
]

const HARDWARE_CONCURRENCY_OPTIONS = [
  { value: '', label: '自动随机' },
  { value: '2', label: '2 核' },
  { value: '4', label: '4 核' },
  { value: '6', label: '6 核' },
  { value: '8', label: '8 核' },
  { value: '10', label: '10 核' },
  { value: '12', label: '12 核' },
  { value: '16', label: '16 核' },
]

const DEVICE_MEMORY_OPTIONS = [
  { value: '', label: '自动随机' },
  { value: '2', label: '2 GB' },
  { value: '4', label: '4 GB' },
  { value: '8', label: '8 GB' },
  { value: '16', label: '16 GB' },
  { value: '32', label: '32 GB' },
]

const COLOR_DEPTH_OPTIONS = [
  { value: '', label: '自动随机' },
  { value: '24', label: '24 位（标准）' },
  { value: '30', label: '30 位（HDR）' },
  { value: '32', label: '32 位' },
]

const WEBRTC_OPTIONS = [
  { value: '', label: '自动防泄漏' },
  { value: 'disable_non_proxied_udp', label: '禁用非代理 UDP（推荐）' },
  { value: 'default_public_interface_only', label: '仅公网接口' },
  { value: 'default_public_and_private_interfaces', label: '公网+私网接口' },
]

const TOUCH_POINTS_OPTIONS = [
  { value: '', label: '自动随机' },
  { value: '0', label: '0（无触摸）' },
  { value: '1', label: '1 点触摸' },
  { value: '5', label: '5 点触摸' },
  { value: '10', label: '10 点触摸' },
]

const AUTO_RENDERER_OPTIONS = [{ value: '', label: '跟随供应商随机' }, { value: 'custom', label: '自定义...' }]

const pick = <T,>(items: T[]): T => items[Math.floor(Math.random() * items.length)]
const randomRendererForVendor = (vendor: string): string => {
  const candidates = (WEBGL_RENDERER_OPTIONS[vendor] || WEBGL_RENDERER_OPTIONS.Intel).filter(item => item.value && item.value !== 'custom')
  return pick(candidates).value
}

const COMMON_FONTS: Record<string, string[]> = {
  windows: [
    'Arial,Segoe UI,Calibri,Microsoft YaHei,SimSun,Times New Roman,Courier New',
    'Arial,Helvetica,Verdana,Tahoma,Times New Roman,Courier New,Georgia',
    'Arial,Segoe UI,Calibri,Verdana,Microsoft YaHei,Times New Roman',
  ],
  mac: [
    'Arial,Helvetica,PingFang SC,Hiragino Sans GB,STHeiti,Times New Roman',
    'Arial,Helvetica,San Francisco,Menlo,Georgia,Times New Roman',
    'Arial,Helvetica,Hiragino Kaku Gothic ProN,Yu Gothic,Times New Roman',
  ],
  linux: [
    'Arial,Noto Sans,Ubuntu,DejaVu Sans,Liberation Sans,Times New Roman',
    'Arial,Noto Sans,Roboto,DejaVu Serif,Liberation Mono,Times New Roman',
  ],
}

function commonFontsForLocale(platform: string, lang?: string): string[] {
  const normalizedPlatform = COMMON_FONTS[platform] ? platform : 'windows'
  const normalizedLang = (lang || '').toLowerCase()

  if (normalizedLang.startsWith('zh')) {
    if (normalizedPlatform === 'mac') {
      return [
        'Arial,Helvetica,PingFang SC,Hiragino Sans GB,STHeiti,Songti SC,Times New Roman',
        'Arial,Helvetica,PingFang SC,Heiti SC,Kaiti SC,Times New Roman',
      ]
    }
    if (normalizedPlatform === 'linux') {
      return [
        'Arial,Noto Sans CJK SC,WenQuanYi Micro Hei,Noto Sans,DejaVu Sans,Times New Roman',
        'Arial,Noto Serif CJK SC,Noto Sans CJK SC,Liberation Sans,Times New Roman',
      ]
    }
    return [
      'Arial,Segoe UI,Microsoft YaHei,SimSun,SimHei,Calibri,Times New Roman',
      'Arial,Microsoft YaHei UI,Microsoft YaHei,SimSun,FangSong,Times New Roman',
    ]
  }

  if (normalizedLang.startsWith('ja')) {
    if (normalizedPlatform === 'mac') {
      return [
        'Arial,Helvetica,Hiragino Kaku Gothic ProN,Yu Gothic,Hiragino Mincho ProN,Times New Roman',
        'Arial,Helvetica,Yu Gothic,Hiragino Sans,Osaka,Times New Roman',
      ]
    }
    if (normalizedPlatform === 'linux') {
      return [
        'Arial,Noto Sans CJK JP,Noto Serif CJK JP,Noto Sans,DejaVu Sans,Times New Roman',
        'Arial,Noto Sans JP,Noto Serif JP,Liberation Sans,Times New Roman',
      ]
    }
    return [
      'Arial,Segoe UI,Yu Gothic,Meiryo,MS Gothic,Times New Roman',
      'Arial,Yu Gothic UI,Meiryo,MS PGothic,Times New Roman',
    ]
  }

  if (normalizedLang.startsWith('ko')) {
    if (normalizedPlatform === 'mac') {
      return [
        'Arial,Helvetica,Apple SD Gothic Neo,Arial Unicode MS,Times New Roman',
        'Arial,Helvetica,AppleGothic,Apple SD Gothic Neo,Times New Roman',
      ]
    }
    if (normalizedPlatform === 'linux') {
      return [
        'Arial,Noto Sans CJK KR,Noto Serif CJK KR,Noto Sans,DejaVu Sans,Times New Roman',
        'Arial,Noto Sans KR,Noto Serif KR,Liberation Sans,Times New Roman',
      ]
    }
    return [
      'Arial,Segoe UI,Malgun Gothic,Gulim,Dotum,Times New Roman',
      'Arial,Malgun Gothic,Microsoft JhengHei,Times New Roman',
    ]
  }

  if (normalizedLang.startsWith('ar')) {
    return [
      'Arial,Segoe UI,Tahoma,Arial Unicode MS,Times New Roman',
      'Arial,Tahoma,Noto Naskh Arabic,Noto Sans Arabic,Times New Roman',
    ]
  }

  return COMMON_FONTS[normalizedPlatform]
}

function rendererMatchesVendor(vendor: string, renderer: string): boolean {
  const normalizedVendor = vendor.toLowerCase()
  const normalizedRenderer = renderer.toLowerCase()
  if (normalizedVendor === 'intel') return normalizedRenderer.includes('intel')
  if (normalizedVendor === 'nvidia') return normalizedRenderer.includes('nvidia') || normalizedRenderer.includes('geforce')
  if (normalizedVendor === 'amd') return normalizedRenderer.includes('amd') || normalizedRenderer.includes('radeon')
  if (normalizedVendor === 'apple') return normalizedRenderer.includes('apple')
  return true
}

function hasFontToken(fonts: string | undefined, tokens: string[]): boolean {
  const normalized = (fonts || '').toLowerCase()
  return tokens.some(token => normalized.includes(token.toLowerCase()))
}

function buildFingerprintConsistencyWarnings(config: FingerprintConfig): string[] {
  const warnings: string[] = []
  const normalizedLang = (config.lang || '').toLowerCase()

  if (config.brand === 'Safari' && config.platform && config.platform !== 'mac') {
    warnings.push('Safari 与非 macOS 平台组合不自然，建议切换为 macOS 或改用 Chrome/Edge。')
  }
  if (config.platform && config.platform !== 'mac' && config.webglVendor === 'Apple') {
    warnings.push('Apple WebGL 供应商不适合 Windows/Linux 画像，建议换成 Intel/NVIDIA/AMD。')
  }
  if (config.webglVendor && config.webglRenderer && !rendererMatchesVendor(config.webglVendor, config.webglRenderer)) {
    warnings.push('WebGL 供应商与渲染器名称不匹配，建议重新选择渲染器。')
  }
  if (config.platform === 'mac' && config.touchPoints && config.touchPoints !== '0') {
    warnings.push('桌面 macOS 通常没有触摸点，建议触摸点数设为 0。')
  }
  if (config.lang && config.timezone && !findRegionPresetByLocale(config.lang, config.timezone)) {
    warnings.push(`语言 ${config.lang} 与时区 ${config.timezone} 未命中地区预设，建议用“地区国家”重新联动。`)
  }
  if (normalizedLang.startsWith('zh') && config.fonts && !hasFontToken(config.fonts, ['Microsoft YaHei', 'SimSun', 'PingFang', 'Noto Sans CJK SC', 'WenQuanYi'])) {
    warnings.push('中文语言画像缺少常见中文字体，建议重新随机设备或补充中文字体。')
  }
  if (normalizedLang.startsWith('ja') && config.fonts && !hasFontToken(config.fonts, ['Yu Gothic', 'Meiryo', 'Hiragino', 'Noto Sans CJK JP', 'Noto Sans JP'])) {
    warnings.push('日语语言画像缺少常见日文字体，建议重新随机设备或补充日文字体。')
  }
  if (normalizedLang.startsWith('ko') && config.fonts && !hasFontToken(config.fonts, ['Malgun Gothic', 'Apple SD Gothic', 'Noto Sans CJK KR', 'Noto Sans KR'])) {
    warnings.push('韩语语言画像缺少常见韩文字体，建议重新随机设备或补充韩文字体。')
  }

  return warnings
}

const AUTO_HARDWARE_CONFIG: Partial<FingerprintConfig> = {
  autoHardware: true,
  seed: undefined,
  brand: undefined,
  platform: undefined,
  resolution: undefined,
  customResolution: undefined,
  colorDepth: undefined,
  hardwareConcurrency: undefined,
  deviceMemory: undefined,
  touchPoints: undefined,
  webglVendor: undefined,
  webglRenderer: undefined,
  canvasNoise: undefined,
  audioNoise: undefined,
  webrtcPolicy: undefined,
  doNotTrack: undefined,
  mediaDevices: undefined,
  fonts: undefined,
}

function randomHardwareFingerprint(base: FingerprintConfig): FingerprintConfig {
  const platform = pick(['windows', 'windows', 'windows', 'mac', 'linux'])
  const brand = platform === 'mac' ? pick(['Chrome', 'Safari']) : pick(['Chrome', 'Chrome', 'Edge'])
  const vendor = platform === 'mac' ? 'Apple' : pick(platform === 'linux' ? ['Intel', 'AMD'] : ['Intel', 'Intel', 'NVIDIA', 'AMD'])
  const renderer = randomRendererForVendor(vendor)
  const highEnd = ['NVIDIA', 'AMD', 'Apple'].includes(vendor) && Math.random() > 0.35
  const resolution = pick(highEnd ? ['1920,1080', '2560,1440', '1600,900', '1440,900'] : ['1920,1080', '1366,768', '1440,900', '1600,900', '1280,800'])
  const hardwareConcurrency = pick(highEnd ? ['8', '10', '12', '16'] : ['4', '6', '8'])
  const deviceMemory = pick(highEnd ? ['8', '16', '32'] : ['4', '8', '16'])
  const touchPoints = platform === 'windows' && Math.random() < 0.12 ? pick(['1', '5']) : '0'

  return {
    ...base,
    seed: randomFingerprintSeed(),
    brand,
    platform,
    resolution,
    customResolution: undefined,
    colorDepth: pick(platform === 'mac' ? ['24', '30'] : ['24', '24', '32']),
    hardwareConcurrency,
    deviceMemory,
    touchPoints,
    webglVendor: vendor,
    webglRenderer: renderer,
    canvasNoise: true,
    audioNoise: true,
    webrtcPolicy: 'disable_non_proxied_udp',
    doNotTrack: false,
    mediaDevices: pick(['1,1,1', '2,1,1', '0,1,1']),
    fonts: pick(commonFontsForLocale(platform, base.lang)),
  }
}

function withAutoFingerprintDefaults(base: FingerprintConfig): FingerprintConfig {
  const hasHardwareConfig = !!base.seed ||
    !!base.brand ||
    !!base.platform ||
    !!base.resolution ||
    !!base.colorDepth ||
    !!base.hardwareConcurrency ||
    !!base.deviceMemory ||
    !!base.webglVendor ||
    !!base.webglRenderer ||
    base.canvasNoise !== undefined ||
    base.audioNoise !== undefined ||
    !!base.webrtcPolicy ||
    base.doNotTrack !== undefined ||
    !!base.touchPoints ||
    !!base.mediaDevices ||
    !!base.fonts

  return { ...base, autoHardware: base.autoHardware ?? !hasHardwareConfig }
}

const PRESET_OPTIONS = [
  { value: '', label: '选择预设...' },
  { value: '__auto_hardware__', label: '自动随机硬件画像（每次启动变化）' },
  ...FINGERPRINT_PRESETS.map(p => ({ value: p.id, label: p.name })),
]

export function FingerprintPanel({ value, onChange }: FingerprintPanelProps) {
  const [config, setConfig] = useState<FingerprintConfig>(() => withAutoFingerprintDefaults(deserialize(value)))
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [, setCustomRenderer] = useState('')
  const [confirmSeedOpen, setConfirmSeedOpen] = useState(false)

  useEffect(() => {
    const parsed = deserialize(value)
    const next = withAutoFingerprintDefaults(parsed)
    setConfig(next)
    const nextArgs = serialize(next)
    if (nextArgs.join('\n') !== value.join('\n')) {
      onChange(nextArgs)
    }
  }, [value.join('\n')])

  const update = (patch: Partial<FingerprintConfig>) => {
    const next = withAutoFingerprintDefaults({ ...config, ...patch })
    setConfig(next)
    onChange(serialize(next))
  }

  const handlePresetChange = (presetId: string) => {
    if (!presetId) return
    if (presetId === '__auto_hardware__') {
      const next: FingerprintConfig = {
        ...config,
        ...AUTO_HARDWARE_CONFIG,
        lang: config.lang,
        timezone: config.timezone,
        unknownArgs: config.unknownArgs,
      }
      setConfig(next)
      onChange(serialize(next))
      return
    }
    const preset = FINGERPRINT_PRESETS.find(p => p.id === presetId)
    if (!preset) return
    // 应用预设时自动生成新种子，保留未知参数
    const next: FingerprintConfig = {
      ...preset.config,
      seed: randomFingerprintSeed(),
      autoHardware: false,
      unknownArgs: config.unknownArgs,
    }
    setConfig(next)
    onChange(serialize(next))
  }

  const handleAdvancedChange = (text: string) => {
    const args = text.split('\n').map(s => s.trim()).filter(Boolean)
    const parsed = withAutoFingerprintDefaults(deserialize(args))
    setConfig(parsed)
    onChange(serialize(parsed))
  }

  const handleRandomHardware = () => {
    const next = {
      ...randomHardwareFingerprint(config),
      autoHardware: false,
      lang: config.lang,
      timezone: config.timezone,
      unknownArgs: config.unknownArgs,
    }
    setConfig(next)
    onChange(serialize(next))
  }

  const handleAutoHardware = () => {
    const next: FingerprintConfig = {
      ...config,
      ...AUTO_HARDWARE_CONFIG,
      lang: config.lang,
      timezone: config.timezone,
      unknownArgs: config.unknownArgs,
    }
    setConfig(next)
    onChange(serialize(next))
  }

  const rendererOptions = config.webglVendor
    ? (WEBGL_RENDERER_OPTIONS[config.webglVendor] ?? AUTO_RENDERER_OPTIONS)
    : AUTO_RENDERER_OPTIONS
  const consistencyWarnings = buildFingerprintConsistencyWarnings(config)

  const isCustomRenderer = config.webglRenderer
    ? !rendererOptions.some(o => o.value === config.webglRenderer && o.value !== 'custom')
    : false

  const advancedText = serialize(config).join('\n')

  return (
    <div className="space-y-4">
      {/* 指纹种子 */}
      <div className="p-3 rounded-lg bg-[var(--color-bg-hover)] border border-[var(--color-border)] space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wide">指纹种子（Fingerprint Seed）</span>
          <span className="text-xs text-[var(--color-text-muted)]">决定所有随机噪声的根值，不同种子 = 不同指纹</span>
        </div>
        <div className="flex items-center gap-2">
          <Input
            value={config.seed ?? ''}
            onChange={e => update({ seed: e.target.value || undefined })}
            placeholder="留空则由系统按 ProfileId 自动生成"
            className="flex-1 font-mono text-sm"
          />
          <button
            type="button"
            title="随机生成新种子"
            onClick={() => {
              if (config.seed) {
                setConfirmSeedOpen(true)
              } else {
                update({ seed: randomFingerprintSeed() })
              }
            }}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs bg-[var(--color-primary)] text-white hover:opacity-90 transition-opacity shrink-0"
          >
            <RefreshCw className="w-3.5 h-3.5" />
            随机
          </button>
        </div>
      </div>

      <ConfirmModal
        open={confirmSeedOpen}
        onClose={() => setConfirmSeedOpen(false)}
        onConfirm={() => update({ seed: randomFingerprintSeed() })}
        title="重新生成指纹种子"
        content="重新生成后，当前指纹将完全改变，浏览器的 Canvas、WebGL、Audio 等所有噪声特征都会随之变化。确定继续？"
        confirmText="确定重新生成"
        danger
      />

      {/* 预设选择 */}
      <div className="flex items-center gap-3 p-3 rounded-lg bg-[var(--color-bg-hover)] border border-[var(--color-border)]">
        <Wand2 className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
        <div className="flex-1 min-w-0">
          <Select
            value=""
            onChange={e => handlePresetChange(e.target.value)}
            options={PRESET_OPTIONS}
          />
        </div>
        <span className="text-xs text-[var(--color-text-muted)] shrink-0">选择后覆盖当前配置</span>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3 p-3 rounded-lg bg-[var(--color-bg-hover)] border border-[var(--color-border)]">
        <div>
          <p className="text-sm font-medium text-[var(--color-text-primary)]">硬件画像一键随机</p>
          <p className="text-xs text-[var(--color-text-muted)] mt-1">一键随机会立即生成并保存一套设备识别；启用自动随机后，每次浏览器重新启动都会换一套设备识别。语言和时区不参与硬件随机。</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={handleRandomHardware}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs bg-[var(--color-primary)] text-white hover:opacity-90 transition-opacity shrink-0"
          >
            <Wand2 className="w-3.5 h-3.5" />
            一键随机
          </button>
          <button
            type="button"
            onClick={handleAutoHardware}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs transition-opacity shrink-0 ${config.autoHardware ? 'bg-emerald-600 text-white' : 'bg-[var(--color-bg-secondary)] text-[var(--color-text-secondary)] border border-[var(--color-border)] hover:bg-[var(--color-bg-hover)]'}`}
          >
            <RefreshCw className="w-3.5 h-3.5" />
            {config.autoHardware ? '已启用自动随机' : '启用自动随机'}
          </button>
        </div>
      </div>

      {config.autoHardware && (
        <div className="rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2 text-xs text-emerald-800">
          当前使用自动随机硬件画像预设：保存后，浏览器每次重新启动都会重新生成指纹种子、浏览器品牌、系统、屏幕、CPU、内存、WebGL、噪声、WebRTC、媒体设备和字体。语言与时区保持当前设置。
        </div>
      )}

      {consistencyWarnings.length > 0 && (
        <Alert
          type="warning"
          title="指纹一致性提示"
          message={
            <ul className="list-disc pl-4 space-y-1">
              {consistencyWarnings.map(item => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          }
        />
      )}

      {/* 基础身份 */}
      <div>
        <p className="text-xs font-medium text-[var(--color-text-muted)] mb-2 uppercase tracking-wide">基础身份</p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <FormItem label="浏览器品牌">
            <Select value={config.brand ?? ''} onChange={e => update({ brand: e.target.value || undefined, autoHardware: e.target.value ? false : true })} options={BRAND_OPTIONS} />
          </FormItem>
          <FormItem label="操作系统">
            <Select value={config.platform ?? ''} onChange={e => update({ platform: e.target.value || undefined, autoHardware: e.target.value ? false : true })} options={PLATFORM_OPTIONS} />
          </FormItem>
          <FormItem label="语言">
            <Select value={config.lang ?? ''} onChange={e => update({ lang: e.target.value || undefined })} options={ALL_LANG_OPTIONS} />
          </FormItem>
          <FormItem label="时区">
            <Select value={config.timezone ?? ''} onChange={e => update({ timezone: e.target.value || undefined })} options={ALL_TIMEZONE_OPTIONS.map(opt =>
              opt.value === 'system'
                ? { ...opt, label: `跟随系统时区 (当前: ${getSystemTimezone()})` }
                : opt
            )} />
          </FormItem>
        </div>
      </div>

      {/* 屏幕与硬件 */}
      <div>
        <p className="text-xs font-medium text-[var(--color-text-muted)] mb-2 uppercase tracking-wide">屏幕与硬件</p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <FormItem label="分辨率">
            <Select
              value={config.resolution ?? ''}
              onChange={e => update({ resolution: e.target.value || undefined, customResolution: undefined, autoHardware: e.target.value ? false : true })}
              options={RESOLUTION_OPTIONS}
            />
          </FormItem>
          {config.resolution === 'custom' && (
            <FormItem label="自定义分辨率">
              <Input value={config.customResolution ?? ''} onChange={e => update({ customResolution: e.target.value || undefined })} placeholder="1600,900" />
            </FormItem>
          )}
          <FormItem label="色深">
            <Select value={config.colorDepth ?? ''} onChange={e => update({ colorDepth: e.target.value || undefined, autoHardware: e.target.value ? false : true })} options={COLOR_DEPTH_OPTIONS} />
          </FormItem>
          <FormItem label="CPU 核心数">
            <Select value={config.hardwareConcurrency ?? ''} onChange={e => update({ hardwareConcurrency: e.target.value || undefined, autoHardware: e.target.value ? false : true })} options={HARDWARE_CONCURRENCY_OPTIONS} />
          </FormItem>
          <FormItem label="设备内存">
            <Select value={config.deviceMemory ?? ''} onChange={e => update({ deviceMemory: e.target.value || undefined, autoHardware: e.target.value ? false : true })} options={DEVICE_MEMORY_OPTIONS} />
          </FormItem>
          <FormItem label="触摸点数">
            <Select value={config.touchPoints ?? ''} onChange={e => update({ touchPoints: e.target.value || undefined, autoHardware: e.target.value ? false : true })} options={TOUCH_POINTS_OPTIONS} />
          </FormItem>
        </div>
      </div>

      {/* 渲染指纹 */}
      <div>
        <p className="text-xs font-medium text-[var(--color-text-muted)] mb-2 uppercase tracking-wide">渲染指纹</p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <FormItem label="WebGL 供应商">
            <Select
              value={config.webglVendor ?? ''}
              onChange={e => {
                const vendor = e.target.value || undefined
                const renderer = vendor ? randomRendererForVendor(vendor) : undefined
                update({ webglVendor: vendor, webglRenderer: renderer, autoHardware: vendor ? false : true })
              }}
              options={WEBGL_VENDOR_OPTIONS}
            />
          </FormItem>
          <FormItem label="WebGL 渲染器">
            {isCustomRenderer ? (
              <Input
                value={config.webglRenderer ?? ''}
                onChange={e => update({ webglRenderer: e.target.value || undefined })}
                placeholder="自定义渲染器名称"
              />
            ) : (
              <Select
                value={config.webglRenderer ?? ''}
                onChange={e => {
                  if (e.target.value === 'custom') {
                    setCustomRenderer('')
                    update({ webglRenderer: undefined })
                  } else {
                    update({ webglRenderer: e.target.value || undefined })
                  }
                }}
                options={rendererOptions}
                disabled={!config.webglVendor}
              />
            )}
          </FormItem>
          <FormItem label="Canvas 噪声">
            <Select
              value={config.canvasNoise === undefined ? '' : String(config.canvasNoise)}
              onChange={e => { const v = e.target.value; update({ canvasNoise: v === '' ? undefined : v === 'true', autoHardware: v ? false : true }) }}
              options={BOOL_OPTIONS}
            />
          </FormItem>
          <FormItem label="Audio 噪声">
            <Select
              value={config.audioNoise === undefined ? '' : String(config.audioNoise)}
              onChange={e => { const v = e.target.value; update({ audioNoise: v === '' ? undefined : v === 'true', autoHardware: v ? false : true }) }}
              options={BOOL_OPTIONS}
            />
          </FormItem>
        </div>
      </div>

      {/* 网络与隐私 */}
      <div>
        <p className="text-xs font-medium text-[var(--color-text-muted)] mb-2 uppercase tracking-wide">网络与隐私</p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <FormItem label="WebRTC 策略">
            <Select value={config.webrtcPolicy ?? ''} onChange={e => update({ webrtcPolicy: e.target.value || undefined, autoHardware: e.target.value ? false : true })} options={WEBRTC_OPTIONS} />
          </FormItem>
          <FormItem label="Do Not Track">
            <Select
              value={config.doNotTrack === undefined ? '' : String(config.doNotTrack)}
              onChange={e => { const v = e.target.value; update({ doNotTrack: v === '' ? undefined : v === 'true', autoHardware: v ? false : true }) }}
              options={BOOL_OPTIONS}
            />
          </FormItem>
          <FormItem label="媒体设备 (摄像头,麦克风,扬声器)">
            <Input
              value={config.mediaDevices ?? ''}
              onChange={e => update({ mediaDevices: e.target.value || undefined, autoHardware: e.target.value ? false : true })}
              placeholder="2,1,1"
            />
          </FormItem>
        </div>
      </div>

      {/* 字体 */}
      <div>
        <p className="text-xs font-medium text-[var(--color-text-muted)] mb-2 uppercase tracking-wide">字体</p>
        <FormItem label="字体列表">
          <Input
            value={config.fonts ?? ''}
            onChange={e => update({ fonts: e.target.value || undefined, autoHardware: e.target.value ? false : true })}
            placeholder="Arial,Helvetica,Times New Roman（逗号分隔）"
          />
        </FormItem>
      </div>

      {/* 高级模式 */}
      <div className="border border-[var(--color-border)] rounded-lg overflow-hidden">
        <button
          type="button"
          className="w-full flex items-center justify-between px-4 py-2.5 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-bg-hover)] transition-colors"
          onClick={() => setAdvancedOpen(v => !v)}
        >
          <span>高级模式（原始参数）</span>
          {advancedOpen ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
        </button>
        {advancedOpen && (
          <div className="px-4 pb-4 pt-2 border-t border-[var(--color-border)]">
            <p className="text-xs text-[var(--color-text-muted)] mb-2">每行一个参数，修改后自动同步到上方控件</p>
            <Textarea
              value={advancedText}
              onChange={e => handleAdvancedChange(e.target.value)}
              rows={6}
              placeholder="--fingerprint-brand=Chrome"
            />
          </div>
        )}
      </div>
    </div>
  )
}
