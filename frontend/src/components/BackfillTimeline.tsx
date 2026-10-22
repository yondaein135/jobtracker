import { Plus, Trash2 } from 'lucide-react'
import { useId } from 'react'
import { toLocalDateInputValue } from '../lib/format'
import { newStep, optionsFor, timelineErrors, type TimelineStep } from '../lib/timeline'
import { STATUS_META, getNextAllowedStatuses, isTerminalStatus, type Status } from '../types'

interface BackfillTimelineProps {
  steps: TimelineStep[]
  onChange: (steps: TimelineStep[]) => void
  showErrors: boolean
}

/** Editor for the status history of an application that already happened. */
export default function BackfillTimeline({ steps, onChange, showErrors }: BackfillTimelineProps) {
  const uid = useId()
  const errors = timelineErrors(steps)
  const today = toLocalDateInputValue()
  const last = steps[steps.length - 1]
  const canAdd = !isTerminalStatus(last.status)

  const update = (idx: number, patch: Partial<TimelineStep>) => {
    const next = steps.map((s, i) => (i === idx ? { ...s, ...patch } : s))
    // A status change can make later steps unreachable; drop everything from the first broken link.
    const broken = next.findIndex((s, i) => i > 0 && !optionsFor(next, i).includes(s.status))
    onChange(broken === -1 ? next : next.slice(0, broken))
  }

  const add = () => {
    const options = getNextAllowedStatuses(last.status)
    if (options.length) onChange([...steps, newStep(options[0], last.date)])
  }

  const removeLast = () => onChange(steps.slice(0, -1))

  return (
    <div>
      <span className="field-label">Status history</span>
      <p className="mb-2 text-[11px] text-slate-500">
        Add each step in the order it happened. Only forward moves are allowed.
      </p>
      <ol className="space-y-2">
        {steps.map((step, idx) => {
          const err = showErrors ? errors[idx] : ''
          const isLast = idx === steps.length - 1
          return (
            <li key={step.key} className="border border-slate-200 bg-slate-50/60 p-2.5">
              <div className="flex items-start gap-2">
                <span
                  className={`mt-2.5 size-2 shrink-0 ${STATUS_META[step.status].dot}`}
                  style={{ borderRadius: '1px' }}
                  aria-hidden
                />
                <div className="grid min-w-0 flex-1 gap-2 sm:grid-cols-2">
                  <div>
                    <label htmlFor={`${uid}-status-${step.key}`} className="sr-only">
                      Step {idx + 1} status
                    </label>
                    <select
                      id={`${uid}-status-${step.key}`}
                      className="field-clinical font-sans text-xs uppercase"
                      value={step.status}
                      onChange={(e) => update(idx, { status: e.target.value as Status })}
                    >
                      {optionsFor(steps, idx).map((s) => (
                        <option key={s} value={s}>
                          {STATUS_META[s].label}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label htmlFor={`${uid}-date-${step.key}`} className="sr-only">
                      Step {idx + 1} date
                    </label>
                    <input
                      id={`${uid}-date-${step.key}`}
                      type="date"
                      className="field-clinical font-sans text-xs"
                      value={step.date}
                      min={idx > 0 ? steps[idx - 1].date || undefined : undefined}
                      max={today}
                      onChange={(e) => update(idx, { date: e.target.value })}
                      aria-invalid={Boolean(err)}
                    />
                    {err && <p className="mt-1 font-sans text-[11px] text-rose-600">{err}</p>}
                  </div>
                  <div className="sm:col-span-2">
                    <label htmlFor={`${uid}-notes-${step.key}`} className="sr-only">
                      Step {idx + 1} note
                    </label>
                    <input
                      id={`${uid}-notes-${step.key}`}
                      className="field-clinical"
                      placeholder="Note (optional)"
                      value={step.notes}
                      onChange={(e) => update(idx, { notes: e.target.value })}
                      maxLength={20000}
                    />
                  </div>
                </div>
                {isLast && steps.length > 1 ? (
                  <button
                    type="button"
                    onClick={removeLast}
                    className="btn-clinical-icon mt-1 shrink-0 hover:text-rose-600"
                    aria-label={`Remove step ${idx + 1}`}
                  >
                    <Trash2 className="size-3.5" />
                  </button>
                ) : (
                  <span className="size-7 shrink-0" aria-hidden />
                )}
              </div>
            </li>
          )
        })}
      </ol>
      <button
        type="button"
        onClick={add}
        disabled={!canAdd}
        className="btn-clinical btn-clinical-ghost mt-2 w-full"
        id="backfill-add-step"
      >
        <Plus className="size-3.5" />
        <span>{canAdd ? 'Add next step' : `${STATUS_META[last.status].label} is final`}</span>
      </button>
    </div>
  )
}
