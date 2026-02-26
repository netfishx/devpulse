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

  activities: (page = 1, perPage = 20) =>
    request<ActivityListResponse>(
      `/api/activities?page=${page}&per_page=${perPage}`
    ),

  summaries: (days = 30) =>
    request<SummaryListResponse>(`/api/summaries?days=${days}`),
};
