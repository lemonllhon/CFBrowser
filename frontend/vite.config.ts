import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import yaml from 'js-yaml'

const defaultDevPort = 5218
const configDir = path.dirname(fileURLToPath(import.meta.url))

function resolveBoolean(rawValue: string | undefined, fallbackValue: boolean) {
  const raw = String(rawValue ?? '').trim().toLowerCase()
  if (!raw) {
    return fallbackValue
  }
  if (raw === '1' || raw === 'true' || raw === 'yes' || raw === 'on') {
    return true
  }
  if (raw === '0' || raw === 'false' || raw === 'no' || raw === 'off') {
    return false
  }
  return fallbackValue
}

function resolveDevPort() {
  const raw = Number.parseInt(process.env.FRONTEND_PORT || '', 10)
  if (Number.isInteger(raw) && raw > 0 && raw <= 65535) {
    return raw
  }
  return defaultDevPort
}

function normalizeBuildVersion(value: unknown) {
  const version = String(value ?? '').trim()
  if (!version || version.toLowerCase() === 'unknown') {
    return ''
  }
  return version.replace(/^v/i, '')
}

function resolveBuildVersion() {
  const envVersion = normalizeBuildVersion(process.env.TRACE_BROWSER_VERSION || process.env.VERSION)
  if (envVersion) {
    return envVersion
  }

  try {
    const configPath = path.resolve(configDir, '..', 'build', 'config.yml')
    const rawConfig = fs.readFileSync(configPath, 'utf8')
    const config = yaml.load(rawConfig) as { info?: { version?: string } } | null
    return normalizeBuildVersion(config?.info?.version) || 'dev'
  } catch {
    return 'dev'
  }
}

const devPort = resolveDevPort()
const disableHmr = resolveBoolean(process.env.FRONTEND_DISABLE_HMR, false)
const buildVersion = resolveBuildVersion()

export default defineConfig({
  plugins: [react()],
  define: {
    __TRACE_BROWSER_BUILD_VERSION__: JSON.stringify(buildVersion),
  },
  server: {
    port: devPort,
    strictPort: true,
    host: '127.0.0.1',
    cors: true,
    hmr: disableHmr
      ? false
      : {
          host: '127.0.0.1',
          protocol: 'ws',
        },
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'react-vendor': ['react', 'react-dom', 'react-router-dom'],
        },
      },
    },
  },
})

