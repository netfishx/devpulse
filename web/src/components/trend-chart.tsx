"use client"

import {
  ComposedChart,
  Bar,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
} from "recharts"

import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart"
import type { DailySummary, PeriodSummary } from "@/lib/api"

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface TrendChartProps {
  dailySummaries: DailySummary[]
  weeklySummaries: PeriodSummary[]
  monthlySummaries: PeriodSummary[]
}

// ---------------------------------------------------------------------------
// Chart data point
// ---------------------------------------------------------------------------

interface ChartDataPoint {
  label: string
  commits: number
  prs: number
  prevCommits: number
}

// ---------------------------------------------------------------------------
// Chart config
// ---------------------------------------------------------------------------

const chartConfig = {
  commits: { label: "Commits", color: "var(--color-chart-1)" },
  prs: { label: "PRs", color: "var(--color-chart-2)" },
  prevCommits: { label: "Prev Period", color: "var(--color-chart-3)" },
} satisfies ChartConfig

// ---------------------------------------------------------------------------
// Format helpers
// ---------------------------------------------------------------------------

function formatDate(dateStr: string): string {
  const d = new Date(dateStr)
  return `${String(d.getMonth() + 1).padStart(2, "0")}/${String(d.getDate()).padStart(2, "0")}`
}

function formatWeek(period: string): string {
  return period.split("-")[1] ?? period
}

function formatMonth(period: string): string {
  const months = [
    "Jan",
    "Feb",
    "Mar",
    "Apr",
    "May",
    "Jun",
    "Jul",
    "Aug",
    "Sep",
    "Oct",
    "Nov",
    "Dec",
  ]
  const monthIdx = parseInt(period.split("-")[1] ?? "1", 10) - 1
  return months[monthIdx] ?? period
}

// ---------------------------------------------------------------------------
// Data transformation
// ---------------------------------------------------------------------------

function prepareDailyData(summaries: DailySummary[]): ChartDataPoint[] {
  const sorted = [...summaries].reverse()
  const midpoint = Math.floor(sorted.length / 2)
  const prev = sorted.slice(0, midpoint)
  const current = sorted.slice(midpoint)

  return current.map((day, i) => ({
    label: formatDate(day.date),
    commits: day.totalCommits,
    prs: day.totalPrs,
    prevCommits: prev[i]?.totalCommits ?? 0,
  }))
}

function preparePeriodicData(
  summaries: PeriodSummary[],
  formatFn: (period: string) => string,
): ChartDataPoint[] {
  const midpoint = Math.floor(summaries.length / 2)
  const prev = summaries.slice(0, midpoint)
  const current = summaries.slice(midpoint)

  return current.map((item, i) => ({
    label: formatFn(item.period),
    commits: item.totalCommits,
    prs: item.totalPrs,
    prevCommits: prev[i]?.totalCommits ?? 0,
  }))
}

// ---------------------------------------------------------------------------
// Reusable chart view
// ---------------------------------------------------------------------------

function TrendChartView({ data }: { data: ChartDataPoint[] }) {
  return (
    <ChartContainer config={chartConfig} className="h-[300px] w-full">
      <ComposedChart data={data} accessibilityLayer>
        <CartesianGrid vertical={false} />
        <XAxis dataKey="label" tickLine={false} axisLine={false} />
        <YAxis tickLine={false} axisLine={false} />
        <ChartTooltip content={<ChartTooltipContent />} />
        <Bar
          dataKey="commits"
          stackId="current"
          fill="var(--color-commits)"
          radius={[0, 0, 0, 0]}
        />
        <Bar
          dataKey="prs"
          stackId="current"
          fill="var(--color-prs)"
          radius={[4, 4, 0, 0]}
        />
        <Line
          dataKey="prevCommits"
          type="monotone"
          stroke="var(--color-prevCommits)"
          strokeWidth={2}
          strokeDasharray="5 5"
          strokeOpacity={0.5}
          dot={false}
        />
      </ComposedChart>
    </ChartContainer>
  )
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function TrendChart({
  dailySummaries,
  weeklySummaries,
  monthlySummaries,
}: TrendChartProps) {
  const isEmpty =
    dailySummaries.length === 0 &&
    weeklySummaries.length === 0 &&
    monthlySummaries.length === 0

  if (isEmpty) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Activity Trends</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex h-[300px] items-center justify-center text-muted-foreground">
            No data yet
          </div>
        </CardContent>
      </Card>
    )
  }

  const dailyData = prepareDailyData(dailySummaries)
  const weeklyData = preparePeriodicData(weeklySummaries, formatWeek)
  const monthlyData = preparePeriodicData(monthlySummaries, formatMonth)

  return (
    <Card>
      <CardHeader>
        <CardTitle>Activity Trends</CardTitle>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="daily">
          <TabsList>
            <TabsTrigger value="daily">Daily</TabsTrigger>
            <TabsTrigger value="weekly">Weekly</TabsTrigger>
            <TabsTrigger value="monthly">Monthly</TabsTrigger>
          </TabsList>
          <TabsContent value="daily">
            <TrendChartView data={dailyData} />
          </TabsContent>
          <TabsContent value="weekly">
            <TrendChartView data={weeklyData} />
          </TabsContent>
          <TabsContent value="monthly">
            <TrendChartView data={monthlyData} />
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  )
}
