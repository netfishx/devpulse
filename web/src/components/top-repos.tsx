import { Badge } from "@/components/ui/badge"
import type { RepoStats } from "@/lib/api"

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface TopReposProps {
  repos: RepoStats[]
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function TopRepos({ repos }: TopReposProps) {
  if (repos.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">No repository data yet.</p>
    )
  }

  return (
    <div className="flex flex-col gap-3">
      {repos.map((repo, index) => (
        <div key={repo.name} className="flex items-center gap-3">
          <span className="w-5 shrink-0 text-right text-sm text-muted-foreground">
            {index + 1}
          </span>
          <span className="flex-1 truncate text-sm font-medium">
            {repo.name}
          </span>
          <Badge variant="secondary">{repo.count}</Badge>
        </div>
      ))}
    </div>
  )
}
