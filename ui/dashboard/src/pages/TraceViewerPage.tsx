import { useState, useEffect, useRef } from 'react'
import { api, type TraceEntry } from '@/lib/api'
import { cn } from '@/lib/utils'

export function TraceViewerPage() {
  const [entries, setEntries] = useState<TraceEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [count, setCount] = useState(50)
  const [expandedIndex, setExpandedIndex] = useState<number | null>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const load = async () => {
    try {
      const data = await api.getTrace(count)
      setEntries(data ?? [])
    } catch {
      // ignore transient errors on polling
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [count])

  useEffect(() => {
    if (autoRefresh) {
      intervalRef.current = setInterval(load, 2000)
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [autoRefresh, count])

  const methodColors: Record<string, string> = {
    GET: 'text-emerald-700',
    POST: 'text-blue-700',
    PUT: 'text-amber-700',
    PATCH: 'text-orange-700',
    DELETE: 'text-red-700',
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">Request Trace</h1>
        <div className="flex items-center gap-4">
          <label className="flex items-center gap-2 text-sm">
            <span className="text-[hsl(var(--muted-foreground))]">Last</span>
            <input
              type="number"
              min={1}
              max={500}
              value={count}
              onChange={e => setCount(Math.max(1, parseInt(e.target.value) || 50))}
              className="w-16 px-2 py-1 text-sm border border-[hsl(var(--border))] rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-[hsl(var(--ring))]"
            />
          </label>
          <label className="flex items-center gap-2 text-sm cursor-pointer">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={e => setAutoRefresh(e.target.checked)}
              className="rounded"
            />
            <span className="text-[hsl(var(--muted-foreground))]">Auto-refresh</span>
          </label>
        </div>
      </div>

      <div className="border border-[hsl(var(--border))] rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-[hsl(var(--muted))] text-left">
              <th className="px-4 py-3 font-medium text-[hsl(var(--muted-foreground))]">Time</th>
              <th className="px-4 py-3 font-medium text-[hsl(var(--muted-foreground))]">Method</th>
              <th className="px-4 py-3 font-medium text-[hsl(var(--muted-foreground))]">Path</th>
              <th className="px-4 py-3 font-medium text-[hsl(var(--muted-foreground))]">Matched</th>
              <th className="px-4 py-3 font-medium text-[hsl(var(--muted-foreground))]">Status</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-[hsl(var(--muted-foreground))]">
                  Loading...
                </td>
              </tr>
            ) : entries.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-[hsl(var(--muted-foreground))]">
                  No trace entries
                </td>
              </tr>
            ) : (
              entries.map((entry, i) => (
                <>
                  <tr
                    key={i}
                    onClick={() => setExpandedIndex(expandedIndex === i ? null : i)}
                    className="border-t border-[hsl(var(--border))] hover:bg-[hsl(var(--accent))] cursor-pointer transition-colors"
                  >
                    <td className="px-4 py-3 font-mono text-xs text-[hsl(var(--muted-foreground))]">
                      {entry.timestamp ? new Date(entry.timestamp).toLocaleTimeString() : '-'}
                    </td>
                    <td className={cn('px-4 py-3 font-mono text-xs font-medium', methodColors[entry.method])}>
                      {entry.method}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">{entry.path}</td>
                    <td className="px-4 py-3">
                      {entry.matched_id ? (
                        <span className="text-xs font-mono text-emerald-700">{entry.matched_id}</span>
                      ) : (
                        <span className="text-xs text-[hsl(var(--muted-foreground))]">none</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      {entry.rate_limited && (
                        <span className="inline-block px-2 py-0.5 rounded text-xs font-medium bg-amber-100 text-amber-800">
                          429
                        </span>
                      )}
                    </td>
                  </tr>
                  {expandedIndex === i && entry.candidates && entry.candidates.length > 0 && (
                    <tr key={`${i}-expanded`} className="border-t border-[hsl(var(--border))]">
                      <td colSpan={5} className="px-8 py-3 bg-[hsl(var(--muted))]">
                        <div className="text-xs font-medium text-[hsl(var(--muted-foreground))] mb-2">
                          Candidates evaluated:
                        </div>
                        <div className="grid gap-1">
                          {entry.candidates.map((c, ci) => (
                            <div key={ci} className="flex items-center gap-3 text-xs">
                              <span
                                className={cn(
                                  'inline-block w-14 text-center px-1.5 py-0.5 rounded font-medium',
                                  c.matched ? 'bg-emerald-100 text-emerald-800' : 'bg-red-100 text-red-800',
                                )}
                              >
                                {c.matched ? 'MATCH' : 'FAIL'}
                              </span>
                              <span className="font-mono">{c.scenario_id}</span>
                              <span className="text-[hsl(var(--muted-foreground))]">
                                {c.scenario_name}
                              </span>
                              {!c.matched && c.failed_field && (
                                <span className="text-red-600">
                                  field: {c.failed_field}
                                </span>
                              )}
                            </div>
                          ))}
                        </div>
                      </td>
                    </tr>
                  )}
                </>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
