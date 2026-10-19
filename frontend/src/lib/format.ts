const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' })

const UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
  ['year', 365 * 24 * 3600],
  ['month', 30 * 24 * 3600],
  ['week', 7 * 24 * 3600],
  ['day', 24 * 3600],
  ['hour', 3600],
  ['minute', 60],
]

/** "3 days ago", "just now", … */
export function relativeTime(iso: string, now = Date.now()): string {
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return '—'
  const diff = (t - now) / 1000
  for (const [unit, secs] of UNITS) {
    if (Math.abs(diff) >= secs) return rtf.format(Math.round(diff / secs), unit)
  }
  return 'just now'
}

/** Locale date + time, e.g. "Oct 19, 2026, 5:36 PM". */
export function formatDateTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

export function formatDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleDateString(undefined, { dateStyle: 'medium' })
}

/** Compact duration between two timestamps: "3d", "5h", "2w". */
export function durationBetween(fromIso: string, toIso: string): string {
  const secs = Math.max(0, (Date.parse(toIso) - Date.parse(fromIso)) / 1000)
  if (secs < 3600) return `${Math.max(1, Math.round(secs / 60))}m`
  if (secs < 86400) return `${Math.round(secs / 3600)}h`
  if (secs < 14 * 86400) return `${Math.round(secs / 86400)}d`
  if (secs < 60 * 86400) return `${Math.round(secs / (7 * 86400))}w`
  return `${Math.round(secs / (30 * 86400))}mo`
}

/** Value for <input type="datetime-local"> in the user's local timezone. */
export function toLocalInputValue(d = new Date()): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function percent(part: number, whole: number): string {
  if (!whole) return '0%'
  const p = (part / whole) * 100
  return `${p < 10 && p > 0 ? p.toFixed(1) : Math.round(p)}%`
}
