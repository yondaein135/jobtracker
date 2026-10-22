import { AlertCircle, CheckCircle2, GitFork, History, LayoutGrid, Plus, X } from 'lucide-react'
import { useCallback, useLayoutEffect, useRef, useState } from 'react'
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
  const tabRefs = useRef<Record<ActiveTab, HTMLButtonElement | null>>({ table: null, graph: null })
  const [indicator, setIndicator] = useState({ x: 0, w: 0 })

  // Measure the active tab so the indicator can slide to it.
  useLayoutEffect(() => {
    const measure = () => {
      const el = tabRefs.current[activeTab]
      if (el) setIndicator({ x: el.offsetLeft, w: el.offsetWidth })
    }
    measure()
    window.addEventListener('resize', measure)
    document.fonts?.ready.then(measure)
    return () => window.removeEventListener('resize', measure)
  }, [activeTab])

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
        ? `Added ${app.company_name} (${app.role_title})`
        : `Updated ${app.company_name}`,
      'success',
    )
  }

  return (
    <div className="relative min-h-screen bg-[var(--color-canvas-base)] font-sans text-[var(--color-ink-primary)]">
      {/* Clinical Horizon Atmospheric Background Mesh & Crosshairs */}
      <AmbientCanvas />

      {/* Main Open-Air Layout Frame */}
      <div className="relative z-10 mx-auto flex min-h-screen w-full flex-col px-4 py-5 sm:px-8 lg:px-14">
        {/* Monumental Header & Asymmetric Primary Control Spine */}
        <header className="mb-7 reveal">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            {/* Monumental Ultralight Title in Plus Jakarta Sans */}
            <div>
              <h1 className="mt-1 font-display text-4xl sm:text-5xl lg:text-6xl font-extralight tracking-[0.24em] text-[var(--color-ink-primary)] uppercase select-none">
                Job Tracker
              </h1>
            </div>

            {/* Viewport Action Spine & Unboxed Navigation Tabs */}
            <div className="flex flex-wrap items-center gap-3 pt-2 lg:pt-0">
              {/* Unboxed View Switcher with sliding indicator */}
              <nav
                className="relative flex items-center border border-[rgba(15,20,25,0.14)] bg-white/60 p-0.5 backdrop-blur-xs"
                aria-label="Application views"
              >
                <span
                  className="tab-indicator"
                  style={{ width: indicator.w, transform: `translateX(${indicator.x}px)` }}
                  aria-hidden
                />
                <button
                  type="button"
                  id="tab-applications"
                  ref={(el) => {
                    tabRefs.current.table = el
                  }}
                  onClick={() => setActiveTab('table')}
                  aria-pressed={activeTab === 'table'}
                  className={`btn-clinical relative z-10 ${
                    activeTab === 'table' ? 'text-white' : 'text-slate-600 hover:text-[var(--color-ink-primary)]'
                  }`}
                >
                  <LayoutGrid className="size-3.5" />
                  <span>Applications</span>
                </button>
                <button
                  type="button"
                  id="tab-pipeline-graph"
                  ref={(el) => {
                    tabRefs.current.graph = el
                  }}
                  onClick={() => setActiveTab('graph')}
                  aria-pressed={activeTab === 'graph'}
                  className={`btn-clinical relative z-10 ${
                    activeTab === 'graph' ? 'text-white' : 'text-slate-600 hover:text-[var(--color-ink-primary)]'
                  }`}
                >
                  <GitFork className="size-3.5" />
                  <span>Pipeline</span>
                </button>
              </nav>

              {/* Secondary action: enter an application that already happened */}
              <button
                type="button"
                id="header-backfill-application"
                onClick={() => setFormMode({ kind: 'backfill' })}
                className="btn-clinical btn-clinical-ghost bg-white/60 backdrop-blur-xs"
              >
                <History className="size-3.5" />
                <span>Fill past application</span>
              </button>

              {/* Primary Action Button */}
              <button
                type="button"
                id="header-add-application"
                onClick={() => setFormMode({ kind: 'create' })}
                className="btn-clinical btn-clinical-primary"
              >
                <Plus className="size-3.5" />
                <span>Add application</span>
              </button>
            </div>
          </div>

          {/* Tapered Datum Rule */}
          <div className="tapered-datum-rule" />
        </header>

        {/* Content Projection Area — keyed so each tab switch replays the entry animation */}
        <main className="flex-1">
          <div key={activeTab} className="view-enter">
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
          </div>
        </main>
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
            className={`toast-enter pointer-events-auto flex items-center justify-between gap-3 border bg-white/95 p-3 shadow-lg backdrop-blur-md ${
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
