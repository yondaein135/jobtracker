import {
  INITIAL_STATUSES,
  getNextAllowedStatuses,
  type BackfillEventInput,
  type Status,
} from '../types'
import { toLocalDateInputValue } from './format'

export interface TimelineStep {
  key: number
  status: Status
  /** Local calendar date, YYYY-MM-DD. */
  date: string
  notes: string
}

let nextKey = 1
export const newStep = (status: Status, date = '', notes = ''): TimelineStep => ({ key: nextKey++, status, date, notes })

export const initialTimeline = (): TimelineStep[] => [newStep('APPLIED')]

/** Statuses a step may take, given the step before it. */
export function optionsFor(steps: TimelineStep[], idx: number): readonly Status[] {
  return idx === 0 ? INITIAL_STATUSES : getNextAllowedStatuses(steps[idx - 1].status)
}

/** Per-step error messages ('' when the step is fine). Mirrors the backend rules. */
export function timelineErrors(steps: TimelineStep[]): string[] {
  const today = toLocalDateInputValue()
  return steps.map((s, i) => {
    if (!s.date) return 'Date is required'
    if (s.date > today) return 'Cannot be in the future'
    if (i > 0 && steps[i - 1].date && s.date < steps[i - 1].date) return 'Earlier than the previous step'
    return ''
  })
}

/** Converts steps to API events. Dates are sent as local midnight so they never land in the future. */
export function toBackfillEvents(steps: TimelineStep[]): BackfillEventInput[] {
  return steps.map((s) => ({
    status: s.status,
    changed_at: new Date(`${s.date}T00:00`).toISOString(),
    notes: s.notes.trim() || undefined,
  }))
}
