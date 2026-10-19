import type {
  Application,
  ApplicationDetail,
  CreateApplicationInput,
  GraphAnalytics,
  ListFilters,
  StatusChangeInput,
  UpdateApplicationInput,
} from '../types'

const BASE = (import.meta.env.VITE_API_BASE as string | undefined) ?? ''

export class ApiError extends Error {
  readonly status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body !== undefined) headers.set('Content-Type', 'application/json')
  headers.set('Accept', 'application/json')

  let res: Response
  try {
    res = await fetch(`${BASE}/api${path}`, { ...init, headers })
  } catch {
    throw new ApiError(0, 'Cannot reach the server. Is the backend running?')
  }

  if (res.status === 204) return undefined as T

  const text = await res.text()
  let body: unknown = undefined
  if (text) {
    try {
      body = JSON.parse(text)
    } catch {
      body = text
    }
  }

  if (!res.ok) {
    const msg =
      body && typeof body === 'object' && 'error' in body
        ? String((body as { error: unknown }).error)
        : `Request failed (${res.status})`
    throw new ApiError(res.status, msg)
  }
  return body as T
}

const json = (data: unknown) => JSON.stringify(data)

export const api = {
  listApplications(filters: ListFilters = {}, signal?: AbortSignal) {
    const qs = new URLSearchParams()
    if (filters.status) qs.set('status', filters.status)
    if (filters.search?.trim()) qs.set('search', filters.search.trim())
    const q = qs.toString()
    return request<Application[]>(`/applications${q ? `?${q}` : ''}`, { signal })
  },

  getApplication(id: number, signal?: AbortSignal) {
    return request<ApplicationDetail>(`/applications/${id}`, { signal })
  },

  createApplication(input: CreateApplicationInput) {
    return request<ApplicationDetail>('/applications', { method: 'POST', body: json(input) })
  },

  updateApplication(id: number, input: UpdateApplicationInput) {
    return request<ApplicationDetail>(`/applications/${id}`, { method: 'PATCH', body: json(input) })
  },

  addStatus(id: number, input: StatusChangeInput) {
    return request<ApplicationDetail>(`/applications/${id}/status`, { method: 'POST', body: json(input) })
  },

  deleteApplication(id: number) {
    return request<void>(`/applications/${id}`, { method: 'DELETE' })
  },

  getGraph(signal?: AbortSignal) {
    return request<GraphAnalytics>('/analytics/graph', { signal })
  },
}
