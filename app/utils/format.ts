export function formatBytes(bytes?: number | null): string {
  if (bytes === null || bytes === undefined || Number.isNaN(bytes)) return '0 B'
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const i = Math.min(Math.floor(Math.log(Math.abs(bytes)) / Math.log(k)), sizes.length - 1)
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

export function formatNumber(value?: number | null): string {
  if (value === null || value === undefined || Number.isNaN(value)) return '0'
  return new Intl.NumberFormat().format(value)
}

// normalizePercent treats negative sentinels (-1), NaN/Infinity, null and
// undefined as unavailable, while preserving valid 0/0.0 idle readings.
export function normalizePercent(value?: number | null): number | null {
  if (value === null || value === undefined || !Number.isFinite(value) || value < 0) return null
  return value
}

// Displays 1 decimal place for fractional floats (12.34 -> "12.3%") and whole
// integers without trailing ".0" (42 -> "42%", 0 / 0.0 -> "0%").
export function formatPercent(value?: number | null): string {
  const normalized = normalizePercent(value)
  if (normalized === null) return 'N/A'
  const rounded = Math.round(normalized * 10) / 10
  return Number.isInteger(rounded) ? `${rounded}%` : `${rounded.toFixed(1)}%`
}

export function formatDuration(millis?: number | null): string {
  if (!millis || millis <= 0) return '0s'
  const seconds = Math.floor(millis / 1000)
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = seconds % 60
  const parts: string[] = []
  if (days) parts.push(`${days}d`)
  if (hours) parts.push(`${hours}h`)
  if (minutes) parts.push(`${minutes}m`)
  if (secs || parts.length === 0) parts.push(`${secs}s`)
  return parts.join(' ')
}

export function formatTimestamp(millis?: number | null): string {
  if (!millis || millis <= 0) return '-'
  return new Date(millis).toLocaleString()
}

export function extractDetailedError(body: any): string {
  if (!body) return ''
  if (typeof body === 'string') return body
  if (typeof body.error === 'string') return body.error
  if (body.error && typeof body.error === 'object') {
    const parts: string[] = []
    if (typeof body.error.reason === 'string') parts.push(body.error.reason)
    if (body.error.caused_by && typeof body.error.caused_by.reason === 'string') {
      parts.push(body.error.caused_by.reason)
    }
    if (Array.isArray(body.error.root_cause) && body.error.root_cause[0]?.reason) {
      const root = body.error.root_cause[0].reason
      if (!parts.includes(root)) parts.push(root)
    }
    if (parts.length) return parts.join(' — ')
    if (typeof body.error.type === 'string') return body.error.type
  }
  if (Array.isArray(body.failed_indices) && body.failed_indices.length) {
    const reasons = body.failed_indices
      .map((f: any) => f?.reason || f?.cause)
      .filter(Boolean)
    if (reasons.length) return reasons.join('; ')
  }
  if (typeof body.message === 'string') return body.message
  return ''
}
