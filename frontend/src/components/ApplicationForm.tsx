import { Bookmark, Building2, FileText, Link2, Loader2, MapPin, Send, StickyNote, X } from 'lucide-react'
import { useEffect, useId, useRef, useState, type FormEvent } from 'react'
import { api } from '../api/client'
import type { Application, ApplicationDetail, InitialStatus } from '../types'

export type FormMode = { kind: 'create' } | { kind: 'edit'; app: Application }

interface ApplicationFormProps {
  mode: FormMode
  onClose: () => void
  onSaved: (app: ApplicationDetail, created: boolean) => void
}

interface FormState {
  company_name: string
  role_title: string
  initial_status: InitialStatus
  location: string
  salary_range: string
  job_url: string
  notes: string
  job_description: string
}

const empty: FormState = {
  company_name: '',
  role_title: '',
  initial_status: 'APPLIED',
  location: '',
  salary_range: '',
  job_url: '',
  notes: '',
  job_description: '',
}

function isValidUrl(v: string) {
  if (!v) return true
  try {
    const u = new URL(v)
    return u.protocol === 'http:' || u.protocol === 'https:'
  } catch {
    return false
  }
}

/** Clinical horizon record entry/edit modal with elegant sans-serif typography. */
export default function ApplicationForm({ mode, onClose, onSaved }: ApplicationFormProps) {
  const isEdit = mode.kind === 'edit'
  const [form, setForm] = useState<FormState>(() =>
    mode.kind === 'edit'
      ? {
          ...empty,
          company_name: mode.app.company_name,
          role_title: mode.app.role_title,
          location: mode.app.location,
          salary_range: mode.app.salary_range,
          job_url: mode.app.job_url,
          notes: mode.app.notes,
          job_description: mode.app.job_description || '',
        }
      : empty,
  )
  const [touched, setTouched] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const firstRef = useRef<HTMLInputElement>(null)
  const uid = useId()

  useEffect(() => {
    firstRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = prevOverflow
    }
  }, [onClose])

  const set = <K extends keyof FormState>(k: K, v: FormState[K]) => setForm((f) => ({ ...f, [k]: v }))

  const errors = {
    company_name: !form.company_name.trim() ? 'Company entity name is required' : '',
    role_title: !form.role_title.trim() ? 'Role title is required' : '',
    job_url: !isValidUrl(form.job_url.trim()) ? 'Must be a valid http(s) URL' : '',
  }
  const hasErrors = Object.values(errors).some(Boolean)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setTouched(true)
    if (hasErrors || saving) return
    setSaving(true)
    setError(null)
    const payload = {
      company_name: form.company_name.trim(),
      role_title: form.role_title.trim(),
      location: form.location.trim(),
      salary_range: form.salary_range.trim(),
      job_url: form.job_url.trim(),
      notes: form.notes.trim(),
      job_description: form.job_description.trim(),
    }
    try {
      const saved =
        mode.kind === 'edit'
          ? await api.updateApplication(mode.app.id, payload)
          : await api.createApplication({ ...payload, initial_status: form.initial_status })
      onSaved(saved, !isEdit)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to commit record')
      setSaving(false)
    }
  }

  const showErr = (k: keyof typeof errors) => touched && errors[k]

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4 backdrop-blur-xs font-sans"
      onMouseDown={(e) => e.target === e.currentTarget && onClose()}
      role="presentation"
    >
      <form
        onSubmit={submit}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) submit(e)
        }}
        role="dialog"
        aria-modal="true"
        aria-labelledby={`${uid}-title`}
        className="relative max-h-[92vh] w-full max-w-xl overflow-y-auto border border-slate-300 bg-white p-6 shadow-2xl text-slate-800"
        noValidate
      >
        {/* Exterior Corner L-Brackets */}
        <span className="corner-l-bracket-tl" />
        <span className="corner-l-bracket-tr" />
        <span className="corner-l-bracket-bl" />
        <span className="corner-l-bracket-br" />

        {/* Modal Header */}
        <header className="mb-4">
          <div className="flex items-center justify-between">
            <span className="font-sans text-[11px] font-bold tracking-wider text-slate-400 uppercase">
              {isEdit ? 'RECORD_OPS // MODIFY' : 'RECORD_OPS // INITIALIZE'}
            </span>
            <button
              type="button"
              onClick={onClose}
              className="btn-clinical-icon size-7 text-slate-400 hover:text-slate-900"
              aria-label="Close dialog"
            >
              <X className="size-4" />
            </button>
          </div>
          <h2 id={`${uid}-title`} className="mt-1 font-display text-2xl font-light tracking-wide text-slate-900 uppercase">
            {isEdit ? `Edit: ${mode.app.company_name}` : 'New Application Record'}
          </h2>
          <div className="tapered-datum-rule" />
        </header>

        {error && (
          <div className="mb-4 border border-rose-300 bg-rose-50 p-2.5 font-sans text-xs text-rose-700">
            {error}
          </div>
        )}

        <div className="space-y-4">
          {/* Company & Role */}
          <div className="grid gap-3 sm:grid-cols-2">
            <div>
              <label htmlFor={`${uid}-company`} className="field-label">
                Company Entity *
              </label>
              <div className="relative">
                <Building2 className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-slate-400" />
                <input
                  ref={firstRef}
                  id={`${uid}-company`}
                  required
                  value={form.company_name}
                  onChange={(e) => set('company_name', e.target.value)}
                  placeholder="e.g. Google, Stripe"
                  className="field-clinical pl-8"
                  aria-invalid={Boolean(showErr('company_name'))}
                />
              </div>
              {showErr('company_name') && (
                <p className="mt-1 font-sans text-[11px] text-rose-600">{errors.company_name}</p>
              )}
            </div>

            <div>
              <label htmlFor={`${uid}-role`} className="field-label">
                Spec / Role Title *
              </label>
              <input
                id={`${uid}-role`}
                required
                value={form.role_title}
                onChange={(e) => set('role_title', e.target.value)}
                placeholder="e.g. Senior Backend Engineer"
                className="field-clinical"
                aria-invalid={Boolean(showErr('role_title'))}
              />
              {showErr('role_title') && (
                <p className="mt-1 font-sans text-[11px] text-rose-600">{errors.role_title}</p>
              )}
            </div>
          </div>

          {/* Initial status (creation only) */}
          {!isEdit && (
            <div>
              <span className="field-label">Initial State Routing</span>
              <div className="grid grid-cols-2 gap-2">
                {[
                  { v: 'APPLIED' as const, label: 'Applied', desc: 'Application dispatched', icon: <Send className="size-3.5" /> },
                  { v: 'SAVED' as const, label: 'Saved', desc: 'Bookmarked, not yet submitted', icon: <Bookmark className="size-3.5" /> },
                ].map((o) => {
                  const active = form.initial_status === o.v
                  return (
                    <button
                      key={o.v}
                      type="button"
                      id={`initial-status-${o.v.toLowerCase()}`}
                      onClick={() => set('initial_status', o.v)}
                      aria-pressed={active}
                      className={`flex cursor-pointer items-start gap-2 border p-2.5 text-left transition-all ${
                        active
                          ? 'border-[var(--color-ink-slate)] bg-slate-100 text-slate-900'
                          : 'border-slate-200 bg-white text-slate-500 hover:border-slate-300 hover:text-slate-800'
                      }`}
                    >
                      <span className={`mt-0.5 ${active ? 'text-[#0088cc]' : 'text-slate-400'}`}>{o.icon}</span>
                      <div>
                        <div className="font-sans text-xs font-bold uppercase">{o.label}</div>
                        <div className="text-[11px] text-slate-500">{o.desc}</div>
                      </div>
                    </button>
                  )
                })}
              </div>
            </div>
          )}

          {/* Location & Salary */}
          <div className="grid gap-3 sm:grid-cols-2">
            <div>
              <label htmlFor={`${uid}-location`} className="field-label">
                Location / Work Model
              </label>
              <div className="relative">
                <MapPin className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-slate-400" />
                <input
                  id={`${uid}-location`}
                  value={form.location}
                  onChange={(e) => set('location', e.target.value)}
                  placeholder="e.g. Zurich, Remote"
                  className="field-clinical pl-8"
                />
              </div>
            </div>

            <div>
              <label htmlFor={`${uid}-salary`} className="field-label">
                Compensation / Range
              </label>
              <div className="relative">
                <span className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 font-sans text-[11px] font-bold text-slate-400">
                  CHF
                </span>
                <input
                  id={`${uid}-salary`}
                  value={form.salary_range}
                  onChange={(e) => set('salary_range', e.target.value)}
                  placeholder="140'000 – 165'000"
                  className="field-clinical pl-10 tabular-nums"
                />
              </div>
            </div>
          </div>

          {/* Job Posting URL */}
          <div>
            <label htmlFor={`${uid}-url`} className="field-label">
              Posting URL
            </label>
            <div className="relative">
              <Link2 className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-slate-400" />
              <input
                id={`${uid}-url`}
                type="url"
                value={form.job_url}
                onChange={(e) => set('job_url', e.target.value)}
                placeholder="https://..."
                className="field-clinical pl-8 font-sans text-xs"
                aria-invalid={Boolean(showErr('job_url'))}
              />
            </div>
            {showErr('job_url') && (
              <p className="mt-1 font-sans text-[11px] text-rose-600">{errors.job_url}</p>
            )}
          </div>

          {/* Job Description */}
          <div>
            <label htmlFor={`${uid}-job-desc`} className="field-label flex items-center justify-between">
              <span className="flex items-center gap-1.5">
                <FileText className="size-3 text-[#0088cc]" />
                Job Description
              </span>
              <span className="text-[10px] font-normal text-slate-400">RAW SPECIFICATION</span>
            </label>
            <textarea
              id={`${uid}-job-desc`}
              rows={3}
              value={form.job_description}
              onChange={(e) => set('job_description', e.target.value)}
              placeholder="Paste job posting description, requirements, and responsibilities…"
              className="field-clinical resize-y font-sans text-xs"
              maxLength={50000}
            />
          </div>

          {/* Notes */}
          <div>
            <label htmlFor={`${uid}-notes`} className="field-label flex items-center justify-between">
              <span className="flex items-center gap-1.5">
                <StickyNote className="size-3 text-slate-400" />
                Telemetry Notes
              </span>
              <span className="text-[10px] font-normal text-slate-400">OPTIONAL</span>
            </label>
            <textarea
              id={`${uid}-notes`}
              rows={2}
              value={form.notes}
              onChange={(e) => set('notes', e.target.value)}
              placeholder="Referral contacts, recruiter handles, interview prep…"
              className="field-clinical resize-y font-sans text-xs"
              maxLength={20000}
            />
          </div>
        </div>

        {/* Action Spine */}
        <footer className="mt-6 flex flex-wrap items-center justify-between gap-3 border-t border-slate-200 pt-4">
          <span className="font-sans text-[11px] font-medium text-slate-400">
            [ ⌘ + ENTER ] TO COMMIT
          </span>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={onClose}
              disabled={saving}
              className="btn-clinical btn-clinical-ghost"
              id="form-cancel"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={saving}
              className="btn-clinical btn-clinical-primary"
              id="form-submit"
            >
              {saving ? <Loader2 className="size-3.5 animate-spin" /> : null}
              <span>{isEdit ? 'Save Changes' : 'Commit Record'}</span>
            </button>
          </div>
        </footer>
      </form>
    </div>
  )
}
