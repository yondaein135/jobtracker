import type { EChartsCoreOption } from 'echarts/core'
import * as echarts from 'echarts/core'
import { useEffect, useRef, type CSSProperties } from 'react'

interface EChartProps {
  option: EChartsCoreOption
  style?: CSSProperties
  className?: string
}

/**
 * Lightweight, pure ESM React wrapper for ECharts.
 * Handles automatic responsive resizing via ResizeObserver and safe lifecycle cleanup.
 */
export default function EChart({ option, style, className }: EChartProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<echarts.ECharts | null>(null)

  useEffect(() => {
    if (!containerRef.current) return

    const chart = echarts.init(containerRef.current, undefined, {
      renderer: 'canvas',
    })
    chartRef.current = chart

    const observer = new ResizeObserver(() => {
      chart.resize()
    })
    observer.observe(containerRef.current)

    return () => {
      observer.disconnect()
      chart.dispose()
      chartRef.current = null
    }
  }, [])

  useEffect(() => {
    if (chartRef.current) {
      chartRef.current.setOption(option, true)
    }
  }, [option])

  return <div ref={containerRef} style={style} className={className} />
}
