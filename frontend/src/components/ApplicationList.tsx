import {
  Briefcase,
  CalendarClock,
  ChevronRight,
  DollarSign,
  ExternalLink,
  History,
  Loader2,
  MapPin,
  Pencil,
  Plus,
  Search,
  SearchX,
  Trash2,
  X,
} from 'lucide-react'
import { Fragment, useEffect, useMemo, useRef, useState, type FormEvent, type ReactNode } from 'react'
import { api } from '../api/client'
import { durationBetween, formatDate, formatDateTime, relativeTime, toLocalInputValue } from '../lib/format'
import { STATUSES, STATUS_META, type Application, type ApplicationDetail, type Status } from '../types'
import StatusBadge, { StatusSelect } from './StatusBadge'

export type Notify = (message: string, kind?: 'success' | 'error') => void

interface ApplicationListProps {
  refreshKey: number
  onChanged: () => void
  onEdit: (app: Application) => void
  onCreate: () => void
  notify: Notify
}

type Filter = Status | ''

export default function ApplicationList({ refreshKey, onChanged, onEdit, onCreate, notify }: ApplicationListProps) {
  const [statusFilter, setStatusFilter] = useState<Filter>('')
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [apps, setApps] = useState<Application[]>([])
  const [counts, setCounts] = useState<Partial<Record<Status, number>>>({})
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [expandedId, setExpandedId] = useState<number | null>(null)
  const [busy, setBusy] = useState<Set<number>>(new Set())
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)
  const searchRef = useRef<HTMLInputElement>(null)

  // Debounce text search.
  useEffect(() => {
    const t = setTimeout(() => setDebouncedSearch(search), 220)
    return () => clearTimeout(t)
  }, [search])

  // "/" focuses search.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement)?.tagName
      if (e.key === '/' && tag !== 'INPUT' && tag !== 'TEXTAREA' && tag !== 'SELECT') {
        e.preventDefault()
        searchRef.current?.focus()
      }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [])

  // Fetch the (server-side filtered) list.
  useEffect(() => {
    const ctrl = new AbortController()
    setLoading(true)
    api
      .listApplications({ status: statusFilter, search: debouncedSearch }, ctrl.signal)
      .then((data) => {
        setApps(data)
        setError(null)
      })
      .catch((err: unknown) => {
        if (ctrl.signal.aborted) return
        setError(err instanceof Error ? err.message : 'Failed to load applications')
      })
      .finally(() => {
        if (!ctrl.signal.aborted) setLoading(false)
      })
    return () => ctrl.abort()
  }, [statusFilter, debouncedSearch, refreshKey])

  // Per-status counts for the filter chips.
  useEffect(() => {
    const ctrl = new AbortController()
    api
      .getGraph(ctrl.signal)
      .then((g) => {
        setCounts(g.current_distribution)
        setTotal(g.total_applications)
      })
      .catch(() => {})
    return () => ctrl.abort()
  }, [refreshKey])

  // Auto-cancel delete confirmation.
  useEffect(() => {
    if (confirmDelete === null) return
    const t = setTimeout(() => setConfirmDelete(null), 3500)
    return () => clearTimeout(t)
  }, [confirmDelete])

  const setBusyFor = (id: number, on: boolean) =>
    setBusy((prev) => {
      const next = new Set(prev)
      if (on) next.add(id)
      else next.delete(id)
      return next
    })

  const changeStatus = async (app: Application, status: Status) => {
    const previous = app
    setBusyFor(app.id, true)
    // Optimistic update.
    setApps((list) =>
      list.map((a) => (a.id === app.id ? { ...a, current_status: status, status_updated_at: new Date().toISOString() } : a)),
    )
    try {
      await api.addStatus(app.id, { status })
      notify(`${app.company_name} → ${STATUS_META[status].label}`)
      onChanged()
    } catch (err) {
      setApps((list) => list.map((a) => (a.id === app.id ? previous : a)))
      notify(err instanceof Error ? err.message : 'Failed to update status', 'error')
    } finally {
      setBusyFor(app.id, false)
    }
  }

  const remove = async (app: Application) => {
    if (confirmDelete !== app.id) {
      setConfirmDelete(app.id)
      return
    }
    setConfirmDelete(null)
    setBusyFor(app.id, true)
    try {
      await api.deleteApplication(app.id)
      setApps((list) => list.filter((a) => a.id !== app.id))
      if (expandedId === app.id) setExpandedId(null)
      notify(`Deleted ${app.company_name}`)
      onChanged()
    } catch (err) {
      notify(err instanceof Error ? err.message : 'Failed to delete', 'error')
    } finally {
      setBusyFor(app.id, false)
    }
  }

  const filtersActive = statusFilter !== '' || debouncedSearch.trim() !== ''
  const chips = useMemo(() => [{ key: '' as Filter, label: 'All', count: total }].concat(
    STATUSES.map((s) => ({ key: s as Filter, label: STATUS_META[s].label, count: counts[s] ?? 0 })),
  ), [counts, total])

  return (
    <section className="animate-slide-up space-y-4" aria-labelledby="applications-heading">
      <h2 id="applications-heading" className="sr-only">
        Applications
      </h2>

      {/* Filter bar */}
      <div className="glass flex flex-col gap-3 p-3 lg:flex-row lg:items-center">
        <div className="relative lg:w-80">
          <Search className="pointer-events-none absolute top-1/2 left-3.5 size-4 -translate-y-1/2 text-slate-500" />
          <input
            ref={searchRef}
            id="search-applications"
            type="search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search company or role…"
            className="field pr-16 pl-10 [&::-webkit-search-cancel-button]:hidden"
            aria-label="Search by company or role"
          />
          {search ? (
            <button
              type="button"
              onClick={() => setSearch('')}
              className="btn-icon absolute top-1/2 right-1.5 size-7 -translate-y-1/2"
              aria-label="Clear search"
            >
              <X className="size-3.5" />
            </button>
          ) : (
            <kbd className="pointer-events-none absolute top-1/2 right-3 -translate-y-1/2 rounded border border-white/10 bg-white/5 px-1.5 text-[11px] text-slate-500">
              /
            </kbd>
          )}
        </div>

        <div className="-mx-1 flex gap-1.5 overflow-x-auto px-1 pb-0.5 lg:flex-wrap lg:overflow-visible" role="tablist" aria-label="Filter by status">
          {chips.map((c) => {
            const active = statusFilter === c.key
            const meta = c.key ? STATUS_META[c.key] : null
            return (
              <button
                key={c.key || 'ALL'}
                id={`filter-${c.key || 'ALL'}`}
                type="button"
                role="tab"
                aria-selected={active}
                onClick={() => setStatusFilter(c.key)}
                className={`chip ${
                  active
                    ? 'border-violet-400/50 bg-violet-500/15 text-white shadow-[0_0_20px_-6px_rgb(167_139_250/0.6)]'
                    : 'border-white/[0.08] bg-white/[0.02] text-slate-400 hover:border-white/20 hover:text-slate-200'
                }`}
              >
                {meta && <span className={`size-1.5 rounded-full ${meta.dot}`} />}
                {c.label}
                <span className={`tabular-nums ${active ? 'text-violet-200' : 'text-slate-600'}`}>{c.count}</span>
              </button>
            )
          })}
        </div>
      </div>

      {/* Table */}
      <div className="glass overflow-hidden">
        {error ? (
          <div className="flex flex-col items-center gap-3 px-6 py-16 text-center">
            <p className="text-rose-300">{error}</p>
            <button className="btn btn-ghost" onClick={onChanged} id="retry-load">
              Retry
            </button>
          </div>
        ) : loading && apps.length === 0 ? (
          <SkeletonRows />
        ) : apps.length === 0 ? (
          filtersActive ? (
            <EmptyState
              icon={<SearchX className="size-6" />}
              title="No matching applications"
              body="Try a different status filter or search term."
              action={
                <button
                  className="btn btn-ghost"
                  id="clear-filters"
                  onClick={() => {
                    setStatusFilter('')
                    setSearch('')
                  }}
                >
                  Clear filters
                </button>
              }
            />
          ) : (
            <EmptyState
              icon={<Briefcase className="size-6" />}
              title="Your pipeline is empty"
              body="Add the first role you've applied to and start tracking every step of the journey."
              action={
                <button className="btn btn-primary" onClick={onCreate} id="empty-add-application">
                  <Plus className="size-4" /> Add application
                </button>
              }
            />
          )
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[640px] text-left text-sm">
              <thead>
                <tr className="border-b border-white/[0.06] text-[11px] font-semibold tracking-wider text-slate-500 uppercase">
                  <th className="w-10 py-3 pl-4" aria-label="Expand" />
                  <th className="px-3 py-3">Company</th>
                  <th className="hidden px-3 py-3 md:table-cell">Role</th>
                  <th className="px-3 py-3">Current status</th>
                  <th className="px-3 py-3">Last updated</th>
                  <th className="py-3 pr-4 text-right" aria-label="Actions" />
                </tr>
              </thead>
              <tbody className={loading ? 'opacity-60 transition-opacity' : 'transition-opacity'}>
                {apps.map((app, i) => {
                  const expanded = expandedId === app.id
                  return (
                    <Fragment key={app.id}>
                      <tr
                        className={`group border-b border-white/[0.04] transition-colors duration-150 hover:bg-white/[0.025] ${expanded ? 'bg-white/[0.03]' : ''}`}
                        style={{ animation: `slide-up 0.4s cubic-bezier(0.16,1,0.3,1) ${Math.min(i, 12) * 25}ms both` }}
                      >
                        <td className="py-3 pl-4">
                          <button
                            type="button"
                            id={`expand-${app.id}`}
                            onClick={() => setExpandedId(expanded ? null : app.id)}
                            className="btn-icon"
                            aria-expanded={expanded}
                            aria-label={expanded ? 'Hide history' : 'Show history'}
                          >
                            <ChevronRight className={`size-4 transition-transform duration-200 ${expanded ? 'rotate-90 text-violet-300' : ''}`} />
                          </button>
                        </td>
                        <td className="px-3 py-3">
                          <div className="flex items-center gap-3">
                            <CompanyAvatar name={app.company_name} />
                            <div className="min-w-0">
                              <div className="flex items-center gap-1.5">
                                <button
                                  type="button"
                                  onClick={() => setExpandedId(expanded ? null : app.id)}
                                  className="cursor-pointer truncate font-medium text-slate-100 hover:text-white"
                                >
                                  {app.company_name}
                                </button>
                                {app.job_url && (
                                  <a
                                    href={app.job_url}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="text-slate-500 transition-colors hover:text-violet-300"
                                    aria-label={`Open job posting for ${app.company_name}`}
                                  >
                                    <ExternalLink className="size-3.5" />
                                  </a>
                                )}
                              </div>
                              <div className="truncate text-xs text-slate-500 md:hidden">{app.role_title}</div>
                              {app.location && <div className="hidden truncate text-xs text-slate-500 md:block">{app.location}</div>}
                            </div>
                          </div>
                        </td>
                        <td className="hidden max-w-xs px-3 py-3 md:table-cell">
                          <div className="truncate text-slate-300">{app.role_title}</div>
                          {app.salary_range && <div className="truncate text-xs text-slate-500">{app.salary_range}</div>}
                        </td>
                        <td className="px-3 py-3">
                          <StatusSelect
                            id={`status-${app.id}`}
                            value={app.current_status}
                            busy={busy.has(app.id)}
                            onChange={(s) => changeStatus(app, s)}
                          />
                        </td>
                        <td className="px-3 py-3 whitespace-nowrap">
                          <span className="text-slate-300" title={formatDateTime(app.status_updated_at)}>
                            {relativeTime(app.status_updated_at)}
                          </span>
                        </td>
                        <td className="py-3 pr-4">
                          <div className="flex items-center justify-end gap-0.5 opacity-70 transition-opacity group-hover:opacity-100">
                            <button
                              type="button"
                              id={`edit-${app.id}`}
                              onClick={() => onEdit(app)}
                              className="btn-icon"
                              aria-label={`Edit ${app.company_name}`}
                            >
                              <Pencil className="size-3.5" />
                            </button>
                            {confirmDelete === app.id ? (
                              <button
                                type="button"
                                id={`confirm-delete-${app.id}`}
                                onClick={() => remove(app)}
                                className="animate-pop-in ml-1 cursor-pointer rounded-lg bg-rose-500/15 px-2.5 py-1.5 text-xs font-medium text-rose-300 ring-1 ring-rose-400/30 hover:bg-rose-500/25"
                              >
                                Delete?
                              </button>
                            ) : (
                              <button
                                type="button"
                                id={`delete-${app.id}`}
                                onClick={() => remove(app)}
                                className="btn-icon hover:text-rose-300!"
                                aria-label={`Delete ${app.company_name}`}
                              >
                                <Trash2 className="size-3.5" />
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                      {expanded && (
                        <tr className="border-b border-white/[0.04] bg-ink-950/40">
                          <td colSpan={6} className="p-0">
                            <HistoryPanel app={app} refreshKey={refreshKey} onChanged={onChanged} onEdit={onEdit} notify={notify} />
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {apps.length > 0 && (
        <p className="px-1 text-xs text-slate-500">
          Showing {apps.length} of {total} application{total === 1 ? '' : 's'}
        </p>
      )}
    </section>
  )
}

/* ------------------------------------------------------------------ */
/* Expanded row: history timeline + log status change                  */
/* ------------------------------------------------------------------ */

function HistoryPanel({
  app,
  refreshKey,
  onChanged,
  onEdit,
  notify,
}: {
  app: Application
  refreshKey: number
  onChanged: () => void
  onEdit: (app: Application) => void
  notify: Notify
}) {
  const [detail, setDetail] = useState<ApplicationDetail | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [status, setStatus] = useState<Status>(app.current_status)
  const [notes, setNotes] = useState('')
  const [date, setDate] = useState(() => toLocalInputValue())
  const [dateTouched, setDateTouched] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    const ctrl = new AbortController()
    api
      .getApplication(app.id, ctrl.signal)
      .then((d) => {
        setDetail(d)
        setLoadError(null)
      })
      .catch((err: unknown) => {
        if (!ctrl.signal.aborted) setLoadError(err instanceof Error ? err.message : 'Failed to load history')
      })
    return () => ctrl.abort()
  }, [app.id, app.status_updated_at, refreshKey])

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (saving) return
    setSaving(true)
    try {
      const changed_at = dateTouched && date ? new Date(date).toISOString() : undefined
      const d = await api.addStatus(app.id, { status, notes: notes.trim(), changed_at })
      setDetail(d)
      setNotes('')
      setDate(toLocalInputValue())
      setDateTouched(false)
      notify(`Logged ${STATUS_META[status].label} for ${app.company_name}`)
      onChanged()
    } catch (err) {
      notify(err instanceof Error ? err.message : 'Failed to log status', 'error')
    } finally {
      setSaving(false)
    }
  }

  const history = detail?.history ?? []
  const fieldId = `log-${app.id}`

  return (
    <div className="animate-fade-in grid gap-6 px-4 py-5 sm:px-6 lg:grid-cols-[1.25fr_1fr] lg:pl-16">
      {/* Timeline */}
      <div>
        <h3 className="mb-4 flex items-center gap-2 text-xs font-semibold tracking-wider text-slate-400 uppercase">
          <History className="size-3.5" /> Status history
          {history.length > 0 && <span className="text-slate-600">· {history.length} events</span>}
        </h3>
        {loadError ? (
          <p className="text-sm text-rose-300">{loadError}</p>
        ) : !detail ? (
          <div className="flex items-center gap-2 text-sm text-slate-500">
            <Loader2 className="size-4 animate-spin" /> Loading history…
          </div>
        ) : (
          <ol className="relative">
            {history.map((h, idx) => {
              const meta = STATUS_META[h.status]
              const isLast = idx === history.length - 1
              const prev = history[idx - 1]
              return (
                <li key={h.id} className="relative flex gap-4 pb-5 last:pb-0" style={{ animation: `slide-up 0.35s ease-out ${idx * 40}ms both` }}>
                  {!isLast && <span className="absolute top-5 bottom-0 left-[7px] w-px bg-linear-to-b from-white/15 to-white/5" aria-hidden />}
                  <span className="relative mt-1 flex size-[15px] shrink-0 items-center justify-center">
                    {isLast && <span className={`absolute inset-0 animate-ping rounded-full opacity-40 ${meta.dot}`} />}
                    <span className={`relative size-[11px] rounded-full ring-4 ring-ink-950 ${meta.dot}`} />
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                      <StatusBadge status={h.status} size="sm" />
                      <span className="text-xs text-slate-400" title={formatDateTime(h.changed_at)}>
                        {formatDateTime(h.changed_at)}
                      </span>
                      {prev && (
                        <span className="rounded-md bg-white/[0.04] px-1.5 py-0.5 text-[10px] text-slate-500">
                          +{durationBetween(prev.changed_at, h.changed_at)}
                        </span>
                      )}
                    </div>
                    {h.notes && <p className="mt-1.5 text-sm break-words whitespace-pre-wrap text-slate-300">{h.notes}</p>}
                  </div>
                </li>
              )
            })}
          </ol>
        )}
      </div>

      {/* Details + log form */}
      <div className="space-y-4">
        <div className="rounded-xl border border-white/[0.06] bg-white/[0.02] p-4">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="text-xs font-semibold tracking-wider text-slate-400 uppercase">Details</h3>
            <button type="button" onClick={() => onEdit(app)} className="btn-icon size-7" aria-label="Edit details" id={`edit-details-${app.id}`}>
              <Pencil className="size-3.5" />
            </button>
          </div>
          <dl className="grid grid-cols-1 gap-2 text-sm sm:grid-cols-2">
            <Detail icon={<Briefcase />} label="Role" value={app.role_title} />
            <Detail icon={<MapPin />} label="Location" value={app.location} />
            <Detail icon={<DollarSign />} label="Salary" value={app.salary_range} />
            <Detail icon={<CalendarClock />} label="Added" value={formatDate(app.created_at)} />
          </dl>
          {app.job_url && (
            <a
              href={app.job_url}
              target="_blank"
              rel="noopener noreferrer"
              className="mt-3 inline-flex max-w-full items-center gap-1.5 truncate text-sm text-violet-300 hover:text-violet-200"
            >
              <ExternalLink className="size-3.5 shrink-0" />
              <span className="truncate">{app.job_url}</span>
            </a>
          )}
          {app.notes && <p className="mt-3 border-t border-white/[0.06] pt-3 text-sm whitespace-pre-wrap text-slate-400">{app.notes}</p>}
        </div>

        <form onSubmit={submit} className="rounded-xl border border-violet-400/15 bg-linear-to-br from-violet-500/[0.06] to-transparent p-4">
          <h3 className="mb-3 text-xs font-semibold tracking-wider text-slate-400 uppercase">Log a status change</h3>
          <div className="grid gap-3 sm:grid-cols-2">
            <div>
              <label htmlFor={`${fieldId}-status`} className="field-label">
                Status
              </label>
              <select
                id={`${fieldId}-status`}
                className="field cursor-pointer"
                value={status}
                onChange={(e) => setStatus(e.target.value as Status)}
              >
                {STATUSES.map((s) => (
                  <option key={s} value={s} className="bg-ink-900">
                    {STATUS_META[s].label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label htmlFor={`${fieldId}-date`} className="field-label">
                When
              </label>
              <input
                id={`${fieldId}-date`}
                type="datetime-local"
                className="field"
                value={date}
                max={toLocalInputValue(new Date(Date.now() + 86400000))}
                onChange={(e) => {
                  setDate(e.target.value)
                  setDateTouched(true)
                }}
              />
            </div>
          </div>
          <label htmlFor={`${fieldId}-notes`} className="field-label mt-3">
            Note
          </label>
          <input
            id={`${fieldId}-notes`}
            className="field"
            placeholder="e.g. Passed phone screen"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            maxLength={20000}
          />
          <div className="mt-3 flex justify-end">
            <button type="submit" className="btn btn-primary" disabled={saving} id={`${fieldId}-submit`}>
              {saving ? <Loader2 className="size-4 animate-spin" /> : <Plus className="size-4" />}
              Log status
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

/* ------------------------------------------------------------------ */
/* Small presentational helpers                                        */
/* ------------------------------------------------------------------ */

function Detail({ icon, label, value }: { icon: ReactNode; label: string; value: string }) {
  return (
    <div className="flex min-w-0 items-center gap-2">
      <span className="text-slate-500 [&>svg]:size-3.5">{icon}</span>
      <dt className="sr-only">{label}</dt>
      <dd className={`truncate ${value ? 'text-slate-300' : 'text-slate-600'}`}>{value || `No ${label.toLowerCase()}`}</dd>
    </div>
  )
}

const AVATAR_GRADIENTS = [
  'from-indigo-500 to-violet-500',
  'from-sky-500 to-cyan-400',
  'from-fuchsia-500 to-pink-500',
  'from-emerald-500 to-teal-400',
  'from-amber-500 to-orange-500',
  'from-rose-500 to-red-500',
  'from-violet-500 to-purple-400',
  'from-blue-500 to-indigo-400',
]

function CompanyAvatar({ name }: { name: string }) {
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = (hash * 31 + name.charCodeAt(i)) | 0
  const gradient = AVATAR_GRADIENTS[Math.abs(hash) % AVATAR_GRADIENTS.length]
  const initials = name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase())
    .join('')
  return (
    <span
      className={`flex size-9 shrink-0 items-center justify-center rounded-xl bg-linear-to-br ${gradient} text-xs font-semibold text-white shadow-lg shadow-black/30 ring-1 ring-white/20`}
      aria-hidden
    >
      {initials || '?'}
    </span>
  )
}

function SkeletonRows() {
  return (
    <div className="divide-y divide-white/[0.04]" aria-busy="true" aria-label="Loading applications">
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="flex items-center gap-4 px-4 py-4">
          <div className="size-9 animate-pulse rounded-xl bg-white/[0.06]" />
          <div className="flex-1 space-y-2">
            <div className="h-3 w-40 animate-pulse rounded bg-white/[0.06]" />
            <div className="h-2.5 w-24 animate-pulse rounded bg-white/[0.04]" />
          </div>
          <div className="h-6 w-24 animate-pulse rounded-full bg-white/[0.05]" />
          <div className="hidden h-3 w-16 animate-pulse rounded bg-white/[0.04] sm:block" />
        </div>
      ))}
    </div>
  )
}

function EmptyState({ icon, title, body, action }: { icon: ReactNode; title: string; body: string; action: ReactNode }) {
  return (
    <div className="flex flex-col items-center px-6 py-20 text-center">
      <div className="relative mb-5">
        <div className="absolute inset-0 rounded-2xl bg-violet-500/40 blur-xl" />
        <div className="relative flex size-14 items-center justify-center rounded-2xl bg-linear-to-br from-indigo-500 to-fuchsia-500 text-white shadow-xl">
          {icon}
        </div>
      </div>
      <h3 className="font-display text-lg font-semibold text-white">{title}</h3>
      <p className="mt-1.5 max-w-sm text-sm text-slate-400">{body}</p>
      <div className="mt-6">{action}</div>
    </div>
  )
}
