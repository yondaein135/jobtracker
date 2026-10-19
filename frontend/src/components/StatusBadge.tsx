import { Check, ChevronDown, Loader2 } from 'lucide-react'
import { useCallback, useEffect, useLayoutEffect, useRef, useState, type KeyboardEvent as ReactKeyboardEvent } from 'react'
import { createPortal } from 'react-dom'
import { STATUSES, STATUS_META, type Status } from '../types'

interface StatusBadgeProps {
  status: Status
  size?: 'sm' | 'md'
  className?: string
}

/** Coloured pill showing a status label. */
export default function StatusBadge({ status, size = 'md', className = '' }: StatusBadgeProps) {
  const meta = STATUS_META[status]
  const sizing = size === 'sm' ? 'px-2 py-0.5 text-[11px]' : 'px-2.5 py-1 text-xs'
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full font-medium whitespace-nowrap ring-1 ring-inset ${meta.badge} ${sizing} ${className}`}
      title={meta.description}
    >
      <span className={`size-1.5 rounded-full ${meta.dot} ${status === 'INTERVIEWING' ? 'animate-pulse' : ''}`} />
      {meta.label}
    </span>
  )
}

interface StatusSelectProps {
  value: Status
  onChange: (status: Status) => void
  busy?: boolean
  disabled?: boolean
  id?: string
}

const POPOVER_WIDTH = 248
const POPOVER_MAX_HEIGHT = 380

/**
 * Inline status picker. Renders the menu in a portal with fixed positioning so
 * it is never clipped by scrollable table containers.
 */
export function StatusSelect({ value, onChange, busy, disabled, id }: StatusSelectProps) {
  const [open, setOpen] = useState(false)
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  const close = useCallback(() => setOpen(false), [])

  useLayoutEffect(() => {
    if (!open || !buttonRef.current) return
    const r = buttonRef.current.getBoundingClientRect()
    const spaceBelow = window.innerHeight - r.bottom
    const top = spaceBelow < POPOVER_MAX_HEIGHT + 12 && r.top > spaceBelow ? r.top - 6 - POPOVER_MAX_HEIGHT : r.bottom + 6
    const left = Math.min(Math.max(8, r.left), window.innerWidth - POPOVER_WIDTH - 8)
    setPos({ top: Math.max(8, top), left })
  }, [open])

  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      const t = e.target as Node
      if (!menuRef.current?.contains(t) && !buttonRef.current?.contains(t)) close()
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        close()
        buttonRef.current?.focus()
      }
    }
    const onScroll = (e: Event) => {
      if (!menuRef.current?.contains(e.target as Node)) close()
    }
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    window.addEventListener('scroll', onScroll, true)
    window.addEventListener('resize', close)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
      window.removeEventListener('scroll', onScroll, true)
      window.removeEventListener('resize', close)
    }
  }, [open, close])

  // Focus the selected option when the menu opens.
  useEffect(() => {
    if (open && pos) {
      menuRef.current?.querySelector<HTMLButtonElement>('[aria-selected="true"]')?.focus()
    }
  }, [open, pos])

  const onMenuKeyDown = (e: ReactKeyboardEvent) => {
    if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return
    e.preventDefault()
    const items = Array.from(menuRef.current?.querySelectorAll<HTMLButtonElement>('[role="option"]') ?? [])
    const idx = items.indexOf(document.activeElement as HTMLButtonElement)
    const next = e.key === 'ArrowDown' ? (idx + 1) % items.length : (idx - 1 + items.length) % items.length
    items[next]?.focus()
  }

  const pick = (s: Status) => {
    close()
    buttonRef.current?.focus()
    if (s !== value) onChange(s)
  }

  return (
    <>
      <button
        ref={buttonRef}
        id={id}
        type="button"
        disabled={disabled || busy}
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="listbox"
        aria-expanded={open}
        className="group inline-flex cursor-pointer items-center gap-1 rounded-full transition-transform duration-150 hover:scale-[1.03] focus-visible:ring-2 focus-visible:ring-accent-400/60 focus-visible:outline-none disabled:cursor-wait"
      >
        <StatusBadge status={value} />
        {busy ? (
          <Loader2 className="size-3.5 animate-spin text-slate-400" />
        ) : (
          <ChevronDown
            className={`size-3.5 text-slate-500 transition-transform duration-200 group-hover:text-slate-300 ${open ? 'rotate-180' : ''}`}
          />
        )}
      </button>

      {open &&
        pos &&
        createPortal(
          <div
            ref={menuRef}
            role="listbox"
            aria-label="Change status"
            onKeyDown={onMenuKeyDown}
            style={{ top: pos.top, left: pos.left, width: POPOVER_WIDTH, maxHeight: POPOVER_MAX_HEIGHT }}
            className="animate-pop-in fixed z-50 overflow-y-auto rounded-xl border border-white/10 bg-ink-800/95 p-1.5 shadow-2xl shadow-black/60 backdrop-blur-xl"
          >
            <div className="px-2.5 pt-1.5 pb-2 text-[10px] font-semibold tracking-widest text-slate-500 uppercase">
              Move to stage
            </div>
            {STATUSES.map((s) => {
              const meta = STATUS_META[s]
              const selected = s === value
              return (
                <button
                  key={s}
                  type="button"
                  role="option"
                  aria-selected={selected}
                  onClick={() => pick(s)}
                  className={`flex w-full cursor-pointer items-center gap-3 rounded-lg px-2.5 py-2 text-left transition-colors outline-none hover:bg-white/[0.06] focus:bg-white/[0.08] ${selected ? 'bg-white/[0.05]' : ''}`}
                >
                  <span className={`size-2 shrink-0 rounded-full ${meta.dot}`} />
                  <span className="min-w-0 flex-1">
                    <span className="block text-sm font-medium text-slate-100">{meta.label}</span>
                    <span className="block truncate text-[11px] text-slate-500">{meta.description}</span>
                  </span>
                  {selected && <Check className="size-4 shrink-0 text-accent-400" />}
                </button>
              )
            })}
          </div>,
          document.body,
        )}
    </>
  )
}
