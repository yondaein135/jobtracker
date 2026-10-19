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
    color: '#94a3b8',
    badge: 'bg-slate-400/10 text-slate-300 ring-slate-400/25',
    dot: 'bg-slate-400',
    terminal: false,
  },
  APPLIED: {
    label: 'Applied',
    description: 'Application sent',
    color: '#60a5fa',
    badge: 'bg-sky-400/10 text-sky-300 ring-sky-400/25',
    dot: 'bg-sky-400',
    terminal: false,
  },
  RESPONSE_RECEIVED: {
    label: 'Response',
    description: 'Recruiter screen / initial outreach',
    color: '#a78bfa',
    badge: 'bg-violet-400/10 text-violet-300 ring-violet-400/25',
    dot: 'bg-violet-400',
    terminal: false,
  },
  INTERVIEWING: {
    label: 'Interviewing',
    description: 'Technical / behavioral rounds',
    color: '#fbbf24',
    badge: 'bg-amber-400/10 text-amber-300 ring-amber-400/25',
    dot: 'bg-amber-400',
    terminal: false,
  },
  OFFER: {
    label: 'Offer',
    description: 'Offer extended',
    color: '#34d399',
    badge: 'bg-emerald-400/10 text-emerald-300 ring-emerald-400/25',
    dot: 'bg-emerald-400',
    terminal: false,
  },
  ACCEPTED: {
    label: 'Accepted',
    description: 'Offer signed',
    color: '#2dd4bf',
    badge: 'bg-teal-400/15 text-teal-200 ring-teal-300/40',
    dot: 'bg-teal-300',
    terminal: true,
  },
  REJECTED: {
    label: 'Rejected',
    description: 'Rejected at any stage',
    color: '#fb7185',
    badge: 'bg-rose-400/10 text-rose-300 ring-rose-400/25',
    dot: 'bg-rose-400',
    terminal: true,
  },
  GHOSTED: {
    label: 'Ghosted',
    description: 'No response after threshold',
    color: '#71717a',
    badge: 'bg-zinc-400/10 text-zinc-400 ring-zinc-400/20',
    dot: 'bg-zinc-500',
    terminal: true,
  },
  WITHDRAWN: {
    label: 'Withdrawn',
    description: 'Candidate pulled out',
    color: '#fb923c',
    badge: 'bg-orange-400/10 text-orange-300 ring-orange-400/25',
    dot: 'bg-orange-400',
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
