import {
  Briefcase,
  Calendar,
  CalendarClock,
  ChevronRight,
  ExternalLink,
  FileText,
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
import { Fragment, useEffect, useLayoutEffect, useMemo, useRef, useState, type FormEvent, type ReactNode } from 'react'
import { api } from '../api/client'
import { durationBetween, formatDate, formatDateTime, relativeTime, toLocalInputValue } from '../lib/format'
import { STATUSES, STATUS_META, getNextAllowedStatuses, type Application, type ApplicationDetail, type Status } from '../types'
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
  // Identifies which filter the currently shown rows belong to; changing it remounts
  // the rows so the whole list re-reveals together (refreshes keep it stable).
  const [listKey, setListKey] = useState('')
  const chipRefs = useRef<Record<string, HTMLButtonElement | null>>({})
  const [chipIndicator, setChipIndicator] = useState({ x: 0, y: 0, w: 0, h: 0, ready: false })

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
        setListKey(`${statusFilter}|${debouncedSearch}`)
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
    const allowed = getNextAllowedStatuses(app.current_status)
    if (!allowed.includes(status)) {
      notify(`Cannot transition from ${STATUS_META[app.current_status].label} to ${STATUS_META[status].label}`, 'error')
      return
    }
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
  const chips = useMemo(
    () =>
      [{ key: '' as Filter, label: 'All', count: total }].concat(
        STATUSES.map((s) => ({ key: s as Filter, label: STATUS_META[s].label, count: counts[s] ?? 0 })),
      ),
    [counts, total],
  )

  // Slide the active-filter indicator to the selected chip.
  useLayoutEffect(() => {
    const measure = () => {
      const el = chipRefs.current[statusFilter || 'ALL']
      if (el) {
        setChipIndicator((p) => ({
          x: el.offsetLeft,
          y: el.offsetTop,
          w: el.offsetWidth,
          h: el.offsetHeight,
          ready: p.ready || p.w > 0,
        }))
      }
    }
    measure()
    const raf = requestAnimationFrame(() => setChipIndicator((p) => ({ ...p, ready: true })))
    window.addEventListener('resize', measure)
    document.fonts?.ready.then(measure)
    return () => {
      cancelAnimationFrame(raf)
      window.removeEventListener('resize', measure)
    }
  }, [statusFilter, chips])

  return (
    <section className="space-y-6" aria-labelledby="applications-heading">
      <h2 id="applications-heading" className="sr-only">
        Applications
      </h2>

      {/* Unboxed Filter Zone Framed with animation */}
      <div className="filter-bootup-zone">
        {/* Expanding Container Body / Optical Projection Plane */}
        <div className="filter-bootup-plane relative p-3 bg-white/40 backdrop-blur-xs">
          {/* Content revealed as the horizontal aperture expands */}
          <div className="filter-bootup-content flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            {/* Clinical Search Input */}
            <div className="relative lg:w-80">
              <Search className="pointer-events-none absolute top-1/2 left-3 size-3.5 -translate-y-1/2 text-slate-400" />
              <input
                ref={searchRef}
                id="search-applications"
                type="search"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search company or role"
                className="field-clinical pl-9 pr-14 font-sans text-xs"
                aria-label="Search by company or role"
              />
              {search ? (
                <button
                  type="button"
                  onClick={() => setSearch('')}
                  className="btn-clinical-icon absolute top-1/2 right-1.5 size-6 -translate-y-1/2"
                  aria-label="Clear search"
                >
                  <X className="size-3" />
                </button>
              ) : (
                <kbd className="pointer-events-none absolute top-1/2 right-2.5 -translate-y-1/2 border border-slate-300 bg-slate-50 px-1.5 py-0.5 font-sans text-[10px] font-semibold text-slate-500">
                  /
                </kbd>
              )}
            </div>

            {/* Unboxed Status Filter Strip */}
            <div
              className="relative -mx-1 flex gap-1 overflow-x-auto px-1 py-0.5 lg:flex-wrap lg:overflow-visible"
              role="tablist"
              aria-label="Filter by status"
            >
              <span
                className="filter-indicator"
                style={{
                  width: chipIndicator.w,
                  height: chipIndicator.h,
                  transform: `translate(${chipIndicator.x}px, ${chipIndicator.y}px)`,
                  transitionDuration: chipIndicator.ready ? undefined : '0ms',
                  opacity: chipIndicator.w > 0 ? 1 : 0,
                }}
                aria-hidden
              />
              {chips.map((c) => {
                const active = statusFilter === c.key
                return (
                  <button
                    key={c.key || 'ALL'}
                    ref={(el) => {
                      chipRefs.current[c.key || 'ALL'] = el
                    }}
                    id={`filter-${c.key || 'ALL'}`}
                    type="button"
                    role="tab"
                    aria-selected={active}
                    onClick={() => setStatusFilter(c.key)}
                    className={`filter-item ${active ? 'filter-item-active' : ''}`}
                  >
                    <span>{c.label}</span>
                    <span
                      className={`font-sans text-[11px] font-semibold tabular-nums transition-colors duration-300 ${
                        active ? 'text-sky-200' : 'text-slate-400'
                      }`}
                    >
                      [{c.count}]
                    </span>
                  </button>
                )
              })}
            </div>
          </div>
        </div>

        {/* 4 Corner L-Brackets expanding vertically from horizontal midline */}
        <div className="filter-bootup-rail-top" aria-hidden="true">
          <span className="corner-l-bracket-tl" />
          <span className="corner-l-bracket-tr" />
        </div>
        <div className="filter-bootup-rail-bottom" aria-hidden="true">
          <span className="corner-l-bracket-bl" />
          <span className="corner-l-bracket-br" />
        </div>
      </div>

      {/* Open-Air Data Projection Table (Zero 4-Sided Outer Border) */}
      <div className="overflow-hidden">
        {error ? (
          <div className="flex flex-col items-center gap-3 py-16 text-center">
            <p className="font-sans text-xs text-rose-600">{error}</p>
            <button className="btn-clinical btn-clinical-ghost" onClick={onChanged} id="retry-load">
              Retry
            </button>
          </div>
        ) : loading && apps.length === 0 ? (
          <SkeletonRows />
        ) : apps.length === 0 ? (
          filtersActive ? (
            <EmptyState
              icon={<SearchX className="size-6 text-slate-400" />}
              title="No matches"
              body="No applications match this filter or search."
              action={
                <button
                  className="btn-clinical btn-clinical-ghost"
                  id="clear-filters"
                  onClick={() => {
                    setStatusFilter('')
                    setSearch('')
                  }}
                >
                  Clear Filters
                </button>
              }
            />
          ) : (
            <EmptyState
              icon={<Briefcase className="size-6 text-slate-400" />}
              title="No applications yet"
              body="Add your first application to start tracking."
              action={
                <button
                  className="btn-clinical btn-clinical-primary"
                  onClick={onCreate}
                  id="empty-add-application"
                >
                  <Plus className="size-3.5" />
                  <span>Add application</span>
                </button>
              }
            />
          )
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[700px] text-left text-sm">
              <thead>
                <tr className="border-b border-[rgba(15,20,25,0.14)] font-sans text-[11px] font-bold tracking-wider text-slate-500 uppercase">
                  <th className="w-8 py-2.5 pl-2" aria-label="Expand" />
                  <th className="px-3 py-2.5">Company</th>
                  <th className="hidden px-3 py-2.5 md:table-cell">Role</th>
                  <th className="px-3 py-2.5">Status</th>
                  <th className="hidden px-3 py-2.5 sm:table-cell">Applied</th>
                  <th className="px-3 py-2.5">Updated</th>
                  <th className="py-2.5 pr-2 text-right" aria-label="Actions" />
                </tr>
              </thead>
              <tbody key={listKey} className={`transition-opacity duration-300 ${loading ? 'opacity-60' : ''}`}>
                {apps.map((app, idx) => {
                  const expanded = expandedId === app.id
                  return (
                    <Fragment key={`${listKey}-${app.id}`}>
                      <tr
                        className={`reveal-row group border-b border-[rgba(15,20,25,0.07)] transition-colors duration-300 ${
                          expanded ? 'bg-slate-100/70 shadow-[inset_2px_0_0_#0088cc]' : 'hover:bg-slate-50/80'
                        }`}
                        style={{ animationDelay: `${Math.min(idx, 12) * 45}ms` }}
                      >
                        <td className="py-3 pl-2">
                          <button
                            type="button"
                            id={`expand-${app.id}`}
                            onClick={() => setExpandedId(expanded ? null : app.id)}
                            className="btn-clinical-icon size-6"
                            aria-expanded={expanded}
                            aria-label={expanded ? 'Hide history' : 'Show history'}
                          >
                            <ChevronRight
                              className={`size-3.5 transition-[transform,color] duration-300 ease-[cubic-bezier(0.22,1,0.36,1)] ${
                                expanded ? 'rotate-90 text-[#0088cc]' : 'text-slate-400'
                              }`}
                            />
                          </button>
                        </td>
                        <td className="px-3 py-3">
                          <div className="flex items-center gap-3">
                            <CompanyMark name={app.company_name} />
                            <div className="min-w-0">
                              <div className="flex items-center gap-1.5">
                                <button
                                  type="button"
                                  onClick={() => setExpandedId(expanded ? null : app.id)}
                                  className="cursor-pointer truncate font-semibold text-slate-900 transition-colors duration-200 hover:text-[#0088cc] text-left"
                                >
                                  {app.company_name}
                                </button>
                                {app.job_url && (
                                  <a
                                    href={app.job_url}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="text-slate-400 hover:text-[#0088cc]"
                                    aria-label={`Open job posting for ${app.company_name}`}
                                  >
                                    <ExternalLink className="size-3" />
                                  </a>
                                )}
                              </div>
                              <div className="truncate font-sans text-[11px] text-slate-500 md:hidden">
                                {app.role_title}
                              </div>
                              {app.location && (
                                <div className="hidden truncate font-sans text-[11px] text-slate-400 md:block">
                                  {app.location}
                                </div>
                              )}
                            </div>
                          </div>
                        </td>
                        <td className="hidden max-w-xs px-3 py-3 md:table-cell">
                          <div className="truncate font-medium text-slate-800">{app.role_title}</div>
                          {app.salary_range && (
                            <div className="truncate font-sans text-xs text-slate-500 tabular-nums">
                              {app.salary_range}
                            </div>
                          )}
                        </td>
                        <td className="px-3 py-3">
                          <StatusSelect
                            id={`status-${app.id}`}
                            value={app.current_status}
                            busy={busy.has(app.id)}
                            onChange={(s) => changeStatus(app, s)}
                          />
                        </td>
                        <td className="hidden px-3 py-3 whitespace-nowrap sm:table-cell">
                          <span
                            className="font-sans text-xs text-slate-600 tabular-nums"
                            title={formatDateTime(app.applied_on)}
                          >
                            {formatDate(app.applied_on)}
                          </span>
                        </td>
                        <td className="px-3 py-3 whitespace-nowrap">
                          <span
                            className="font-sans text-xs text-slate-500 tabular-nums"
                            title={formatDateTime(app.status_updated_at)}
                          >
                            {relativeTime(app.status_updated_at)}
                          </span>
                        </td>
                        <td className="py-3 pr-2">
                          <div className="flex items-center justify-end gap-1 opacity-60 transition-opacity duration-200 group-hover:opacity-100">
                            <button
                              type="button"
                              id={`edit-${app.id}`}
                              onClick={() => onEdit(app)}
                              className="btn-clinical-icon size-7"
                              aria-label={`Edit ${app.company_name}`}
                            >
                              <Pencil className="size-3.5 text-slate-500 hover:text-slate-900" />
                            </button>
                            {confirmDelete === app.id ? (
                              <button
                                type="button"
                                id={`confirm-delete-${app.id}`}
                                onClick={() => remove(app)}
                                className="animate-pop-in ml-1 cursor-pointer border border-rose-400 bg-rose-50 px-2 py-1 font-sans text-[10px] font-bold text-rose-700 transition-colors hover:bg-rose-100"
                              >
                                Confirm?
                              </button>
                            ) : (
                              <button
                                type="button"
                                id={`delete-${app.id}`}
                                onClick={() => remove(app)}
                                className="btn-clinical-icon size-7 text-slate-400 hover:text-rose-600"
                                aria-label={`Delete ${app.company_name}`}
                              >
                                <Trash2 className="size-3.5" />
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                      {expanded && (
                        <tr className="border-b border-[rgba(15,20,25,0.08)] bg-slate-50/60">
                          <td colSpan={7} className="panel-open p-0">
                            <HistoryPanel
                              app={app}
                              refreshKey={refreshKey}
                              onChanged={onChanged}
                              onEdit={onEdit}
                              notify={notify}
                            />
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
        <div className="flex items-center justify-between border-t border-[rgba(15,20,25,0.08)] pt-3 font-sans text-[11px] font-medium tracking-wider text-slate-500 uppercase">
          <span>
            {apps.length} of {total} shown
          </span>
        </div>
      )}
    </section>
  )
}

/* ------------------------------------------------------------------ */
/* Expanded row: status timeline & telemetry sheet                    */
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
  const currentStatus = detail?.current_status ?? app.current_status
  const allowedStatuses = useMemo(() => getNextAllowedStatuses(currentStatus), [currentStatus])
  const isTerminal = allowedStatuses.length === 0
  const [status, setStatus] = useState<Status>(allowedStatuses[0] ?? currentStatus)
  const effectiveStatus = allowedStatuses.includes(status) ? status : (allowedStatuses[0] ?? currentStatus)
  const [notes, setNotes] = useState('')
  const [date, setDate] = useState(() => toLocalInputValue())
  const [dateTouched, setDateTouched] = useState(false)
  const [saving, setSaving] = useState(false)
  const maxDate = useMemo(() => toLocalInputValue(new Date(Date.now() + 86400000)), [])

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
    if (saving || isTerminal) return
    setSaving(true)
    try {
      const changed_at = dateTouched && date ? new Date(date).toISOString() : undefined
      const d = await api.addStatus(app.id, { status: effectiveStatus, notes: notes.trim(), changed_at })
      setDetail(d)
      setNotes('')
      setDate(toLocalInputValue())
      setDateTouched(false)
      notify(`Logged ${STATUS_META[effectiveStatus].label} for ${app.company_name}`)
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
    <div className="grid gap-6 px-4 py-6 sm:px-6 lg:grid-cols-[1.25fr_1fr] lg:pl-12 border-l-2 border-[#0088cc]">
      {/* Surgical Status History Timeline */}
      <div>
        <h3 className="mb-4 flex items-center gap-2 font-sans text-[11px] font-bold tracking-wider text-slate-500 uppercase">
          <History className="size-3.5 text-slate-500" />
          <span>Status history</span>
          {history.length > 0 && <span className="text-slate-400 font-normal">· {history.length} events</span>}
        </h3>
        {loadError ? (
          <p className="font-sans text-xs text-rose-600">{loadError}</p>
        ) : !detail ? (
          <div className="flex items-center gap-2 font-sans text-xs text-slate-400">
            <Loader2 className="size-3.5 animate-spin" /> Loading history…
          </div>
        ) : (
          <ol className="relative pl-2">
            {history.map((h, idx) => {
              const meta = STATUS_META[h.status]
              const isLast = idx === history.length - 1
              const prev = history[idx - 1]
              return (
                <li
                  key={h.id}
                  className="reveal relative flex gap-3.5 pb-5 last:pb-1"
                  style={{ animationDelay: `${120 + idx * 50}ms` }}
                >
                  {!isLast && (
                    <span
                      className="absolute top-4 bottom-0 left-[4px] w-px bg-slate-300"
                      aria-hidden
                    />
                  )}
                  <span className="relative mt-1 flex size-[9px] shrink-0 items-center justify-center">
                    <span
                      className={`size-2 ${meta.dot}`}
                      style={{ borderRadius: '1px' }}
                    />
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                      <StatusBadge status={h.status} size="sm" />
                      <span
                        className="font-sans text-[11px] text-slate-500 tabular-nums"
                        title={formatDateTime(h.changed_at)}
                      >
                        {formatDateTime(h.changed_at)}
                      </span>
                      {prev && (
                        <span className="border border-slate-200 bg-white px-1.5 py-0.2 font-sans text-[10px] font-semibold text-slate-500 tabular-nums">
                          +{durationBetween(prev.changed_at, h.changed_at)}
                        </span>
                      )}
                    </div>
                    {h.notes && (
                      <p className="mt-1 text-xs break-words whitespace-pre-wrap text-slate-700 bg-white p-2.5 border border-slate-200/80 font-sans">
                        {h.notes}
                      </p>
                    )}
                  </div>
                </li>
              )
            })}
          </ol>
        )}
      </div>

      {/* Details Sheet & Status Log Form */}
      <div className="space-y-4">
        {/* Specification Details Box */}
        <div className="border border-slate-300/80 bg-white p-4">
          <div className="mb-3 flex items-center justify-between border-b border-slate-200 pb-2">
            <h3 className="font-sans text-[11px] font-bold tracking-wider text-slate-500 uppercase">
              Details
            </h3>
            <button
              type="button"
              onClick={() => onEdit(app)}
              className="btn-clinical-icon size-6"
              aria-label="Edit details"
              id={`edit-details-${app.id}`}
            >
              <Pencil className="size-3" />
            </button>
          </div>
          <dl className="grid grid-cols-1 gap-2 text-xs sm:grid-cols-2">
            <Detail icon={<Briefcase />} label="Role" value={app.role_title} />
            <Detail icon={<MapPin />} label="Location" value={app.location} />
            <Detail
              icon={<span className="font-sans text-[10px] font-bold text-slate-500">CHF</span>}
              label="Salary"
              value={app.salary_range}
            />
            <Detail icon={<Calendar />} label="Applied" value={formatDate(app.applied_on)} />
            <Detail icon={<CalendarClock />} label="Added" value={formatDate(app.created_at)} />
          </dl>
          {app.job_url && (
            <a
              href={app.job_url}
              target="_blank"
              rel="noopener noreferrer"
              className="mt-3 inline-flex max-w-full items-center gap-1.5 truncate font-sans text-xs text-[#0088cc] hover:underline"
            >
              <ExternalLink className="size-3 shrink-0" />
              <span className="truncate">{app.job_url}</span>
            </a>
          )}
          {app.job_description && (
            <div className="mt-3 border-t border-slate-200 pt-3">
              <div className="mb-1 flex items-center gap-1.5 font-sans text-[11px] font-bold tracking-wider text-slate-500 uppercase">
                <FileText className="size-3 text-[#0088cc]" />
                <span>Job Description</span>
              </div>
              <div className="max-h-48 overflow-y-auto border border-slate-200 bg-slate-50 p-2.5 font-sans text-xs leading-relaxed whitespace-pre-wrap text-slate-700">
                {app.job_description}
              </div>
            </div>
          )}
          {app.notes && (
            <p className="mt-3 border-t border-slate-200 pt-2 text-xs whitespace-pre-wrap text-slate-600 font-sans">
              {app.notes}
            </p>
          )}
        </div>

        {/* Transition Log Form */}
        {isTerminal ? (
          <div className="border border-slate-300/80 bg-slate-50/70 p-4 font-sans">
            <h3 className="mb-2 font-sans text-[11px] font-bold tracking-wider text-slate-500 uppercase">
              Application Lifecycle Complete
            </h3>
            <p className="text-xs text-slate-600 leading-relaxed">
              This application has reached terminal status <span className="font-semibold text-slate-800">{STATUS_META[currentStatus].label}</span>. No further forward status changes can be logged.
            </p>
          </div>
        ) : (
          <form
            onSubmit={submit}
            className="border border-slate-300/80 bg-white p-4"
          >
            <h3 className="mb-3 font-sans text-[11px] font-bold tracking-wider text-slate-500 uppercase">
              Log forward status change
            </h3>
            <div className="grid gap-3 sm:grid-cols-2">
              <div>
                <label htmlFor={`${fieldId}-status`} className="field-label">
                  Next Status
                </label>
                <select
                  id={`${fieldId}-status`}
                  className="field-clinical font-sans text-xs uppercase"
                  value={effectiveStatus}
                  onChange={(e) => setStatus(e.target.value as Status)}
                >
                  {allowedStatuses.map((s) => (
                    <option key={s} value={s}>
                      {STATUS_META[s].label}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label htmlFor={`${fieldId}-date`} className="field-label">
                  Date
                </label>
                <input
                  id={`${fieldId}-date`}
                  type="datetime-local"
                  className="field-clinical font-sans text-xs"
                  value={date}
                  max={maxDate}
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
              className="field-clinical"
              placeholder="e.g. Completed initial recruiter screen"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              maxLength={20000}
            />
            <div className="mt-3 flex justify-end">
              <button
                type="submit"
                className="btn-clinical btn-clinical-primary"
                disabled={saving || allowedStatuses.length === 0}
                id={`${fieldId}-submit`}
              >
                {saving ? <Loader2 className="size-3.5 animate-spin" /> : <Plus className="size-3.5" />}
                <span>Log status</span>
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}

/* ------------------------------------------------------------------ */
/* Small presentational helpers                                        */
/* ------------------------------------------------------------------ */

function Detail({ icon, label, value }: { icon: ReactNode; label: string; value: string }) {
  return (
    <div className="flex min-w-0 items-center gap-1.5">
      <span className="text-slate-400 [&>svg]:size-3">{icon}</span>
      <dt className="sr-only">{label}</dt>
      <dd className={`truncate font-sans ${value ? 'text-slate-800' : 'text-slate-400'}`}>
        {value || `No ${label.toLowerCase()}`}
      </dd>
    </div>
  )
}

function CompanyMark({ name }: { name: string }) {
  const initials = name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase())
    .join('')
  return (
    <span
      className="flex size-7 shrink-0 items-center justify-center border border-slate-300 bg-white font-sans text-[11px] font-bold text-slate-800"
      style={{ clipPath: 'polygon(0 0, calc(100% - 3px) 0, 100% 3px, 100% 100%, 0 100%)' }}
      aria-hidden
    >
      {initials || '?'}
    </span>
  )
}

function SkeletonRows() {
  return (
    <div className="divide-y divide-slate-200" aria-busy="true" aria-label="Loading applications">
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="flex items-center gap-4 px-3 py-3">
          <div className="size-7 animate-pulse bg-slate-200" />
          <div className="flex-1 space-y-1.5">
            <div className="h-3 w-40 animate-pulse bg-slate-200" />
            <div className="h-2.5 w-24 animate-pulse bg-slate-100" />
          </div>
          <div className="h-5 w-20 animate-pulse bg-slate-200" />
          <div className="hidden h-3 w-16 animate-pulse bg-slate-100 sm:block" />
        </div>
      ))}
    </div>
  )
}

function EmptyState({ icon, title, body, action }: { icon: ReactNode; title: string; body: string; action: ReactNode }) {
  return (
    <div className="reveal flex flex-col items-center px-4 py-16 text-center">
      <div className="mb-3 flex size-12 items-center justify-center border border-slate-300 bg-white shadow-xs">
        {icon}
      </div>
      <h3 className="font-sans text-xs font-bold tracking-wider text-slate-800 uppercase">{title}</h3>
      <p className="mt-1 max-w-sm text-xs text-slate-500 font-sans">{body}</p>
      <div className="mt-4">{action}</div>
    </div>
  )
}
