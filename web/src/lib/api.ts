const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token =
    typeof window !== "undefined" ? localStorage.getItem("token") : null;

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Request failed: ${res.status}`);
  }

  return res.json();
}

// Types aligned with docs/openapi.yaml

export interface UserResponse {
  id: number;
  email: string;
  name: string;
  avatarUrl: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface LoginResponse {
  token: string;
  user: UserResponse;
}

export interface Activity {
  id: number;
  source: string;
  type: string;
  payload: Record<string, unknown>;
  occurredAt: string;
}

export interface ActivityListResponse {
  activities: Activity[];
  total: number;
  page: number;
  perPage: number;
}

export interface DailySummary {
  date: string;
  totalCommits: number;
  totalPrs: number;
  codingMinutes: number;
}

export interface SummaryListResponse {
  summaries: DailySummary[];
}

export interface PeriodSummary {
  period: string;
  totalCommits: number;
  totalPrs: number;
  codingMinutes: number;
}

export interface PeriodSummariesResponse {
  summaries: PeriodSummary[];
}

export interface HeatmapDay {
  date: string;
  level: number;
  count: number;
}

export interface HeatmapResponse {
  days: HeatmapDay[];
}

export interface RepoStats {
  name: string;
  count: number;
  lastActive: string;
}

export interface TopReposResponse {
  repos: RepoStats[];
}

export interface DataSourceInfo {
  id: number;
  provider: string;
  connected: boolean;
  connectedAt: string;
}

export interface DataSourcesResponse {
  sources: DataSourceInfo[];
}

export const api = {
  register: (data: { email: string; name: string; password: string }) =>
    request<UserResponse>("/api/register", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  login: (data: { email: string; password: string }) =>
    request<LoginResponse>("/api/login", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  me: () => request<UserResponse>("/api/me"),

  activities: (page = 1, perPage = 20, source = "") =>
    request<ActivityListResponse>(
      `/api/activities?page=${page}&per_page=${perPage}${source ? `&source=${source}` : ""}`
    ),

  summaries: (days = 30) =>
    request<SummaryListResponse>(`/api/summaries?days=${days}`),

  weeklySummaries: (weeks = 12) =>
    request<PeriodSummariesResponse>(`/api/summaries/weekly?weeks=${weeks}`),

  monthlySummaries: (months = 12) =>
    request<PeriodSummariesResponse>(`/api/summaries/monthly?months=${months}`),

  heatmap: (days = 365) =>
    request<HeatmapResponse>(`/api/summaries/heatmap?days=${days}`),

  topRepos: (days = 30, source = "") =>
    request<TopReposResponse>(
      `/api/activities/top-repos?days=${days}${source ? `&source=${source}` : ""}`
    ),

  dataSources: () =>
    request<DataSourcesResponse>("/api/data-sources"),
};
