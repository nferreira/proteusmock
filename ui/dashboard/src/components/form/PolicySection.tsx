import type { PolicyFormData } from '@/lib/types'

interface Props {
  policy: PolicyFormData | undefined
  onChange: (policy: PolicyFormData | undefined) => void
}

export function PolicySection({ policy, onChange }: Props) {
  const ensurePolicy = (): PolicyFormData => policy ?? {}

  const toggleRateLimit = () => {
    const p = ensurePolicy()
    if (p.rate_limit) {
      const { rate_limit: _, ...rest } = p
      onChange(Object.keys(rest).length > 0 ? rest : undefined)
    } else {
      onChange({ ...p, rate_limit: { rate: 10, burst: 20, key: '' } })
    }
  }

  const toggleLatency = () => {
    const p = ensurePolicy()
    if (p.latency) {
      const { latency: _, ...rest } = p
      onChange(Object.keys(rest).length > 0 ? rest : undefined)
    } else {
      onChange({ ...p, latency: { fixed_ms: 100, jitter_ms: 50 } })
    }
  }

  const togglePagination = () => {
    const p = ensurePolicy()
    if (p.pagination) {
      const { pagination: _, ...rest } = p
      onChange(Object.keys(rest).length > 0 ? rest : undefined)
    } else {
      onChange({ ...p, pagination: { style: 'page_size', default_size: 10, max_size: 100, data_path: '$' } })
    }
  }

  return (
    <div className="border border-[hsl(var(--border))] rounded-lg p-4">
      <h3 className="text-sm font-medium text-[hsl(var(--muted-foreground))] mb-3">Policy (optional)</h3>
      <div className="grid gap-4">
        {/* Rate Limit */}
        <div>
          <label className="flex items-center gap-2 mb-2">
            <input type="checkbox" checked={!!policy?.rate_limit} onChange={toggleRateLimit} className="rounded" />
            <span className="text-sm font-medium text-[hsl(var(--foreground))]">Rate Limit</span>
          </label>
          {policy?.rate_limit && (
            <div className="ml-5 grid grid-cols-3 gap-3">
              <div>
                <label className="text-xs text-[hsl(var(--muted-foreground))] mb-1 block">Rate (req/s)</label>
                <input
                  type="number"
                  min={0}
                  step={0.1}
                  value={policy.rate_limit.rate}
                  onChange={e => onChange({ ...policy, rate_limit: { ...policy.rate_limit!, rate: parseFloat(e.target.value) || 0 } })}
                  className="w-full px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
                />
              </div>
              <div>
                <label className="text-xs text-[hsl(var(--muted-foreground))] mb-1 block">Burst</label>
                <input
                  type="number"
                  min={0}
                  value={policy.rate_limit.burst}
                  onChange={e => onChange({ ...policy, rate_limit: { ...policy.rate_limit!, burst: parseInt(e.target.value) || 0 } })}
                  className="w-full px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
                />
              </div>
              <div>
                <label className="text-xs text-[hsl(var(--muted-foreground))] mb-1 block">Key (optional)</label>
                <input
                  type="text"
                  value={policy.rate_limit.key}
                  onChange={e => onChange({ ...policy, rate_limit: { ...policy.rate_limit!, key: e.target.value } })}
                  placeholder="ip, header:X-Api-Key"
                  className="w-full px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
                />
              </div>
            </div>
          )}
        </div>

        {/* Latency */}
        <div>
          <label className="flex items-center gap-2 mb-2">
            <input type="checkbox" checked={!!policy?.latency} onChange={toggleLatency} className="rounded" />
            <span className="text-sm font-medium text-[hsl(var(--foreground))]">Latency Simulation</span>
          </label>
          {policy?.latency && (
            <div className="ml-5 grid grid-cols-2 gap-3">
              <div>
                <label className="text-xs text-[hsl(var(--muted-foreground))] mb-1 block">Fixed (ms)</label>
                <input
                  type="number"
                  min={0}
                  value={policy.latency.fixed_ms}
                  onChange={e => onChange({ ...policy, latency: { ...policy.latency!, fixed_ms: parseInt(e.target.value) || 0 } })}
                  className="w-full px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
                />
              </div>
              <div>
                <label className="text-xs text-[hsl(var(--muted-foreground))] mb-1 block">Jitter (ms)</label>
                <input
                  type="number"
                  min={0}
                  value={policy.latency.jitter_ms}
                  onChange={e => onChange({ ...policy, latency: { ...policy.latency!, jitter_ms: parseInt(e.target.value) || 0 } })}
                  className="w-full px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
                />
              </div>
            </div>
          )}
        </div>

        {/* Pagination */}
        <div>
          <label className="flex items-center gap-2 mb-2">
            <input type="checkbox" checked={!!policy?.pagination} onChange={togglePagination} className="rounded" />
            <span className="text-sm font-medium text-[hsl(var(--foreground))]">Pagination</span>
          </label>
          {policy?.pagination && (
            <div className="ml-5 grid grid-cols-2 gap-3">
              <div>
                <label className="text-xs text-[hsl(var(--muted-foreground))] mb-1 block">Style</label>
                <select
                  value={policy.pagination.style}
                  onChange={e => onChange({ ...policy, pagination: { ...policy.pagination!, style: e.target.value as 'page_size' | 'offset_limit' } })}
                  className="w-full px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
                >
                  <option value="page_size">Page + Size</option>
                  <option value="offset_limit">Offset + Limit</option>
                </select>
              </div>
              <div>
                <label className="text-xs text-[hsl(var(--muted-foreground))] mb-1 block">Data Path</label>
                <input
                  type="text"
                  value={policy.pagination.data_path}
                  onChange={e => onChange({ ...policy, pagination: { ...policy.pagination!, data_path: e.target.value } })}
                  placeholder="$ (root array)"
                  className="w-full px-2.5 py-1.5 text-sm font-mono rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
                />
              </div>
              <div>
                <label className="text-xs text-[hsl(var(--muted-foreground))] mb-1 block">Default Size</label>
                <input
                  type="number"
                  min={1}
                  value={policy.pagination.default_size}
                  onChange={e => onChange({ ...policy, pagination: { ...policy.pagination!, default_size: parseInt(e.target.value) || 10 } })}
                  className="w-full px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
                />
              </div>
              <div>
                <label className="text-xs text-[hsl(var(--muted-foreground))] mb-1 block">Max Size</label>
                <input
                  type="number"
                  min={1}
                  value={policy.pagination.max_size}
                  onChange={e => onChange({ ...policy, pagination: { ...policy.pagination!, max_size: parseInt(e.target.value) || 100 } })}
                  className="w-full px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
                />
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
