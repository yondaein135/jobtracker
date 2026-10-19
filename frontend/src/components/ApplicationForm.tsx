import { Bookmark, Building2, DollarSign, Link2, Loader2, MapPin, Send, StickyNote, X } from 'lucide-react'
import { useEffect, useId, useRef, useState, type FormEvent, type ReactNode } from 'react'
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
}

const empty: FormState = {
  company_name: '',
  role_title: '',
  initial_status: 'APPLIED',
  location: '',
  salary_range: '',
  job_url: '',
  notes: '',
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

/** Quick-add / edit modal for an application. */
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
    company_name: !form.company_name.trim() ? 'Company is required' : '',
    role_title: !form.role_title.trim() ? 'Role is required' : '',
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
    }
    try {
      const saved =
        mode.kind === 'edit'
          ? await api.updateApplication(mode.app.id, payload)
          : await api.createApplication({ ...payload, initial_status: form.initial_status })
      onSaved(saved, !isEdit)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
      setSaving(false)
    }
  }

  const showErr = (k: keyof typeof errors) => touched && errors[k]

  return (
    <div
      className="animate-fade-in fixed inset-0 z-40 flex items-end justify-center bg-black/60 p-0 backdrop-blur-sm sm:items-center sm:p-6"
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
        className="animate-slide-up relative max-h-[94vh] w-full max-w-xl overflow-y-auto rounded-t-3xl border border-white/10 bg-ink-900/95 shadow-[0_30px_120px_-20px_rgb(99_102_241/0.35)] sm:rounded-3xl"
        noValidate
      >
        <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-linear-to-r from-transparent via-violet-400/60 to-transparent" />

        <header className="flex items-start justify-between gap-4 px-6 pt-6 pb-4 sm:px-7">
          <div>
            <h2 id={`${uid}-title`} className="font-display text-xl font-semibold text-white">
              {isEdit ? 'Edit application' : 'New application'}
            </h2>
            <p className="mt-1 text-sm text-slate-400">
              {isEdit ? 'Update the details for this role.' : 'Track a role you applied to or want to apply to.'}
            </p>
          </div>
          <button type="button" onClick={onClose} className="btn-icon -mt-1 -mr-2" aria-label="Close" id="close-form">
            <X className="size-5" />
          </button>
        </header>

        <div className="space-y-5 px-6 pb-2 sm:px-7">
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Company" required error={showErr('company_name')} htmlFor={`${uid}-company`}>
              <IconInput icon={<Building2 />}>
                <input
                  ref={firstRef}
                  id={`${uid}-company`}
                  className="field pl-10"
                  placeholder="Acme Corp"
                  value={form.company_name}
                  onChange={(e) => set('company_name', e.target.value)}
                  maxLength={300}
                  autoComplete="off"
                />
              </IconInput>
            </Field>
            <Field label="Role" required error={showErr('role_title')} htmlFor={`${uid}-role`}>
              <input
                id={`${uid}-role`}
                className="field"
                placeholder="Senior Backend Engineer"
                value={form.role_title}
                onChange={(e) => set('role_title', e.target.value)}
                maxLength={300}
                autoComplete="off"
              />
            </Field>
          </div>

          {!isEdit && (
            <fieldset>
              <legend className="field-label">Initial status</legend>
              <div className="grid grid-cols-2 gap-2 rounded-xl border border-white/10 bg-ink-950/60 p-1">
                {(
                  [
                    { v: 'APPLIED', label: 'Applied', hint: 'Application sent', icon: <Send className="size-4" /> },
                    { v: 'SAVED', label: 'Saved', hint: 'Not applied yet', icon: <Bookmark className="size-4" /> },
                  ] as const
                ).map((o) => {
                  const active = form.initial_status === o.v
                  return (
                    <button
                      key={o.v}
                      type="button"
                      id={`initial-status-${o.v.toLowerCase()}`}
                      onClick={() => set('initial_status', o.v)}
                      aria-pressed={active}
                      className={`flex cursor-pointer items-center gap-2.5 rounded-lg px-3 py-2.5 text-left transition-all duration-200 ${
                        active
                          ? 'bg-linear-to-br from-indigo-500/25 to-fuchsia-500/20 text-white ring-1 ring-violet-400/40'
                          : 'text-slate-400 hover:bg-white/[0.04] hover:text-slate-200'
                      }`}
                    >
                      <span className={active ? 'text-violet-300' : ''}>{o.icon}</span>
                      <span>
                        <span className="block text-sm font-medium">{o.label}</span>
                        <span className="block text-[11px] text-slate-500">{o.hint}</span>
                      </span>
                    </button>
                  )
                })}
              </div>
            </fieldset>
          )}

          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Location" htmlFor={`${uid}-location`}>
              <IconInput icon={<MapPin />}>
                <input
                  id={`${uid}-location`}
                  className="field pl-10"
                  placeholder="Remote · Berlin"
                  value={form.location}
                  onChange={(e) => set('location', e.target.value)}
                  maxLength={300}
                />
              </IconInput>
            </Field>
            <Field label="Salary range" htmlFor={`${uid}-salary`}>
              <IconInput icon={<DollarSign />}>
                <input
                  id={`${uid}-salary`}
                  className="field pl-10"
                  placeholder="€80k – €95k"
                  value={form.salary_range}
                  onChange={(e) => set('salary_range', e.target.value)}
                  maxLength={300}
                />
              </IconInput>
            </Field>
          </div>

          <Field label="Job URL" error={showErr('job_url')} htmlFor={`${uid}-url`}>
            <IconInput icon={<Link2 />}>
              <input
                id={`${uid}-url`}
                type="url"
                className="field pl-10"
                placeholder="https://careers.acme.com/jobs/123"
                value={form.job_url}
                onChange={(e) => set('job_url', e.target.value)}
                maxLength={2048}
              />
            </IconInput>
          </Field>

          <Field label="Notes" htmlFor={`${uid}-notes`}>
            <IconInput icon={<StickyNote />} top>
              <textarea
                id={`${uid}-notes`}
                className="field min-h-24 resize-y pl-10"
                placeholder="Referral from Jane, team uses Go + Postgres…"
                value={form.notes}
                onChange={(e) => set('notes', e.target.value)}
                rows={3}
              />
            </IconInput>
          </Field>

          {error && (
            <div className="animate-pop-in rounded-xl border border-rose-400/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200" role="alert">
              {error}
            </div>
          )}
        </div>

        <footer className="sticky bottom-0 mt-4 flex items-center justify-between gap-3 border-t border-white/[0.06] bg-ink-900/95 px-6 py-4 backdrop-blur sm:px-7">
          <span className="hidden text-xs text-slate-500 sm:block">
            <kbd className="rounded border border-white/10 bg-white/5 px-1.5 py-0.5 font-sans">Ctrl</kbd> +{' '}
            <kbd className="rounded border border-white/10 bg-white/5 px-1.5 py-0.5 font-sans">Enter</kbd> to save
          </span>
          <div className="flex flex-1 justify-end gap-2">
            <button type="button" onClick={onClose} className="btn btn-ghost" id="cancel-form">
              Cancel
            </button>
            <button type="submit" className="btn btn-primary min-w-32" disabled={saving} id="submit-form">
              {saving && <Loader2 className="size-4 animate-spin" />}
              {isEdit ? 'Save changes' : 'Add application'}
            </button>
          </div>
        </footer>
      </form>
    </div>
  )
}

function Field({
  label,
  required,
  error,
  htmlFor,
  children,
}: {
  label: string
  required?: boolean
  error?: string | false
  htmlFor: string
  children: ReactNode
}) {
  return (
    <div>
      <label htmlFor={htmlFor} className="field-label">
        {label}
        {required && <span className="ml-0.5 text-fuchsia-400">*</span>}
      </label>
      {children}
      {error && <p className="animate-fade-in mt-1.5 text-xs text-rose-300">{error}</p>}
    </div>
  )
}

function IconInput({ icon, top, children }: { icon: ReactNode; top?: boolean; children: ReactNode }) {
  return (
    <div className="relative">
      <span
        className={`pointer-events-none absolute left-3.5 text-slate-500 [&>svg]:size-4 ${top ? 'top-3' : 'top-1/2 -translate-y-1/2'}`}
      >
        {icon}
      </span>
      {children}
    </div>
  )
}
