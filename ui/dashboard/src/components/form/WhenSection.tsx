import type { WhenFormData } from '@/lib/types'
import { HeadersEditor } from './HeadersEditor'
import { BodyMatchingEditor } from './BodyMatchingEditor'

const METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'] as const

interface Props {
  when: WhenFormData
  onChange: (when: WhenFormData) => void
}

export function WhenSection({ when, onChange }: Props) {
  return (
    <div className="border border-[hsl(var(--border))] rounded-lg p-4">
      <h3 className="text-sm font-medium text-[hsl(var(--muted-foreground))] mb-3">Request Matching</h3>
      <div className="grid gap-4">
        <div className="grid grid-cols-[140px_1fr] gap-3">
          <div>
            <label className="text-sm font-medium text-[hsl(var(--foreground))] mb-1 block">Method</label>
            <select
              value={when.method}
              onChange={e => onChange({ ...when, method: e.target.value })}
              className="w-full px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
            >
              {METHODS.map(m => (
                <option key={m} value={m}>{m}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-sm font-medium text-[hsl(var(--foreground))] mb-1 block">Path</label>
            <input
              type="text"
              value={when.path}
              onChange={e => onChange({ ...when, path: e.target.value })}
              placeholder="/api/v1/example/{id}"
              className="w-full px-2.5 py-1.5 text-sm font-mono rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
            />
          </div>
        </div>

        <HeadersEditor
          label="Headers"
          headers={when.headers}
          onChange={headers => onChange({ ...when, headers })}
          keyPlaceholder="Header name"
          valuePlaceholder="=exact or /regex/"
        />

        <BodyMatchingEditor
          body={when.body}
          onChange={body => onChange({ ...when, body })}
        />
      </div>
    </div>
  )
}
