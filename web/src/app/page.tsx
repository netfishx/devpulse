"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import {
  GitCommit,
  GitPullRequest,
  CalendarDays,
  LogOut,
  Settings,
} from "lucide-react";

import {
  api,
  type UserResponse,
  type Activity,
  type DailySummary,
  type PeriodSummary,
  type HeatmapDay,
  type RepoStats,
} from "@/lib/api";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import { TrendChart } from "@/components/trend-chart";
import { Heatmap } from "@/components/heatmap";
import { TopRepos } from "@/components/top-repos";

// -- Helpers ------------------------------------------------------------------

function formatRelativeTime(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diffMs = now - then;
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHr = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHr / 24);

  if (diffSec < 60) return "just now";
  if (diffMin < 60) return `${diffMin} minute${diffMin === 1 ? "" : "s"} ago`;
  if (diffHr < 24) return `${diffHr} hour${diffHr === 1 ? "" : "s"} ago`;
  if (diffDay < 30) return `${diffDay} day${diffDay === 1 ? "" : "s"} ago`;
  const diffMonth = Math.floor(diffDay / 30);
  return `${diffMonth} month${diffMonth === 1 ? "" : "s"} ago`;
}

function badgeVariantForType(
  type: string
): "default" | "secondary" | "outline" | "destructive" {
  switch (type) {
    case "push":
      return "default";
    case "pull_request":
      return "secondary";
    case "review":
      return "outline";
    case "create":
      return "destructive";
    default:
      return "default";
  }
}

// -- Component ----------------------------------------------------------------

export default function DashboardPage() {
  const router = useRouter();

  const [user, setUser] = useState<UserResponse | null>(null);
  const [summaries, setSummaries] = useState<DailySummary[]>([]);
  const [activities, setActivities] = useState<Activity[]>([]);
  const [weeklySummaries, setWeeklySummaries] = useState<PeriodSummary[]>([]);
  const [monthlySummaries, setMonthlySummaries] = useState<PeriodSummary[]>([]);
  const [heatmapDays, setHeatmapDays] = useState<HeatmapDay[]>([]);
  const [topRepos, setTopRepos] = useState<RepoStats[]>([]);
  const [source, setSource] = useState("all");
  const [loading, setLoading] = useState(true);

  const redirectToLogin = useCallback(() => {
    localStorage.removeItem("token");
    router.replace("/login");
  }, [router]);

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) {
      router.replace("/login");
      return;
    }

    const sourceParam = source === "all" ? "" : source;

    async function load() {
      try {
        const [
          userData,
          summaryData,
          activityData,
          weeklyData,
          monthlyData,
          heatmapData,
          reposData,
        ] = await Promise.all([
          api.me(),
          api.summaries(60),
          api.activities(1, 20, sourceParam),
          api.weeklySummaries(24),
          api.monthlySummaries(24),
          api.heatmap(365),
          api.topRepos(30, sourceParam),
        ]);
        setUser(userData);
        setSummaries(summaryData.summaries);
        setActivities(activityData.activities);
        setWeeklySummaries(weeklyData.summaries);
        setMonthlySummaries(monthlyData.summaries);
        setHeatmapDays(heatmapData.days);
        setTopRepos(reposData.repos);
      } catch {
        redirectToLogin();
      } finally {
        setLoading(false);
      }
    }

    load();
  }, [router, redirectToLogin, source]);

  const handleLogout = () => {
    redirectToLogin();
  };

  // Loading state
  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="flex flex-col items-center gap-3">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
          <p className="text-sm text-muted-foreground">Loading...</p>
        </div>
      </div>
    );
  }

  // Period-over-period comparison (60 days split into current 30 + previous 30)
  const current30 = summaries.slice(0, 30);
  const prev30 = summaries.slice(30, 60);

  const currentCommits = current30.reduce((s, d) => s + d.totalCommits, 0);
  const prevCommits = prev30.reduce((s, d) => s + d.totalCommits, 0);
  const commitsDelta =
    prevCommits > 0
      ? Math.round(((currentCommits - prevCommits) / prevCommits) * 100)
      : 0;

  const currentPrs = current30.reduce((s, d) => s + d.totalPrs, 0);
  const prevPrs = prev30.reduce((s, d) => s + d.totalPrs, 0);
  const prsDelta =
    prevPrs > 0
      ? Math.round(((currentPrs - prevPrs) / prevPrs) * 100)
      : 0;

  const activeDays = current30.filter(
    (s) => s.totalCommits > 0 || s.totalPrs > 0
  ).length;
  const prevActiveDays = prev30.filter(
    (s) => s.totalCommits > 0 || s.totalPrs > 0
  ).length;
  const activeDaysDelta =
    prevActiveDays > 0
      ? Math.round(
          ((activeDays - prevActiveDays) / prevActiveDays) * 100
        )
      : 0;

  return (
    <div className="flex min-h-screen flex-col bg-background">
      {/* Header */}
      <header className="flex items-center justify-between border-b px-6 py-4">
        <h1 className="text-xl font-bold tracking-tight">DevPulse</h1>
        <div className="flex items-center gap-3">
          <span className="text-sm text-muted-foreground">
            {user?.name ?? ""}
          </span>
          <Select
            value={source}
            onValueChange={(value) => {
              if (value != null) setSource(value);
            }}
          >
            <SelectTrigger size="sm">
              <SelectValue placeholder="All Sources" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All</SelectItem>
              <SelectItem value="github">GitHub</SelectItem>
              <SelectItem value="wakatime" disabled>
                Wakatime
              </SelectItem>
            </SelectContent>
          </Select>
          <Link
            href="/settings"
            className="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
          >
            <Settings className="size-4" />
            Settings
          </Link>
          <Button variant="ghost" size="sm" onClick={handleLogout}>
            <LogOut data-icon="inline-start" />
            Logout
          </Button>
        </div>
      </header>

      {/* Main content */}
      <main className="flex flex-1 flex-col gap-6 p-6">
        {/* Summary Cards Row */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-sm text-muted-foreground">
                <GitCommit className="size-4" />
                Total Commits
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex flex-col gap-1">
                <p className="text-3xl font-bold tabular-nums">
                  {currentCommits}
                </p>
                {commitsDelta !== 0 && (
                  <p
                    className={`text-xs font-medium ${commitsDelta > 0 ? "text-emerald-600" : "text-red-500"}`}
                  >
                    {commitsDelta > 0 ? "\u2191" : "\u2193"}
                    {Math.abs(commitsDelta)}% vs prev 30d
                  </p>
                )}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-sm text-muted-foreground">
                <GitPullRequest className="size-4" />
                Total PRs
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex flex-col gap-1">
                <p className="text-3xl font-bold tabular-nums">{currentPrs}</p>
                {prsDelta !== 0 && (
                  <p
                    className={`text-xs font-medium ${prsDelta > 0 ? "text-emerald-600" : "text-red-500"}`}
                  >
                    {prsDelta > 0 ? "\u2191" : "\u2193"}
                    {Math.abs(prsDelta)}% vs prev 30d
                  </p>
                )}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-sm text-muted-foreground">
                <CalendarDays className="size-4" />
                Active Days
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex flex-col gap-1">
                <p className="text-3xl font-bold tabular-nums">{activeDays}</p>
                {activeDaysDelta !== 0 && (
                  <p
                    className={`text-xs font-medium ${activeDaysDelta > 0 ? "text-emerald-600" : "text-red-500"}`}
                  >
                    {activeDaysDelta > 0 ? "\u2191" : "\u2193"}
                    {Math.abs(activeDaysDelta)}% vs prev 30d
                  </p>
                )}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Activity Trends (replaces the old 30-day bar chart) */}
        <TrendChart
          dailySummaries={summaries}
          weeklySummaries={weeklySummaries}
          monthlySummaries={monthlySummaries}
        />

        {/* Top Repos + Heatmap */}
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Top Repos</CardTitle>
            </CardHeader>
            <CardContent>
              <TopRepos repos={topRepos} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Contributions</CardTitle>
            </CardHeader>
            <CardContent>
              <Heatmap days={heatmapDays} />
            </CardContent>
          </Card>
        </div>

        {/* Activity Timeline */}
        <Card>
          <CardHeader>
            <CardTitle>Recent Activity</CardTitle>
          </CardHeader>
          <CardContent>
            {activities.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No recent activity.
              </p>
            ) : (
              <div className="flex flex-col gap-3">
                {activities.map((activity) => (
                  <div
                    key={activity.id}
                    className="flex items-center gap-3 rounded-lg border px-4 py-3"
                  >
                    <Badge variant={badgeVariantForType(activity.type)}>
                      {activity.type}
                    </Badge>
                    <span className="flex-1 truncate text-sm font-medium">
                      {(activity.payload?.repo as string) ?? "unknown"}
                    </span>
                    <span className="shrink-0 text-xs text-muted-foreground">
                      {formatRelativeTime(activity.occurredAt)}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </main>
    </div>
  );
}
