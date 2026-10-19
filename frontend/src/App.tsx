import { AlertCircle, CheckCircle2, GitFork, LayoutGrid, Plus, X } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { AmbientCanvas } from './components/AmbientCanvas'
import ApplicationForm, { type FormMode } from './components/ApplicationForm'
import ApplicationList from './components/ApplicationList'
import PipelineGraph from './components/PipelineGraph'
import type { Application, ApplicationDetail } from './types'

type ActiveTab = 'table' | 'graph'

interface Toast {
  id: number
  message: string
  kind: 'success' | 'error'
}

export default function App() {
  const [activeTab, setActiveTab] = useState<ActiveTab>('table')
  const [formMode, setFormMode] = useState<FormMode | null>(null)
  const [refreshKey, setRefreshKey] = useState(0)
  const [toasts, setToasts] = useState<Toast[]>([])
  const [utcTime, setUtcTime] = useState('')

  useEffect(() => {
    const updateTime = () => {
      const now = new Date()
      setUtcTime(now.toISOString().replace('T', ' ').slice(0, 19) + ' UTC')
    }
    updateTime()
    const timer = setInterval(updateTime, 1000)
    return () => clearInterval(timer)
  }, [])

  const notify = useCallback((message: string, kind: 'success' | 'error' = 'success') => {
    const id = Date.now() + Math.random()
    setToasts((prev) => [...prev, { id, message, kind }])
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id))
    }, 3800)
  }, [])

  const removeToast = (id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }

  const triggerRefresh = useCallback(() => {
    setRefreshKey((k) => k + 1)
  }, [])

  const handleSaved = (app: ApplicationDetail, created: boolean) => {
    setFormMode(null)
    triggerRefresh()
    notify(
      created
        ? `Added record: ${app.company_name} [${app.role_title}]`
        : `Updated record: ${app.company_name}`,
      'success',
    )
  }

  return (
    <div className="relative min-h-screen bg-[var(--color-canvas-base)] font-sans text-[var(--color-ink-primary)]">
      {/* Clinical Horizon Atmospheric Background Mesh & Crosshairs */}
      <AmbientCanvas />

      {/* Main Open-Air Layout Frame */}
      <div className="relative z-10 mx-auto flex min-h-screen w-full flex-col px-4 py-5 sm:px-8 lg:px-14">
        {/* Top Peripheral Pre-Header Telemetry */}
        <div className="mb-4 flex flex-wrap items-center justify-between gap-2 border-b border-[rgba(15,20,25,0.08)] pb-2 text-[11px] font-sans font-medium tracking-wider text-[var(--color-ink-muted)] uppercase">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-[var(--color-ink-primary)]">▪ SYS // ACTIVE</span>
            <span>·</span>
            <span>DATA // SQLITE_WAL</span>
            <span>·</span>
            <span className="hidden sm:inline">SPEC // CLINICAL_HORIZON</span>
          </div>
          <div className="flex items-center gap-3">
            <span className="tabular-nums font-medium">{utcTime || 'LIVE_FEED'}</span>
            <span className="font-semibold text-[#0088cc]">STAGE // READY</span>
          </div>
        </div>

        {/* Monumental Header & Asymmetric Primary Control Spine */}
        <header className="mb-7">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            {/* Monumental Ultralight Title in Plus Jakarta Sans */}
            <div>
              <div className="flex items-center gap-3">
                <span className="font-sans text-xs font-bold tracking-widest text-[#0088cc] uppercase">
                  [ JT-01 ]
                </span>
                <span className="text-[11px] font-sans font-medium tracking-wider text-slate-400 uppercase">
                  LOCAL PERSISTENT STORE
                </span>
              </div>
              <h1 className="mt-1 font-display text-4xl sm:text-5xl lg:text-6xl font-extralight tracking-[0.24em] text-[var(--color-ink-primary)] uppercase select-none">
                Job Tracker
              </h1>
              <p className="mt-1 font-sans text-xs font-medium tracking-wider text-[var(--color-ink-secondary)] uppercase">
                Application Pipeline &amp; Telemetry Matrix
              </p>
            </div>

            {/* Viewport Action Spine & Unboxed Navigation Tabs */}
            <div className="flex flex-wrap items-center gap-3 pt-2 lg:pt-0">
              {/* Unboxed View Switcher */}
              <nav
                className="flex items-center border border-[rgba(15,20,25,0.14)] bg-white/60 p-0.5 backdrop-blur-xs"
                aria-label="Application views"
              >
                <button
                  type="button"
                  id="tab-applications"
                  onClick={() => setActiveTab('table')}
                  aria-pressed={activeTab === 'table'}
                  className={`btn-clinical ${
                    activeTab === 'table'
                      ? 'bg-[var(--color-ink-slate)] text-white'
                      : 'text-slate-600 hover:text-[var(--color-ink-primary)]'
                  }`}
                  style={
                    activeTab === 'table'
                      ? { clipPath: 'polygon(0 0, calc(100% - 6px) 0, 100% 6px, 100% 100%, 0 100%)' }
                      : undefined
                  }
                >
                  <LayoutGrid className="size-3.5" />
                  <span>Applications</span>
                </button>
                <button
                  type="button"
                  id="tab-pipeline-graph"
                  onClick={() => setActiveTab('graph')}
                  aria-pressed={activeTab === 'graph'}
                  className={`btn-clinical ${
                    activeTab === 'graph'
                      ? 'bg-[var(--color-ink-slate)] text-white'
                      : 'text-slate-600 hover:text-[var(--color-ink-primary)]'
                  }`}
                  style={
                    activeTab === 'graph'
                      ? { clipPath: 'polygon(0 0, calc(100% - 6px) 0, 100% 6px, 100% 100%, 0 100%)' }
                      : undefined
                  }
                >
                  <GitFork className="size-3.5" />
                  <span>Pipeline Flow</span>
                </button>
              </nav>

              {/* Primary Action Button */}
              <button
                type="button"
                id="header-add-application"
                onClick={() => setFormMode({ kind: 'create' })}
                className="btn-clinical btn-clinical-primary"
              >
                <Plus className="size-3.5" />
                <span>New Application</span>
              </button>
            </div>
          </div>

          {/* Tapered Datum Rule */}
          <div className="tapered-datum-rule" />
        </header>

        {/* Content Projection Area */}
        <main className="flex-1">
          {activeTab === 'table' ? (
            <ApplicationList
              refreshKey={refreshKey}
              onChanged={triggerRefresh}
              onEdit={(app: Application) => setFormMode({ kind: 'edit', app })}
              onCreate={() => setFormMode({ kind: 'create' })}
              notify={notify}
            />
          ) : (
            <PipelineGraph
              refreshKey={refreshKey}
              onCreate={() => setFormMode({ kind: 'create' })}
            />
          )}
        </main>

        {/* Peripheral Telemetry Footer */}
        <footer className="mt-14 border-t border-[rgba(15,20,25,0.08)] pt-4 pb-6 text-xs text-[var(--color-ink-muted)]">
          <div className="flex flex-col items-start justify-between gap-2 font-sans text-[11px] font-medium tracking-wider sm:flex-row sm:items-center uppercase">
            <div className="flex items-center gap-2">
              <span className="font-semibold text-[var(--color-ink-primary)]">▪ ARCHIVAL REPO</span>
              <span>·</span>
              <span>LOCAL SQLITE PERSISTENCE</span>
              <span>·</span>
              <span>WAL MODE ACTIVE</span>
            </div>
            <div className="flex items-center gap-3">
              <span>ZERO EXTERNAL CLOUD TELEMETRY</span>
              <span>·</span>
              <span className="text-[#0088cc] font-semibold">ACYCLIC SANKEY ANALYTICS</span>
            </div>
          </div>
        </footer>
      </div>

      {/* Modal Drawer for Record Entry / Edit */}
      {formMode && (
        <ApplicationForm
          mode={formMode}
          onClose={() => setFormMode(null)}
          onSaved={handleSaved}
        />
      )}

      {/* Floating Clinical Toast Telemetry */}
      <div
        className="pointer-events-none fixed right-4 bottom-4 z-50 flex max-w-sm flex-col gap-2 sm:right-8 sm:bottom-8"
        aria-live="polite"
      >
        {toasts.map((toast) => (
          <div
            key={toast.id}
            className={`animate-pop-in pointer-events-auto flex items-center justify-between gap-3 border bg-white/95 p-3 shadow-lg backdrop-blur-md ${
              toast.kind === 'error'
                ? 'border-rose-400 text-rose-900'
                : 'border-slate-300 text-slate-800'
            }`}
            style={{ clipPath: 'polygon(0 0, calc(100% - 6px) 0, 100% 6px, 100% 100%, 0 100%)' }}
          >
            <div className="flex items-center gap-2 font-sans text-xs">
              {toast.kind === 'error' ? (
                <AlertCircle className="size-4 shrink-0 text-rose-600" />
              ) : (
                <CheckCircle2 className="size-4 shrink-0 text-[#0088cc]" />
              )}
              <span className="font-medium">{toast.message}</span>
            </div>
            <button
              type="button"
              onClick={() => removeToast(toast.id)}
              className="cursor-pointer text-slate-400 hover:text-slate-800"
              aria-label="Dismiss notification"
            >
              <X className="size-3.5" />
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
