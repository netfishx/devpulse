"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Bar, BarChart, XAxis, YAxis } from "recharts";
import { GitCommit, GitPullRequest, CalendarDays, LogOut } from "lucide-react";

import {
  api,
  type UserResponse,
  type Activity,
  type DailySummary,
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
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";

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

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${month}/${day}`;
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

// -- Chart config -------------------------------------------------------------

const chartConfig = {
  commits: {
    label: "Commits",
    color: "var(--color-chart-1)",
  },
  prs: {
    label: "PRs",
    color: "var(--color-chart-2)",
  },
} satisfies ChartConfig;

// -- Component ----------------------------------------------------------------

export default function DashboardPage() {
  const router = useRouter();

  const [user, setUser] = useState<UserResponse | null>(null);
  const [summaries, setSummaries] = useState<DailySummary[]>([]);
  const [activities, setActivities] = useState<Activity[]>([]);
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

    async function load() {
      try {
        const [userData, summaryData, activityData] = await Promise.all([
          api.me(),
          api.summaries(30),
          api.activities(1, 20),
        ]);
        setUser(userData);
        setSummaries(summaryData.summaries);
        setActivities(activityData.activities);
      } catch {
        redirectToLogin();
      } finally {
        setLoading(false);
      }
    }

    load();
  }, [router, redirectToLogin]);

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

  // Computed summary values
  const totalCommits = summaries.reduce((sum, s) => sum + s.totalCommits, 0);
  const totalPrs = summaries.reduce((sum, s) => sum + s.totalPrs, 0);
  const activeDays = summaries.filter(
    (s) => s.totalCommits > 0 || s.totalPrs > 0
  ).length;

  // Chart data
  const chartData = summaries.map((s) => ({
    date: formatDate(s.date),
    commits: s.totalCommits,
    prs: s.totalPrs,
  }));

  return (
    <div className="flex min-h-screen flex-col bg-background">
      {/* Header */}
      <header className="flex items-center justify-between border-b px-6 py-4">
        <h1 className="text-xl font-bold tracking-tight">DevPulse</h1>
        <div className="flex items-center gap-3">
          <span className="text-sm text-muted-foreground">
            {user?.name ?? ""}
          </span>
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
              <p className="text-3xl font-bold tabular-nums">{totalCommits}</p>
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
              <p className="text-3xl font-bold tabular-nums">{totalPrs}</p>
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
              <p className="text-3xl font-bold tabular-nums">{activeDays}</p>
            </CardContent>
          </Card>
        </div>

        {/* 30-Day Activity Bar Chart */}
        <Card>
          <CardHeader>
            <CardTitle>30-Day Activity</CardTitle>
          </CardHeader>
          <CardContent>
            <ChartContainer config={chartConfig} className="h-[300px] w-full">
              <BarChart data={chartData} accessibilityLayer>
                <XAxis
                  dataKey="date"
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(value: string) => value}
                />
                <YAxis tickLine={false} axisLine={false} />
                <ChartTooltip content={<ChartTooltipContent />} />
                <Bar
                  dataKey="commits"
                  stackId="activity"
                  fill="var(--color-commits)"
                  radius={[0, 0, 0, 0]}
                />
                <Bar
                  dataKey="prs"
                  stackId="activity"
                  fill="var(--color-prs)"
                  radius={[4, 4, 0, 0]}
                />
              </BarChart>
            </ChartContainer>
          </CardContent>
        </Card>

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
