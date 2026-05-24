export function normalizeBuildVersion(value: unknown): string {
  const version = String(value ?? '').trim()
  if (!version || version.toLowerCase() === 'unknown') {
    return ''
  }
  return version.replace(/^v/i, '')
}

export function getBundledAppVersion(): string {
  return normalizeBuildVersion(__TRACE_BROWSER_BUILD_VERSION__) || 'dev'
}
