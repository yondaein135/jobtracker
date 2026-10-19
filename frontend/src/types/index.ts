// Canonical status values — must stay in sync with backend/internal/models.
export const STATUSES = [
  'SAVED',
  'APPLIED',
  'RESPONSE_RECEIVED',
  'INTERVIEWING',
  'OFFER',
  'ACCEPTED',
  'REJECTED',
  'GHOSTED',
  'WITHDRAWN',
] as const

export type Status = (typeof STATUSES)[number]

export const INITIAL_STATUSES = ['APPLIED', 'SAVED'] as const satisfies readonly Status[]
export type InitialStatus = (typeof INITIAL_STATUSES)[number]

/** Statuses that represent an in-flight application. */
export const ACTIVE_STATUSES: readonly Status[] = ['APPLIED', 'RESPONSE_RECEIVED', 'INTERVIEWING', 'OFFER']

export interface StatusMeta {
  label: string
  description: string
  /** Hex colour used by charts. */
  color: string
  /** Tailwind classes for badges (kept as literals so Tailwind can detect them). */
  badge: string
  dot: string
  terminal: boolean
}

export const STATUS_META: Record<Status, StatusMeta> = {
  SAVED: {
    label: 'Saved',
    description: 'Bookmarked, not yet applied',
    color: '#64748b',
    badge: 'bg-slate-100 text-slate-700 border-slate-300/80',
    dot: 'bg-slate-400',
    terminal: false,
  },
  APPLIED: {
    label: 'Applied',
    description: 'Application sent',
    color: '#0088cc',
    badge: 'bg-sky-50 text-sky-800 border-sky-300/80',
    dot: 'bg-[#0088cc]',
    terminal: false,
  },
  RESPONSE_RECEIVED: {
    label: 'Response',
    description: 'Recruiter screen / initial outreach',
    color: '#4f46e5',
    badge: 'bg-indigo-50 text-indigo-800 border-indigo-300/80',
    dot: 'bg-indigo-600',
    terminal: false,
  },
  INTERVIEWING: {
    label: 'Interviewing',
    description: 'Technical / behavioral rounds',
    color: '#d97706',
    badge: 'bg-amber-50 text-amber-800 border-amber-300/80',
    dot: 'bg-amber-600',
    terminal: false,
  },
  OFFER: {
    label: 'Offer',
    description: 'Offer extended',
    color: '#059669',
    badge: 'bg-emerald-50 text-emerald-800 border-emerald-300/80',
    dot: 'bg-emerald-600',
    terminal: false,
  },
  ACCEPTED: {
    label: 'Accepted',
    description: 'Offer signed',
    color: '#0d9488',
    badge: 'bg-teal-50 text-teal-800 border-teal-300/80',
    dot: 'bg-teal-600',
    terminal: true,
  },
  REJECTED: {
    label: 'Rejected',
    description: 'Rejected at any stage',
    color: '#dc2626',
    badge: 'bg-rose-50 text-rose-800 border-rose-300/80',
    dot: 'bg-rose-600',
    terminal: true,
  },
  GHOSTED: {
    label: 'Ghosted',
    description: 'No response after threshold',
    color: '#71717a',
    badge: 'bg-zinc-100 text-zinc-700 border-zinc-300/80',
    dot: 'bg-zinc-500',
    terminal: true,
  },
  WITHDRAWN: {
    label: 'Withdrawn',
    description: 'Candidate pulled out',
    color: '#ea580c',
    badge: 'bg-orange-50 text-orange-800 border-orange-300/80',
    dot: 'bg-orange-600',
    terminal: true,
  },
}

export interface Application {
  id: number
  company_name: string
  role_title: string
  location: string
  job_url: string
  salary_range: string
  notes: string
  job_description: string
  created_at: string
  current_status: Status
  status_updated_at: string
}

export interface StatusHistoryEntry {
  id: number
  application_id: number
  status: Status
  notes: string
  changed_at: string
}

export interface ApplicationDetail extends Application {
  history: StatusHistoryEntry[]
}

export interface CreateApplicationInput {
  company_name: string
  role_title: string
  initial_status: InitialStatus
  location?: string
  salary_range?: string
  job_url?: string
  notes?: string
  job_description?: string
}

export type UpdateApplicationInput = Partial<
  Pick<Application, 'company_name' | 'role_title' | 'location' | 'job_url' | 'salary_range' | 'notes' | 'job_description'>
>

export interface StatusChangeInput {
  status: Status
  notes?: string
  /** Optional ISO-8601 timestamp; defaults to now on the server. */
  changed_at?: string
}

export interface ListFilters {
  status?: Status | ''
  search?: string
}

export interface SankeyNode {
  name: Status
  value?: number
}

export interface SankeyLink {
  source: Status
  target: Status
  value: number
}

export interface GraphAnalytics {
  total_applied: number
  total_applications: number
  responded_count: number
  offer_count: number
  current_distribution: Partial<Record<Status, number>>
  reached_counts: Partial<Record<Status, number>>
  sankey_nodes: SankeyNode[]
  sankey_links: SankeyLink[]
  omitted_backward_links: number
}
