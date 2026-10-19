import { memo } from 'react'

/**
 * Atmospheric background geometry layer with elegant sans-serif telemetry markings:
 * - Low-poly geometric refraction facets
 * - Asymmetric registration crosshairs (+)
 * - Sweeping architectural orbital arcs
 * - Peripheral vertical telemetry ruler tracks
 */
export const AmbientCanvas = memo(function AmbientCanvas() {
  return (
    <div
      className="pointer-events-none fixed inset-0 z-0 overflow-hidden select-none"
      aria-hidden="true"
    >
      {/* SVG Polygons, Refraction Facets & Sweeping Orbital Arcs */}
      <svg
        className="absolute inset-0 size-full"
        xmlns="http://www.w3.org/2000/svg"
        preserveAspectRatio="none"
      >
        <defs>
          <linearGradient id="facet-grad-1" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#FFFFFF" stopOpacity="0.8" />
            <stop offset="100%" stopColor="#E9EEF4" stopOpacity="0.4" />
          </linearGradient>
          <linearGradient id="facet-grad-2" x1="100%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" stopColor="#F6F8FB" stopOpacity="0.6" />
            <stop offset="100%" stopColor="#DFE6EE" stopOpacity="0.35" />
          </linearGradient>
          <linearGradient id="facet-grad-3" x1="0%" y1="100%" x2="100%" y2="0%">
            <stop offset="0%" stopColor="#FFFFFF" stopOpacity="0.9" />
            <stop offset="100%" stopColor="#EDF2F7" stopOpacity="0.5" />
          </linearGradient>
        </defs>

        {/* Diagonal Crystalline Shards */}
        <polygon points="0,0 65vw,0 35vw,55vh 0,35vh" fill="url(#facet-grad-1)" />
        <polygon points="45vw,0 100vw,0 100vw,65vh 55vw,35vh" fill="url(#facet-grad-2)" />
        <polygon points="15vw,50vh 85vw,35vh 100vw,100vh 25vw,100vh" fill="url(#facet-grad-3)" />
        <polygon points="0,40vh 40vw,60vh 15vw,100vh 0,100vh" fill="url(#facet-grad-1)" opacity="0.5" />

        {/* Sweeping Architectural Orbital Arcs */}
        <circle
          cx="82vw"
          cy="18vh"
          r="420"
          fill="none"
          stroke="rgba(15, 20, 25, 0.04)"
          strokeWidth="1"
          strokeDasharray="4 6"
        />
        <circle
          cx="82vw"
          cy="18vh"
          r="620"
          fill="none"
          stroke="rgba(15, 20, 25, 0.03)"
          strokeWidth="1"
        />
        <circle
          cx="12vw"
          cy="85vh"
          r="540"
          fill="none"
          stroke="rgba(15, 20, 25, 0.035)"
          strokeWidth="1"
        />
      </svg>

      {/* Scattered Optical Registration Crosshairs (+) */}
      <div className="absolute top-[8%] left-[4%] text-[10px] font-sans font-medium text-slate-400/40">+</div>
      <div className="absolute top-[14%] right-[12%] text-[10px] font-sans font-medium text-slate-400/40">+</div>
      <div className="absolute top-[48%] left-[1.5%] text-[10px] font-sans font-medium text-slate-400/35">+</div>
      <div className="absolute top-[38%] right-[3%] text-[10px] font-sans font-medium text-slate-400/35">+</div>
      <div className="absolute top-[68%] left-[28%] text-[10px] font-sans font-medium text-slate-400/30">+</div>
      <div className="absolute top-[75%] right-[22%] text-[10px] font-sans font-medium text-slate-400/35">+</div>
      <div className="absolute top-[92%] left-[8%] text-[10px] font-sans font-medium text-slate-400/40">+</div>
      <div className="absolute top-[88%] right-[8%] text-[10px] font-sans font-medium text-slate-400/40">+</div>

      {/* Left Peripheral Telemetry Ruler Track */}
      <div className="hidden lg:flex fixed top-0 bottom-0 left-2 flex-col justify-between py-8 text-[9px] font-sans font-semibold tracking-wider text-slate-400/40 select-none">
        <div className="flex items-center gap-1">
          <span className="h-px w-2 bg-slate-400/30" />
        </div>
        <div className="flex items-center gap-1">
          <span className="h-px w-1.5 bg-slate-400/20" />
        </div>
        <div className="flex items-center gap-1">
          <span className="h-px w-2 bg-slate-400/30" />
        </div>
        <div className="flex items-center gap-1">
          <span className="h-px w-1.5 bg-slate-400/20" />
        </div>
        <div className="flex items-center gap-1">
          <span className="h-px w-2 bg-slate-400/30" />
        </div>
      </div>

      {/* Right Peripheral Telemetry Ruler Track */}
      <div className="hidden lg:flex fixed top-0 bottom-0 right-2 flex-col justify-between py-8 text-[9px] font-sans font-semibold tracking-wider text-slate-400/40 select-none items-end">
        <div className="flex items-center gap-1">
          <span className="h-px w-2 bg-slate-400/30" />
        </div>
        <div className="flex items-center gap-1">
          <span className="h-px w-1.5 bg-slate-400/20" />
        </div>
        <div className="flex items-center gap-1">
          <span className="h-px w-2 bg-slate-400/30" />
        </div>
        <div className="flex items-center gap-1">
          <span className="h-px w-1.5 bg-slate-400/20" />
        </div>
        <div className="flex items-center gap-1">
          <span className="h-px w-2 bg-slate-400/30" />
        </div>
      </div>
    </div>
  )
})
