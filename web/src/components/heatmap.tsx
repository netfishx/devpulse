import type { HeatmapDay } from "@/lib/api"

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface HeatmapProps {
  days: HeatmapDay[]
}

// ---------------------------------------------------------------------------
// Color mapping by activity level
// ---------------------------------------------------------------------------

const levelClasses: Record<number, string> = {
  0: "bg-muted",
  1: "bg-emerald-200 dark:bg-emerald-900",
  2: "bg-emerald-400 dark:bg-emerald-700",
  3: "bg-emerald-600 dark:bg-emerald-500",
  4: "bg-emerald-800 dark:bg-emerald-300",
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function buildCells(days: HeatmapDay[]) {
  const dayMap = new Map(days.map((d) => [d.date, d]))

  const today = new Date()
  const cells: { date: string; level: number; count: number }[] = []

  for (let i = 364; i >= 0; i--) {
    const d = new Date(today)
    d.setDate(d.getDate() - i)
    const dateStr = d.toISOString().split("T")[0]!
    const data = dayMap.get(dateStr)
    cells.push({
      date: dateStr,
      level: data?.level ?? 0,
      count: data?.count ?? 0,
    })
  }

  // Pad start so the first cell aligns to Sunday (column start)
  const firstDayOfWeek = new Date(cells[0]!.date).getDay() // 0 = Sun
  const padding = firstDayOfWeek

  return { cells, padding }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function Heatmap({ days }: HeatmapProps) {
  if (days.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No contribution data yet.
      </p>
    )
  }

  const { cells, padding } = buildCells(days)

  return (
    <div className="flex flex-col gap-2">
      {/* Grid */}
      <div className="overflow-x-auto">
        <div className="grid grid-flow-col grid-rows-7 gap-[3px]">
          {/* Padding cells to align first day to correct row */}
          {Array.from({ length: padding }, (_, i) => (
            <div key={`pad-${i}`} className="size-3" />
          ))}

          {/* Data cells */}
          {cells.map((cell) => (
            <div
              key={cell.date}
              className={`size-3 rounded-sm ${levelClasses[cell.level] ?? levelClasses[0]}`}
              title={`${cell.date}: ${cell.count} contributions`}
            />
          ))}
        </div>
      </div>

      {/* Legend */}
      <div className="flex items-center gap-1 text-xs text-muted-foreground">
        <span>Less</span>
        {[0, 1, 2, 3, 4].map((level) => (
          <div
            key={level}
            className={`size-3 rounded-sm ${levelClasses[level]}`}
          />
        ))}
        <span>More</span>
      </div>
    </div>
  )
}
