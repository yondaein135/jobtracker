import { Check, ChevronDown, Loader2 } from 'lucide-react'
import { useCallback, useEffect, useLayoutEffect, useRef, useState, type KeyboardEvent as ReactKeyboardEvent } from 'react'
import { createPortal } from 'react-dom'
import { STATUSES, STATUS_META, type Status } from '../types'

interface StatusBadgeProps {
  status: Status
  size?: 'sm' | 'md'
  className?: string
}

/** Clinical horizon status badge with elegant sans-serif typography */
export default function StatusBadge({ status, size = 'md', className = '' }: StatusBadgeProps) {
  const meta = STATUS_META[status]
  const sizing = size === 'sm' ? 'px-2 py-0.5 text-[10px]' : 'px-2.5 py-1 text-xs'
  return (
    <span
      className={`inline-flex items-center gap-1.5 font-sans uppercase tracking-wider font-semibold border ${meta.badge} ${sizing} ${className}`}
      style={{ clipPath: 'polygon(0 0, calc(100% - 4px) 0, 100% 4px, 100% 100%, 0 100%)' }}
      title={meta.description}
    >
      <span
        className={`size-1.5 ${meta.dot} ${status === 'INTERVIEWING' ? 'animate-pulse' : ''}`}
        style={{ borderRadius: '1px' }}
      />
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

const POPOVER_WIDTH = 250
const POPOVER_MAX_HEIGHT = 380

/**
 * Inline clinical status selector with keyboard navigation and unboxed popover.
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
        className="group inline-flex cursor-pointer items-center gap-1.5 transition-all outline-none focus-visible:ring-1 focus-visible:ring-[#0088cc] disabled:cursor-wait"
      >
        <StatusBadge status={value} />
        {busy ? (
          <Loader2 className="size-3 animate-spin text-slate-500" />
        ) : (
          <ChevronDown
            className={`size-3 text-slate-400 transition-transform duration-150 group-hover:text-slate-700 ${
              open ? 'rotate-180' : ''
            }`}
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
            className="animate-pop-in fixed z-50 overflow-y-auto bg-white p-1 shadow-xl border border-slate-300/80 text-slate-800"
          >
            <div className="flex items-center justify-between border-b border-slate-200 px-2.5 py-1.5 text-[10px] font-sans font-semibold tracking-widest text-slate-400 uppercase">
              <span>Transition Stage</span>
              <span>[ ESC ]</span>
            </div>
            <div className="py-1">
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
                    className={`flex w-full cursor-pointer items-center gap-2.5 px-2.5 py-1.5 text-left transition-colors outline-none hover:bg-slate-100/80 focus:bg-slate-100 ${
                      selected ? 'bg-slate-100/90 font-medium' : ''
                    }`}
                  >
                    <span
                      className={`size-1.5 shrink-0 ${meta.dot}`}
                      style={{ borderRadius: '1px' }}
                    />
                    <span className="min-w-0 flex-1">
                      <span className="block text-xs font-sans font-semibold tracking-wide text-slate-800">
                        {meta.label}
                      </span>
                      <span className="block truncate text-[11px] font-sans text-slate-500">
                        {meta.description}
                      </span>
                    </span>
                    {selected && <Check className="size-3.5 shrink-0 text-[#0088cc]" />}
                  </button>
                )
              })}
            </div>
          </div>,
          document.body,
        )}
    </>
  )
}
