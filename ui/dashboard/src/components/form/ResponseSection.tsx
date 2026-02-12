import { useState, useEffect } from 'react'
import type { ResponseFormData, BodyMode } from '@/lib/types'
import { HeadersEditor } from './HeadersEditor'
import { api } from '@/lib/api'

interface Props {
  response: ResponseFormData
  onChange: (response: ResponseFormData) => void
  bodyMode: BodyMode
  onBodyModeChange: (mode: BodyMode) => void
}

export function ResponseSection({ response, onChange, bodyMode, onBodyModeChange }: Props) {
  const [files, setFiles] = useState<string[]>([])

  useEffect(() => {
    api.listFiles().then(setFiles).catch(() => setFiles([]))
  }, [])

  return (
    <div className="border border-[hsl(var(--border))] rounded-lg p-4">
      <h3 className="text-sm font-medium text-[hsl(var(--muted-foreground))] mb-3">Response</h3>
      <div className="grid gap-4">
        <div className="grid grid-cols-3 gap-3">
          <div>
            <label className="text-sm font-medium text-[hsl(var(--foreground))] mb-1 block">Status</label>
            <input
              type="number"
              min={100}
              max={599}
              value={response.status}
              onChange={e => onChange({ ...response, status: parseInt(e.target.value) || 200 })}
              className="w-full px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
            />
          </div>
          <div>
            <label className="text-sm font-medium text-[hsl(var(--foreground))] mb-1 block">Content-Type</label>
            <input
              type="text"
              value={response.content_type}
              onChange={e => onChange({ ...response, content_type: e.target.value })}
              placeholder="application/json"
              list="content-types"
              className="w-full px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
            />
            <datalist id="content-types">
              <option value="application/json" />
              <option value="application/xml" />
              <option value="text/html" />
              <option value="text/plain" />
            </datalist>
          </div>
          <div>
            <label className="text-sm font-medium text-[hsl(var(--foreground))] mb-1 block">Engine</label>
            <select
              value={response.engine}
              onChange={e => onChange({ ...response, engine: e.target.value as ResponseFormData['engine'] })}
              className="w-full px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
            >
              <option value="">Static (none)</option>
              <option value="expr">Expr ($&#123; &#125;)</option>
              <option value="jinja2">Jinja2 (&#123;&#123; &#125;&#125;)</option>
            </select>
          </div>
        </div>

        <HeadersEditor
          label="Response Headers"
          headers={response.headers}
          onChange={headers => onChange({ ...response, headers })}
        />

        <div>
          <div className="flex items-center gap-3 mb-2">
            <label className="text-sm font-medium text-[hsl(var(--foreground))]">Body</label>
            <div className="flex gap-1 bg-[hsl(var(--muted))] rounded-md p-0.5">
              {(['inline', 'include'] as BodyMode[]).map(mode => (
                <button
                  key={mode}
                  type="button"
                  onClick={() => onBodyModeChange(mode)}
                  className={`px-2 py-0.5 text-xs rounded cursor-pointer border-none transition-colors ${
                    bodyMode === mode
                      ? 'bg-[hsl(var(--background))] text-[hsl(var(--foreground))] shadow-sm'
                      : 'bg-transparent text-[hsl(var(--muted-foreground))]'
                  }`}
                >
                  {mode === 'inline' ? 'Inline' : 'Include'}
                </button>
              ))}
            </div>
          </div>

          {bodyMode === 'inline' && (
            <textarea
              value={response.body}
              onChange={e => onChange({ ...response, body: e.target.value })}
              placeholder='{"message": "hello"}'
              rows={8}
              className="w-full px-2.5 py-1.5 text-sm font-mono rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))] resize-y"
            />
          )}

          {bodyMode === 'include' && (
            <div>
              <input
                type="text"
                value={response.include_file}
                onChange={e => onChange({ ...response, include_file: e.target.value })}
                placeholder="responses/data.json"
                list="include-file-suggestions"
                className="w-full px-2.5 py-1.5 text-sm font-mono rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
              />
              <datalist id="include-file-suggestions">
                {files.map(f => (
                  <option key={f} value={f} />
                ))}
              </datalist>
              <p className="text-xs text-[hsl(var(--muted-foreground))] mt-1">
                File path for <code className="text-xs">!include</code> directive. Supports relative paths, <code className="text-xs">@root/</code>, and <code className="text-xs">@here/</code> prefixes.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
