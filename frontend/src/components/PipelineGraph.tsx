import type { EChartsOption } from 'echarts'
import { BarChart, SankeyChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import * as echarts from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import EChart from './EChart'
import { Info, Loader2, RefreshCw, Workflow } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { api } from '../api/client'
import { percent } from '../lib/format'
import { ACTIVE_STATUSES, STATUSES, STATUS_META, type GraphAnalytics, type Status } from '../types'

echarts.use([SankeyChart, BarChart, TooltipComponent, GridComponent, CanvasRenderer])

interface PipelineGraphProps {
  refreshKey: number
  onCreate: () => void
}

const FONT_SANS = '"Plus Jakarta Sans", -apple-system, BlinkMacSystemFont, sans-serif'

const clinicalTooltip = {
  backgroundColor: '#ffffff',
  borderColor: 'rgba(15, 20, 25, 0.16)',
  borderWidth: 1,
  padding: [8, 12],
  textStyle: { color: '#0f1419', fontFamily: FONT_SANS, fontSize: 12 },
  extraCssText: 'box-shadow: 0 8px 24px -4px rgba(15,20,25,0.12); border-radius: 2px;',
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
      <div className="flex flex-col items-center gap-3 border border-slate-300/80 bg-white p-12 text-center">
        <p className="font-sans text-xs text-rose-600">{error}</p>
        <button
          className="btn-clinical btn-clinical-ghost"
          onClick={() => setReload((r) => r + 1)}
          id="retry-graph"
        >
          Retry Load
        </button>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="flex items-center justify-center py-28 font-sans text-xs font-medium text-slate-500">
        <Loader2 className="mr-2 size-4 animate-spin text-[#0088cc]" />
        <span>PROCESSING TELEMETRY MATRIX…</span>
      </div>
    )
  }

  if (data.total_applications === 0) {
    return (
      <div className="flex flex-col items-center border border-slate-300/80 bg-white p-16 text-center">
        <div className="mb-3 flex size-12 items-center justify-center border border-slate-300 bg-slate-50">
          <Workflow className="size-6 text-slate-400" />
        </div>
        <h2 className="font-sans text-xs font-bold tracking-wider text-slate-800 uppercase">
          Zero Flow Telemetry Detected
        </h2>
        <p className="mt-1 max-w-sm text-xs text-slate-500 font-sans">
          Populate records in the Applications repository to compute flow metrics and Sankey stage routing.
        </p>
        <button className="btn-clinical btn-clinical-primary mt-4" onClick={onCreate} id="graph-empty-add">
          New Application
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
    <section className="space-y-6" aria-labelledby="pipeline-heading">
      {/* Sub-header & Controls */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between border-b border-[rgba(15,20,25,0.1)] pb-3">
        <div>
          <div className="flex items-center gap-2">
            <span className="font-sans text-xs font-bold text-[#0088cc] uppercase">[ SPEC: SANKEY_DAG ]</span>
            <span className="font-sans text-[11px] font-medium text-slate-400 uppercase">TELEMETRY_PIPELINE</span>
          </div>
          <h2 id="pipeline-heading" className="mt-0.5 font-display text-2xl font-light tracking-wide text-slate-900 uppercase">
            Application Flow &amp; Analytics
          </h2>
        </div>
        <button
          type="button"
          className="btn-clinical btn-clinical-ghost self-start sm:self-auto"
          onClick={onRefresh}
          disabled={loading}
          id="refresh-graph"
        >
          <RefreshCw className={`size-3 ${loading ? 'animate-spin' : ''}`} />
          <span>Sync Metrics</span>
        </button>
      </div>

      {/* KPI Numerical Counters: Monumental Ultralight Scale */}
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4 lg:gap-4">
        <KpiCard
          label="Total Applied"
          value={data.total_applied}
          hint={`${data.total_applications} tracked (incl. saved)`}
          accent="#0088cc"
          delay={0}
        />
        <KpiCard
          label="Active Pipeline"
          value={active}
          hint="Applied, interviewing or offer"
          accent="#d97706"
          delay={50}
        />
        <KpiCard
          label="Response Rate"
          value={data.total_applied ? (data.responded_count / data.total_applied) * 100 : 0}
          format={(v) => `${v < 10 && v > 0 ? v.toFixed(1) : Math.round(v)}%`}
          hint={`${data.responded_count} of ${data.total_applied} past Applied`}
          accent="#4f46e5"
          delay={100}
        />
        <KpiCard
          label="Offer Rate"
          value={data.total_applied ? (data.offer_count / data.total_applied) * 100 : 0}
          format={(v) => `${v < 10 && v > 0 ? v.toFixed(1) : Math.round(v)}%`}
          hint={`${data.offer_count} total offer${data.offer_count === 1 ? '' : 's'} logged`}
          accent="#059669"
          delay={150}
        />
      </div>

      {/* Main Sankey Diagram Area */}
      <div className="relative border border-[rgba(15,20,25,0.12)] bg-white/70 p-5 backdrop-blur-xs">
        {/* Exterior L-Brackets */}
        <span className="corner-l-bracket-tl" />
        <span className="corner-l-bracket-tr" />
        <span className="corner-l-bracket-bl" />
        <span className="corner-l-bracket-br" />

        <div className="mb-4 flex flex-wrap items-start justify-between gap-3 border-b border-slate-200 pb-3">
          <div>
            <h3 className="flex items-center gap-2 font-sans text-xs font-bold tracking-wider text-slate-800 uppercase">
              <Workflow className="size-3.5 text-[#0088cc]" />
              <span>Stage Transition Routing</span>
            </h3>
            <p className="mt-0.5 font-sans text-xs text-slate-500">
              Interactive acyclic flow graph mapping volume transitions across application states.
            </p>
          </div>
          {awaiting > 0 && (
            <span className="border border-sky-300 bg-sky-50 px-2 py-0.5 font-sans text-[11px] font-semibold text-sky-800">
              {awaiting} PENDING INITIAL RESPONSE
            </span>
          )}
        </div>

        {data.total_applied === 0 ? (
          <div className="flex h-56 flex-col items-center justify-center font-sans text-xs text-slate-400">
            <p>NO APPLIED TRANSITIONS COMMITTED YET</p>
            <p className="mt-1 text-[11px] text-slate-500">Move a Saved record to Applied to initialize flow paths.</p>
          </div>
        ) : (
          <SankeyChartView data={data} />
        )}

        {data.omitted_backward_links > 0 && (
          <p className="mt-3 flex items-center gap-1.5 font-sans text-[11px] text-slate-400 border-t border-slate-100 pt-2">
            <Info className="size-3 text-slate-400" />
            <span>
              {data.omitted_backward_links} backward transition{data.omitted_backward_links === 1 ? '' : 's'} omitted to preserve DAG layout.
            </span>
          </p>
        )}
      </div>

      {/* Distribution & Stage Penetration Grid */}
      <div className="grid gap-5 lg:grid-cols-5">
        {/* Current Stage Distribution */}
        <div className="relative border border-[rgba(15,20,25,0.12)] bg-white/70 p-5 backdrop-blur-xs lg:col-span-3">
          <div className="mb-3 border-b border-slate-200 pb-2">
            <h3 className="font-sans text-xs font-bold tracking-wider text-slate-800 uppercase">
              Current Stage Partitioning
            </h3>
            <p className="mt-0.5 font-sans text-xs text-slate-500">Active distribution index across all tracked applications.</p>
          </div>
          <DistributionChart distribution={data.current_distribution} />
        </div>

        {/* Stage Penetration Reach */}
        <div className="relative border border-[rgba(15,20,25,0.12)] bg-white/70 p-5 backdrop-blur-xs lg:col-span-2">
          <div className="mb-3 border-b border-slate-200 pb-2">
            <h3 className="font-sans text-xs font-bold tracking-wider text-slate-800 uppercase">
              Stage Penetration Reach
            </h3>
            <p className="mt-0.5 font-sans text-xs text-slate-500">Proportion of applied volume that ever reached each stage.</p>
          </div>
          <StageReach data={data} />
        </div>
      </div>
    </section>
  )
}

/* ------------------------------------------------------------------ */
/* KPI Card with count-up animation & minimal bracket styling          */
/* ------------------------------------------------------------------ */

function useCountUp(target: number, duration = 800) {
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
  label,
  value,
  hint,
  accent,
  format = (v) => Math.round(v).toLocaleString(),
  delay,
}: {
  label: string
  value: number
  hint: string
  accent: string
  format?: (v: number) => string
  delay: number
}) {
  const animated = useCountUp(value)
  return (
    <div
      className="relative border border-[rgba(15,20,25,0.12)] bg-white/80 p-4 transition-all hover:border-[rgba(15,20,25,0.3)] sm:p-5"
      style={{
        borderLeft: `3px solid ${accent}`,
        animation: `slide-up 0.35s cubic-bezier(0.16,1,0.3,1) ${delay}ms both`,
      }}
    >
      <div className="font-sans text-[11px] font-bold tracking-widest text-slate-500 uppercase">
        {label}
      </div>
      <div className="mt-2 font-display text-3xl font-extralight tracking-tight text-slate-900 tabular-nums sm:text-4xl">
        {format(animated)}
      </div>
      <p className="mt-1 truncate font-sans text-[11px] text-slate-400" title={hint}>
        {hint}
      </p>
    </div>
  )
}

/* ------------------------------------------------------------------ */
/* Sankey Chart View                                                   */
/* ------------------------------------------------------------------ */

interface SankeyTooltipParams {
  dataType?: 'node' | 'edge'
  name?: string
  data?: { source?: Status; target?: Status; name?: Status; value?: number }
  value?: number
}

function SankeyChartView({ data }: { data: GraphAnalytics }) {
  const option = useMemo<EChartsOption>(() => {
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
    const label = (s: Status | string | undefined) =>
      s && s in STATUS_META ? STATUS_META[s as Status].label : String(s ?? '')

    return {
      backgroundColor: 'transparent',
      animationDuration: 700,
      animationEasing: 'cubicOut',
      tooltip: {
        ...clinicalTooltip,
        trigger: 'item',
        formatter: (raw: unknown) => {
          const p = raw as SankeyTooltipParams
          if (p.dataType === 'edge' && p.data?.source && p.data?.target) {
            const v = p.data.value ?? 0
            const from = totals.get(p.data.source) ?? 0
            return `<div style="font-family:${FONT_SANS};font-size:12px;font-weight:600;margin-bottom:4px;color:#0f1419;">
              ${esc(label(p.data.source))} → ${esc(label(p.data.target))}</div>
              <div style="font-size:12px;font-family:${FONT_SANS};"><b style="font-size:13px;color:#0f1419;">${v}</b> applications
              <span style="color:#64748b;">· ${percent(v, from)} of ${esc(label(p.data.source))}</span></div>`
          }
          const name = (p.name ?? '') as Status
          const total = totals.get(name) ?? 0
          const current = data.current_distribution[name] ?? 0
          return `<div style="font-family:${FONT_SANS};font-size:12px;font-weight:600;margin-bottom:4px;color:#0f1419;">
            ${esc(label(name))}</div>
            <div style="font-size:12px;font-family:${FONT_SANS};">Reached by <b style="color:#0f1419;">${data.reached_counts[name] ?? total}</b> · ${percent(
              data.reached_counts[name] ?? total,
              data.total_applied,
            )} of applied</div>
            <div style="color:#64748b;font-size:11px;margin-top:2px;font-family:${FONT_SANS};">Currently partitioned here: <b>${current}</b></div>`
        },
      },
      series: [
        {
          type: 'sankey',
          left: 12,
          right: 140,
          top: 14,
          bottom: 14,
          nodeWidth: 12,
          nodeGap: 24,
          layoutIterations: 64,
          draggable: true,
          emphasis: { focus: 'adjacency' },
          data: data.sankey_nodes.map((n) => ({
            name: n.name,
            value: n.value,
            itemStyle: {
              color: STATUS_META[n.name].color,
              borderColor: 'rgba(15, 20, 25, 0.25)',
              borderWidth: 1,
              borderRadius: 1,
            },
          })),
          links: data.sankey_links.map((l) => ({ ...l })),
          lineStyle: { color: 'gradient', curveness: 0.5, opacity: 0.28 },
          label: {
            color: '#0f1419',
            fontFamily: FONT_SANS,
            position: 'right',
            distance: 10,
            formatter: (p: { name: string }) =>
              `{name|${label(p.name)}} {val|[${totals.get(p.name as Status) ?? 0}]}`,
            rich: {
              name: { fontSize: 11, fontWeight: 600, color: '#1e293b', fontFamily: FONT_SANS },
              val: { fontSize: 11, fontWeight: 600, color: '#64748b', fontFamily: FONT_SANS },
            },
          },
        },
      ],
    } as unknown as EChartsOption
  }, [data])

  const height = Math.max(340, Math.min(540, data.sankey_nodes.length * 60))

  return <EChart option={option} style={{ height, width: '100%' }} />
}

/* ------------------------------------------------------------------ */
/* Current Distribution Chart (Clean Light Grid)                       */
/* ------------------------------------------------------------------ */

function DistributionChart({ distribution }: { distribution: GraphAnalytics['current_distribution'] }) {
  const option = useMemo<EChartsOption>(() => {
    const rows = STATUSES.map((s) => ({ s, v: distribution[s] ?? 0 }))
    return {
      backgroundColor: 'transparent',
      animationDuration: 700,
      animationEasing: 'cubicOut',
      grid: { left: 8, right: 36, top: 8, bottom: 8, containLabel: true },
      tooltip: {
        ...clinicalTooltip,
        trigger: 'axis',
        axisPointer: { type: 'shadow', shadowStyle: { color: 'rgba(15,20,25,0.03)' } },
        formatter: (raw: unknown) => {
          const p = (raw as { dataIndex: number }[])[0]
          const row = rows[p.dataIndex]
          return `<b style="font-family:${FONT_SANS};font-size:12px;">${esc(STATUS_META[row.s].label)}</b>: ${row.v}<br/><span style="color:#64748b;font-size:11px;font-family:${FONT_SANS};">${esc(STATUS_META[row.s].description)}</span>`
        },
      },
      xAxis: {
        type: 'value',
        minInterval: 1,
        splitLine: { lineStyle: { color: 'rgba(15,20,25,0.06)' } },
        axisLabel: { color: '#64748b', fontFamily: FONT_SANS, fontSize: 10 },
      },
      yAxis: {
        type: 'category',
        inverse: true,
        data: rows.map((r) => STATUS_META[r.s].label),
        axisLine: { lineStyle: { color: 'rgba(15,20,25,0.15)' } },
        axisTick: { show: false },
        axisLabel: { color: '#334155', fontFamily: FONT_SANS, fontSize: 11, fontWeight: 500 },
      },
      series: [
        {
          type: 'bar',
          barWidth: 12,
          showBackground: true,
          backgroundStyle: { color: 'rgba(15,20,25,0.03)', borderRadius: 1 },
          data: rows.map((r) => ({
            value: r.v,
            itemStyle: {
              borderRadius: 1,
              color: STATUS_META[r.s].color,
            },
          })),
          label: {
            show: true,
            position: 'right',
            color: '#475569',
            fontFamily: FONT_SANS,
            fontSize: 11,
            fontWeight: 600,
          },
        },
      ],
    } as unknown as EChartsOption
  }, [distribution])

  return <EChart option={option} style={{ height: 320, width: '100%' }} />
}

/* ------------------------------------------------------------------ */
/* Stage Reach (Minimalist Progress Bars)                              */
/* ------------------------------------------------------------------ */

const REACH_STAGES: Status[] = ['RESPONSE_RECEIVED', 'INTERVIEWING', 'OFFER', 'ACCEPTED', 'REJECTED', 'GHOSTED', 'WITHDRAWN']

function StageReach({ data }: { data: GraphAnalytics }) {
  const [mounted, setMounted] = useState(false)
  useEffect(() => {
    const id = requestAnimationFrame(() => setMounted(true))
    return () => cancelAnimationFrame(id)
  }, [])

  if (!data.total_applied) {
    return <p className="font-sans text-xs text-slate-400">NO APPLIED RECORDS TO ANALYZE</p>
  }

  return (
    <ul className="space-y-3.5">
      {REACH_STAGES.map((s, i) => {
        const n = data.reached_counts[s] ?? 0
        const pct = (n / data.total_applied) * 100
        const meta = STATUS_META[s]
        return (
          <li key={s}>
            <div className="mb-1 flex items-center justify-between font-sans text-xs">
              <span className="flex items-center gap-1.5 font-medium text-slate-700">
                <span className={`size-1.5 ${meta.dot}`} style={{ borderRadius: '1px' }} />
                <span>{meta.label}</span>
              </span>
              <span className="text-slate-500 tabular-nums">
                <b className="font-semibold text-slate-900">{n}</b>
                <span className="ml-1 text-[11px] font-medium text-slate-400">[{percent(n, data.total_applied)}]</span>
              </span>
            </div>
            <div className="h-1 overflow-hidden bg-slate-200/80">
              <div
                className="h-full transition-[width] duration-700 ease-out"
                style={{
                  width: mounted ? `${pct}%` : '0%',
                  transitionDelay: `${i * 50}ms`,
                  backgroundColor: meta.color,
                }}
              />
            </div>
          </li>
        )
      })}
    </ul>
  )
}
