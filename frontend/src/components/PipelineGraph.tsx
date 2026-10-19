import type { EChartsOption } from 'echarts'
import { BarChart, SankeyChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import * as echarts from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import EChart from './EChart'
import { Activity, Info, Loader2, MessageSquareReply, PartyPopper, RefreshCw, Send, Workflow } from 'lucide-react'
import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { api } from '../api/client'
import { percent } from '../lib/format'
import { ACTIVE_STATUSES, STATUSES, STATUS_META, type GraphAnalytics, type Status } from '../types'

echarts.use([SankeyChart, BarChart, TooltipComponent, GridComponent, CanvasRenderer])

interface PipelineGraphProps {
  refreshKey: number
  onCreate: () => void
}

const FONT = 'Inter, ui-sans-serif, system-ui, sans-serif'

const tooltipBase = {
  backgroundColor: 'rgba(12, 14, 26, 0.96)',
  borderColor: 'rgba(255, 255, 255, 0.1)',
  borderWidth: 1,
  padding: [10, 14],
  textStyle: { color: '#e2e8f0', fontFamily: FONT, fontSize: 12 },
  extraCssText: 'border-radius: 12px; box-shadow: 0 12px 40px -8px rgba(0,0,0,0.7); backdrop-filter: blur(12px);',
}

const esc = (s: string) => s.replace(/[&<>"']/g, (c) => `&#${c.charCodeAt(0)};`)

export default function PipelineGraph({ refreshKey, onCreate }: PipelineGraphProps) {
  const [data, setData] = useState<GraphAnalytics | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [reload, setReload] = useState(0)

  useEffect(() => {
    const ctrl = new AbortController()
    setLoading(true)
    api
      .getGraph(ctrl.signal)
      .then((g) => {
        setData(g)
        setError(null)
      })
      .catch((err: unknown) => {
        if (!ctrl.signal.aborted) setError(err instanceof Error ? err.message : 'Failed to load analytics')
      })
      .finally(() => {
        if (!ctrl.signal.aborted) setLoading(false)
      })
    return () => ctrl.abort()
  }, [refreshKey, reload])

  if (error && !data) {
    return (
      <div className="glass flex flex-col items-center gap-3 px-6 py-20 text-center">
        <p className="text-rose-300">{error}</p>
        <button className="btn btn-ghost" onClick={() => setReload((r) => r + 1)} id="retry-graph">
          Retry
        </button>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="flex items-center justify-center py-32 text-slate-500">
        <Loader2 className="mr-2 size-5 animate-spin" /> Loading analytics…
      </div>
    )
  }

  if (data.total_applications === 0) {
    return (
      <div className="glass animate-slide-up flex flex-col items-center px-6 py-24 text-center">
        <div className="relative mb-5">
          <div className="absolute inset-0 rounded-2xl bg-fuchsia-500/40 blur-xl" />
          <div className="relative flex size-14 items-center justify-center rounded-2xl bg-linear-to-br from-indigo-500 to-fuchsia-500 text-white shadow-xl">
            <Workflow className="size-6" />
          </div>
        </div>
        <h2 className="font-display text-lg font-semibold text-white">No pipeline data yet</h2>
        <p className="mt-1.5 max-w-sm text-sm text-slate-400">
          Add applications and move them through stages — this view maps how every application flowed.
        </p>
        <button className="btn btn-primary mt-6" onClick={onCreate} id="graph-empty-add">
          Add application
        </button>
      </div>
    )
  }

  return <Dashboard data={data} loading={loading} onRefresh={() => setReload((r) => r + 1)} />
}

function Dashboard({ data, loading, onRefresh }: { data: GraphAnalytics; loading: boolean; onRefresh: () => void }) {
  const active = ACTIVE_STATUSES.reduce((sum, s) => sum + (data.current_distribution[s] ?? 0), 0)
  const awaiting = data.current_distribution.APPLIED ?? 0

  return (
    <section className="animate-slide-up space-y-5" aria-labelledby="pipeline-heading">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h2 id="pipeline-heading" className="font-display text-lg font-semibold text-white">
            Pipeline overview
          </h2>
          <p className="text-sm text-slate-400">Every application's journey, from the first submission to the final outcome.</p>
        </div>
        <button type="button" className="btn btn-ghost" onClick={onRefresh} disabled={loading} id="refresh-graph">
          <RefreshCw className={`size-4 ${loading ? 'animate-spin' : ''}`} />
          <span className="hidden sm:inline">Refresh</span>
        </button>
      </div>

      {/* KPI cards */}
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4 lg:gap-4">
        <KpiCard
          icon={<Send />}
          gradient="from-sky-500 to-indigo-500"
          glow="bg-sky-500/20"
          label="Total applied"
          value={data.total_applied}
          hint={`${data.total_applications} tracked including saved`}
          delay={0}
        />
        <KpiCard
          icon={<Activity />}
          gradient="from-amber-400 to-orange-500"
          glow="bg-amber-500/20"
          label="Active pipeline"
          value={active}
          hint="Applied, responded, interviewing or offer"
          delay={60}
        />
        <KpiCard
          icon={<MessageSquareReply />}
          gradient="from-violet-500 to-fuchsia-500"
          glow="bg-violet-500/20"
          label="Response rate"
          value={data.total_applied ? (data.responded_count / data.total_applied) * 100 : 0}
          format={(v) => `${v < 10 && v > 0 ? v.toFixed(1) : Math.round(v)}%`}
          hint={`${data.responded_count} of ${data.total_applied} moved past Applied`}
          delay={120}
        />
        <KpiCard
          icon={<PartyPopper />}
          gradient="from-emerald-400 to-teal-500"
          glow="bg-emerald-500/20"
          label="Offer rate"
          value={data.total_applied ? (data.offer_count / data.total_applied) * 100 : 0}
          format={(v) => `${v < 10 && v > 0 ? v.toFixed(1) : Math.round(v)}%`}
          hint={`${data.offer_count} offer${data.offer_count === 1 ? '' : 's'} received`}
          delay={180}
        />
      </div>

      {/* Sankey */}
      <div className="glass relative overflow-hidden p-5 sm:p-6">
        <div className="pointer-events-none absolute -top-24 -right-24 size-72 rounded-full bg-violet-600/10 blur-3xl" />
        <div className="relative mb-4 flex flex-wrap items-start justify-between gap-3">
          <div>
            <h3 className="flex items-center gap-2 font-display text-base font-semibold text-white">
              <Workflow className="size-4 text-violet-300" /> Application flow
            </h3>
            <p className="mt-0.5 text-sm text-slate-400">How applied jobs moved through each stage and where they ended.</p>
          </div>
          {awaiting > 0 && (
            <span className="rounded-full border border-sky-400/20 bg-sky-400/10 px-3 py-1 text-xs text-sky-200">
              {awaiting} still awaiting a response
            </span>
          )}
        </div>
        {data.total_applied === 0 ? (
          <div className="flex h-64 flex-col items-center justify-center text-center text-sm text-slate-400">
            <p>Nothing has been marked as Applied yet.</p>
            <p className="mt-1 text-slate-500">Move a saved application to Applied to start the flow.</p>
          </div>
        ) : (
          <SankeyChartView data={data} />
        )}
        {data.omitted_backward_links > 0 && (
          <p className="relative mt-3 flex items-center gap-1.5 text-xs text-slate-500">
            <Info className="size-3.5" />
            {data.omitted_backward_links} backward transition{data.omitted_backward_links === 1 ? '' : 's'} (e.g. Rejected → Interviewing)
            hidden to keep the flow readable.
          </p>
        )}
      </div>

      {/* Distribution + stage reach */}
      <div className="grid gap-5 lg:grid-cols-5">
        <div className="glass p-5 sm:p-6 lg:col-span-3">
          <h3 className="font-display text-base font-semibold text-white">Current status distribution</h3>
          <p className="mt-0.5 mb-2 text-sm text-slate-400">Where every tracked application sits right now.</p>
          <DistributionChart distribution={data.current_distribution} />
        </div>
        <div className="glass p-5 sm:p-6 lg:col-span-2">
          <h3 className="font-display text-base font-semibold text-white">Stage reach</h3>
          <p className="mt-0.5 mb-5 text-sm text-slate-400">Share of applied jobs that ever reached each stage.</p>
          <StageReach data={data} />
        </div>
      </div>
    </section>
  )
}

/* ------------------------------------------------------------------ */
/* KPI card with count-up animation                                    */
/* ------------------------------------------------------------------ */

function useCountUp(target: number, duration = 900) {
  const [value, setValue] = useState(0)
  const fromRef = useRef(0)
  useEffect(() => {
    const from = fromRef.current
    let raf = 0
    const start = performance.now()
    const tick = (now: number) => {
      const t = Math.min(1, (now - start) / duration)
      const eased = 1 - Math.pow(1 - t, 3)
      const v = from + (target - from) * eased
      setValue(v)
      fromRef.current = v
      if (t < 1) raf = requestAnimationFrame(tick)
    }
    raf = requestAnimationFrame(tick)
    return () => cancelAnimationFrame(raf)
  }, [target, duration])
  return value
}

function KpiCard({
  icon,
  gradient,
  glow,
  label,
  value,
  hint,
  format = (v) => Math.round(v).toLocaleString(),
  delay,
}: {
  icon: ReactNode
  gradient: string
  glow: string
  label: string
  value: number
  hint: string
  format?: (v: number) => string
  delay: number
}) {
  const animated = useCountUp(value)
  return (
    <div
      className="glass group relative overflow-hidden p-4 transition-all duration-300 hover:-translate-y-0.5 hover:border-white/15 sm:p-5"
      style={{ animation: `slide-up 0.5s cubic-bezier(0.16,1,0.3,1) ${delay}ms both` }}
    >
      <div className={`pointer-events-none absolute -top-10 -right-10 size-32 rounded-full ${glow} blur-2xl transition-opacity duration-300 group-hover:opacity-150`} />
      <div className="relative flex items-center gap-3">
        <span
          className={`flex size-9 items-center justify-center rounded-xl bg-linear-to-br ${gradient} text-white shadow-lg shadow-black/30 [&>svg]:size-4`}
        >
          {icon}
        </span>
        <span className="text-xs font-medium tracking-wide text-slate-400 uppercase">{label}</span>
      </div>
      <div className="relative mt-4 font-display text-3xl font-semibold text-white tabular-nums sm:text-4xl">{format(animated)}</div>
      <p className="relative mt-1 truncate text-xs text-slate-500" title={hint}>
        {hint}
      </p>
    </div>
  )
}

/* ------------------------------------------------------------------ */
/* Sankey                                                              */
/* ------------------------------------------------------------------ */

interface SankeyTooltipParams {
  dataType?: 'node' | 'edge'
  name?: string
  data?: { source?: Status; target?: Status; name?: Status; value?: number }
  value?: number
}

function SankeyChartView({ data }: { data: GraphAnalytics }) {
  const option = useMemo<EChartsOption>(() => {
    // Node totals = max(inflow, outflow, explicit value) — mirrors echarts' own layout.
    const inflow = new Map<Status, number>()
    const outflow = new Map<Status, number>()
    for (const l of data.sankey_links) {
      outflow.set(l.source, (outflow.get(l.source) ?? 0) + l.value)
      inflow.set(l.target, (inflow.get(l.target) ?? 0) + l.value)
    }
    const totals = new Map<Status, number>()
    for (const n of data.sankey_nodes) {
      totals.set(n.name, Math.max(n.value ?? 0, inflow.get(n.name) ?? 0, outflow.get(n.name) ?? 0))
    }
    const label = (s: Status | string | undefined) => (s && s in STATUS_META ? STATUS_META[s as Status].label : String(s ?? ''))

    return {
      backgroundColor: 'transparent',
      animationDuration: 900,
      animationEasing: 'cubicOut',
      tooltip: {
        ...tooltipBase,
        trigger: 'item',
        formatter: (raw: unknown) => {
          const p = raw as SankeyTooltipParams
          if (p.dataType === 'edge' && p.data?.source && p.data?.target) {
            const v = p.data.value ?? 0
            const from = totals.get(p.data.source) ?? 0
            return `<div style="font-weight:600;margin-bottom:4px">${esc(label(p.data.source))} → ${esc(label(p.data.target))}</div>
              <div><b style="font-size:14px">${v}</b> application${v === 1 ? '' : 's'}
              <span style="color:#94a3b8">· ${percent(v, from)} of ${esc(label(p.data.source))}</span></div>`
          }
          const name = (p.name ?? '') as Status
          const total = totals.get(name) ?? 0
          const current = data.current_distribution[name] ?? 0
          return `<div style="font-weight:600;margin-bottom:4px">${esc(label(name))}</div>
            <div>Reached by <b>${data.reached_counts[name] ?? total}</b> · ${percent(data.reached_counts[name] ?? total, data.total_applied)} of applied</div>
            <div style="color:#94a3b8">Currently here: ${current}</div>`
        },
      },
      series: [
        {
          type: 'sankey',
          left: 4,
          right: 130,
          top: 12,
          bottom: 12,
          nodeWidth: 14,
          nodeGap: 22,
          layoutIterations: 64,
          draggable: true,
          emphasis: { focus: 'adjacency' },
          data: data.sankey_nodes.map((n) => ({
            name: n.name,
            value: n.value,
            itemStyle: {
              color: STATUS_META[n.name].color,
              borderColor: STATUS_META[n.name].color,
              borderRadius: 4,
              shadowBlur: 18,
              shadowColor: `${STATUS_META[n.name].color}66`,
            },
          })),
          links: data.sankey_links.map((l) => ({ ...l })),
          lineStyle: { color: 'gradient', curveness: 0.5, opacity: 0.32 },
          label: {
            color: '#e2e8f0',
            fontFamily: FONT,
            position: 'right',
            distance: 10,
            formatter: (p: { name: string }) => `{name|${label(p.name)}}  {val|${totals.get(p.name as Status) ?? 0}}`,
            rich: {
              name: { fontSize: 12, fontWeight: 500, color: '#e2e8f0', fontFamily: FONT },
              val: { fontSize: 12, fontWeight: 700, color: '#94a3b8', fontFamily: FONT },
            },
          },
        },
      ],
    } as unknown as EChartsOption
  }, [data])

  const height = Math.max(340, Math.min(560, data.sankey_nodes.length * 64))

  return <EChart option={option} style={{ height, width: '100%' }} />
}

/* ------------------------------------------------------------------ */
/* Current distribution bar chart                                      */
/* ------------------------------------------------------------------ */

function DistributionChart({ distribution }: { distribution: GraphAnalytics['current_distribution'] }) {
  const option = useMemo<EChartsOption>(() => {
    const rows = STATUSES.map((s) => ({ s, v: distribution[s] ?? 0 }))
    return {
      backgroundColor: 'transparent',
      animationDuration: 800,
      animationEasing: 'cubicOut',
      grid: { left: 8, right: 36, top: 8, bottom: 8, containLabel: true },
      tooltip: {
        ...tooltipBase,
        trigger: 'axis',
        axisPointer: { type: 'shadow', shadowStyle: { color: 'rgba(255,255,255,0.03)' } },
        formatter: (raw: unknown) => {
          const p = (raw as { dataIndex: number }[])[0]
          const row = rows[p.dataIndex]
          return `<b>${esc(STATUS_META[row.s].label)}</b>: ${row.v}<br/><span style="color:#94a3b8">${esc(STATUS_META[row.s].description)}</span>`
        },
      },
      xAxis: {
        type: 'value',
        minInterval: 1,
        splitLine: { lineStyle: { color: 'rgba(255,255,255,0.05)' } },
        axisLabel: { color: '#64748b', fontFamily: FONT },
      },
      yAxis: {
        type: 'category',
        inverse: true,
        data: rows.map((r) => STATUS_META[r.s].label),
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: { color: '#cbd5e1', fontFamily: FONT, fontSize: 12 },
      },
      series: [
        {
          type: 'bar',
          barWidth: 14,
          showBackground: true,
          backgroundStyle: { color: 'rgba(255,255,255,0.025)', borderRadius: 7 },
          data: rows.map((r) => ({
            value: r.v,
            itemStyle: {
              borderRadius: 7,
              color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
                { offset: 0, color: `${STATUS_META[r.s].color}55` },
                { offset: 1, color: STATUS_META[r.s].color },
              ]),
            },
          })),
          label: {
            show: true,
            position: 'right',
            color: '#94a3b8',
            fontFamily: FONT,
            fontWeight: 600,
          },
        },
      ],
    } as unknown as EChartsOption
  }, [distribution])

  return <EChart option={option} style={{ height: 340, width: '100%' }} />
}

/* ------------------------------------------------------------------ */
/* Stage reach (CSS progress bars)                                     */
/* ------------------------------------------------------------------ */

const REACH_STAGES: Status[] = ['RESPONSE_RECEIVED', 'INTERVIEWING', 'OFFER', 'ACCEPTED', 'REJECTED', 'GHOSTED', 'WITHDRAWN']

function StageReach({ data }: { data: GraphAnalytics }) {
  const [mounted, setMounted] = useState(false)
  useEffect(() => {
    const id = requestAnimationFrame(() => setMounted(true))
    return () => cancelAnimationFrame(id)
  }, [])

  if (!data.total_applied) {
    return <p className="text-sm text-slate-500">No applied jobs yet.</p>
  }

  return (
    <ul className="space-y-3.5">
      {REACH_STAGES.map((s, i) => {
        const n = data.reached_counts[s] ?? 0
        const pct = (n / data.total_applied) * 100
        const meta = STATUS_META[s]
        return (
          <li key={s}>
            <div className="mb-1.5 flex items-center justify-between text-sm">
              <span className="flex items-center gap-2 text-slate-300">
                <span className={`size-2 rounded-full ${meta.dot}`} />
                {meta.label}
              </span>
              <span className="text-slate-400 tabular-nums">
                <b className="font-semibold text-slate-200">{n}</b>
                <span className="ml-1.5 text-slate-500">{percent(n, data.total_applied)}</span>
              </span>
            </div>
            <div className="h-1.5 overflow-hidden rounded-full bg-white/[0.05]">
              <div
                className="h-full rounded-full transition-[width] duration-1000 ease-out"
                style={{
                  width: mounted ? `${pct}%` : '0%',
                  transitionDelay: `${i * 70}ms`,
                  background: `linear-gradient(90deg, ${meta.color}66, ${meta.color})`,
                  boxShadow: `0 0 12px ${meta.color}55`,
                }}
              />
            </div>
          </li>
        )
      })}
    </ul>
  )
}
