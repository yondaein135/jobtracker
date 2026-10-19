import { AlertCircle, CheckCircle2, GitFork, LayoutGrid, Plus, Sparkles, X } from 'lucide-react'
import { useCallback, useState } from 'react'
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

  const notify = useCallback((message: string, kind: 'success' | 'error' = 'success') => {
    const id = Date.now() + Math.random()
    setToasts((prev) => [...prev, { id, message, kind }])
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id))
    }, 4000)
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
    <div className="relative min-h-screen bg-ink-950 font-sans text-slate-200 selection:bg-indigo-500/30 selection:text-white">
      {/* Background ambient lighting glows */}
      <div className="pointer-events-none fixed inset-0 overflow-hidden" aria-hidden="true">
        <div className="animate-drift absolute -top-40 left-1/4 h-[500px] w-[500px] rounded-full bg-indigo-600/15 blur-[120px]" />
        <div className="animate-drift absolute top-1/3 -right-20 h-[450px] w-[450px] rounded-full bg-fuchsia-600/10 blur-[140px] [animation-delay:4s]" />
        <div className="animate-drift absolute -bottom-20 left-1/3 h-[600px] w-[600px] rounded-full bg-violet-600/10 blur-[150px] [animation-delay:8s]" />
      </div>

      {/* Main Container */}
      <div className="relative mx-auto flex min-h-screen max-w-7xl flex-col px-4 py-6 sm:px-6 lg:px-8">
        {/* Navigation Bar */}
        <header className="glass mb-8 flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between sm:p-5">
          {/* Logo & Title */}
          <div className="flex items-center gap-3.5">
            <div className="relative flex size-11 items-center justify-center rounded-xl bg-linear-to-tr from-indigo-500 via-violet-500 to-fuchsia-500 p-0.5 shadow-lg shadow-indigo-500/20 ring-1 ring-white/20">
              <div className="flex size-full items-center justify-center rounded-[10px] bg-ink-950/80 backdrop-blur-xs">
                <Sparkles className="size-5 text-indigo-300" />
              </div>
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="font-display text-xl font-bold tracking-tight text-white">
                  Job Tracker
                </h1>
                <span className="rounded-full border border-indigo-500/30 bg-indigo-500/10 px-2 py-0.5 text-[10px] font-semibold tracking-wider text-indigo-300 uppercase">
                  Local
                </span>
              </div>
              <p className="text-xs text-slate-400">
                Visual pipeline & application lifecycle log
              </p>
            </div>
          </div>

          {/* Navigation Tabs & Actions */}
          <div className="flex flex-wrap items-center gap-3">
            {/* View Switcher Tabs */}
            <nav className="flex rounded-xl border border-white/10 bg-ink-900/60 p-1" aria-label="Views">
              <button
                type="button"
                id="tab-applications"
                onClick={() => setActiveTab('table')}
                aria-pressed={activeTab === 'table'}
                className={`flex cursor-pointer items-center gap-2 rounded-lg px-3.5 py-1.5 text-xs font-medium transition-all duration-200 select-none ${
                  activeTab === 'table'
                    ? 'bg-linear-to-r from-indigo-500/20 to-violet-500/20 text-white shadow-xs ring-1 ring-violet-400/40'
                    : 'text-slate-400 hover:bg-white/[0.04] hover:text-slate-200'
                }`}
              >
                <LayoutGrid className="size-3.5" />
                Applications
              </button>
              <button
                type="button"
                id="tab-pipeline-graph"
                onClick={() => setActiveTab('graph')}
                aria-pressed={activeTab === 'graph'}
                className={`flex cursor-pointer items-center gap-2 rounded-lg px-3.5 py-1.5 text-xs font-medium transition-all duration-200 select-none ${
                  activeTab === 'graph'
                    ? 'bg-linear-to-r from-indigo-500/20 to-violet-500/20 text-white shadow-xs ring-1 ring-violet-400/40'
                    : 'text-slate-400 hover:bg-white/[0.04] hover:text-slate-200'
                }`}
              >
                <GitFork className="size-3.5" />
                Pipeline Flow
              </button>
            </nav>

            {/* Quick Add Button */}
            <button
              type="button"
              id="header-add-application"
              onClick={() => setFormMode({ kind: 'create' })}
              className="btn btn-primary cursor-pointer text-xs sm:text-sm"
            >
              <Plus className="size-4" />
              <span>New Application</span>
            </button>
          </div>
        </header>

        {/* Content Area */}
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

        {/* Footer */}
        <footer className="mt-12 flex flex-col items-center justify-between gap-4 border-t border-white/[0.06] pt-6 pb-4 text-xs text-slate-500 sm:flex-row">
          <p>
            Local persistent SQLite storage &bull; Modern Go backend &amp; React frontend
          </p>
          <div className="flex items-center gap-4 text-slate-400">
            <span>WAL Mode Active</span>
            <span>&bull;</span>
            <span>Acyclic Sankey Analytics</span>
          </div>
        </footer>
      </div>

      {/* Quick Add / Edit Modal Drawer */}
      {formMode && (
        <ApplicationForm
          mode={formMode}
          onClose={() => setFormMode(null)}
          onSaved={handleSaved}
        />
      )}

      {/* Floating Toast Notifications */}
      <div
        className="pointer-events-none fixed right-4 bottom-4 z-50 flex max-w-sm flex-col gap-2.5 sm:right-6 sm:bottom-6"
        aria-live="polite"
      >
        {toasts.map((toast) => (
          <div
            key={toast.id}
            className={`animate-pop-in pointer-events-auto flex items-center justify-between gap-3 rounded-xl border px-4 py-3 shadow-2xl backdrop-blur-xl ${
              toast.kind === 'error'
                ? 'border-rose-500/30 bg-rose-950/80 text-rose-200'
                : 'border-violet-500/30 bg-ink-900/90 text-slate-100'
            }`}
          >
            <div className="flex items-center gap-2.5 text-xs font-medium">
              {toast.kind === 'error' ? (
                <AlertCircle className="size-4 shrink-0 text-rose-400" />
              ) : (
                <CheckCircle2 className="size-4 shrink-0 text-violet-400" />
              )}
              <span>{toast.message}</span>
            </div>
            <button
              type="button"
              onClick={() => removeToast(toast.id)}
              className="btn-icon size-6 text-slate-400 hover:text-white"
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
