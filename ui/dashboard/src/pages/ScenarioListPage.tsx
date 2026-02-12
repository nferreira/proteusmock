import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search } from 'lucide-react'
import { api, type ScenarioSummary } from '@/lib/api'
import { cn } from '@/lib/utils'

const methodColors: Record<string, string> = {
  GET: 'bg-emerald-100 text-emerald-800',
  POST: 'bg-blue-100 text-blue-800',
  PUT: 'bg-amber-100 text-amber-800',
  PATCH: 'bg-orange-100 text-orange-800',
  DELETE: 'bg-red-100 text-red-800',
}

export function ScenarioListPage() {
  const [scenarios, setScenarios] = useState<ScenarioSummary[]>([])
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const navigate = useNavigate()

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const data = search.trim()
        ? await api.searchScenarios(search.trim())
        : await api.listScenarios()
      setScenarios(data ?? [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load scenarios')
    } finally {
      setLoading(false)
    }
  }, [search])

  useEffect(() => {
    const timer = setTimeout(load, search ? 300 : 0)
    return () => clearTimeout(timer)
  }, [load, search])

  const extractPath = (pathKey: string, method: string) => {
    return pathKey.startsWith(method + ':') ? pathKey.slice(method.length + 1) : pathKey
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Scenarios</h1>
        <div className="relative">
          <Search
            size={16}
            className="absolute left-3 top-1/2 -translate-y-1/2 text-[hsl(var(--muted-foreground))]"
          />
          <input
            type="text"
            placeholder="Search scenarios..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="pl-9 pr-4 py-2 text-sm border border-[hsl(var(--border))] rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-[hsl(var(--ring))] w-72"
          />
        </div>
      </div>

      {error && (
        <div className="bg-red-50 text-red-700 px-4 py-3 rounded-md mb-4 text-sm">{error}</div>
      )}

      <div className="border border-[hsl(var(--border))] rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-[hsl(var(--muted))] text-left">
              <th className="px-4 py-3 font-medium text-[hsl(var(--muted-foreground))] w-10 text-right">#</th>
              <th className="px-4 py-3 font-medium text-[hsl(var(--muted-foreground))]">Method</th>
              <th className="px-4 py-3 font-medium text-[hsl(var(--muted-foreground))]">Path</th>
              <th className="px-4 py-3 font-medium text-[hsl(var(--muted-foreground))]">ID</th>
              <th className="px-4 py-3 font-medium text-[hsl(var(--muted-foreground))]">Name</th>
              <th className="px-4 py-3 font-medium text-[hsl(var(--muted-foreground))] text-right">
                Priority
              </th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-[hsl(var(--muted-foreground))]">
                  Loading...
                </td>
              </tr>
            ) : scenarios.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-[hsl(var(--muted-foreground))]">
                  No scenarios found
                </td>
              </tr>
            ) : (
              scenarios.map((s, i) => (
                <tr
                  key={s.id}
                  onClick={() => navigate(`/scenarios/${encodeURIComponent(s.id)}`)}
                  className="border-t border-[hsl(var(--border))] hover:bg-[hsl(var(--accent))] cursor-pointer transition-colors"
                >
                  <td className="px-4 py-3 text-right tabular-nums text-[hsl(var(--muted-foreground))]">
                    {i + 1}
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={cn(
                        'inline-block px-2 py-0.5 rounded text-xs font-medium',
                        methodColors[s.method] ?? 'bg-gray-100 text-gray-800',
                      )}
                    >
                      {s.method}
                    </span>
                  </td>
                  <td className="px-4 py-3 font-mono text-xs">
                    {extractPath(s.path_key, s.method)}
                  </td>
                  <td className="px-4 py-3 font-mono text-xs text-[hsl(var(--muted-foreground))]">
                    {s.id}
                  </td>
                  <td className="px-4 py-3">{s.name}</td>
                  <td className="px-4 py-3 text-right tabular-nums">{s.priority}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
