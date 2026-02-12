import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import CodeMirror from '@uiw/react-codemirror'
import { yaml } from '@codemirror/lang-yaml'
import { ArrowLeft, Save, Trash2 } from 'lucide-react'
import { api, type ScenarioDetail } from '@/lib/api'
import { cn } from '@/lib/utils'
import { yamlToFormData, type ParseResult } from '@/lib/yaml-parser'
import { ScenarioForm } from '@/components/form/ScenarioForm'

type Tab = 'overview' | 'form' | 'editor'

export function ScenarioDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [scenario, setScenario] = useState<ScenarioDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [tab, setTab] = useState<Tab>('overview')
  const [editorValue, setEditorValue] = useState('')
  const [saving, setSaving] = useState(false)
  const [saveMessage, setSaveMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [formParseResult, setFormParseResult] = useState<ParseResult | null>(null)

  const load = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError('')
    try {
      const data = await api.getScenario(id)
      setScenario(data)
      setEditorValue(data.source_yaml || '')
      if (data.source_yaml) {
        setFormParseResult(yamlToFormData(data.source_yaml))
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load scenario')
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => { load() }, [load])

  const handleSave = async () => {
    if (!id) return
    setSaving(true)
    setSaveMessage(null)
    try {
      await api.updateScenario(id, editorValue)
      setSaveMessage({ type: 'success', text: 'Scenario saved and reloaded' })
      await load()
    } catch (e) {
      setSaveMessage({ type: 'error', text: e instanceof Error ? e.message : 'Save failed' })
    } finally {
      setSaving(false)
    }
  }

  const handleFormSave = async (yamlContent: string) => {
    if (!id) return
    setSaveMessage(null)
    try {
      await api.updateScenario(id, yamlContent)
      setSaveMessage({ type: 'success', text: 'Scenario saved and reloaded' })
      await load()
    } catch (e) {
      setSaveMessage({ type: 'error', text: e instanceof Error ? e.message : 'Save failed' })
    }
  }

  const handleDelete = async () => {
    if (!id) return
    try {
      await api.deleteScenario(id)
      navigate('/')
    } catch (e) {
      setSaveMessage({ type: 'error', text: e instanceof Error ? e.message : 'Delete failed' })
    }
    setShowDeleteConfirm(false)
  }

  if (loading) {
    return <div className="py-12 text-center text-[hsl(var(--muted-foreground))]">Loading...</div>
  }
  if (error) {
    return <div className="bg-red-50 text-red-700 px-4 py-3 rounded-md text-sm">{error}</div>
  }
  if (!scenario) return null

  const tabLabels: Record<Tab, string> = {
    overview: 'Overview',
    form: 'Form Editor',
    editor: 'YAML Editor',
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate('/')}
            className="p-1.5 rounded-md hover:bg-[hsl(var(--accent))] transition-colors cursor-pointer border-none bg-transparent"
          >
            <ArrowLeft size={20} />
          </button>
          <div>
            <h1 className="text-2xl font-semibold">{scenario.name}</h1>
            <p className="text-sm text-[hsl(var(--muted-foreground))] font-mono">{scenario.id}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {tab === 'editor' && (
            <button
              onClick={handleSave}
              disabled={saving}
              className={cn(
                'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm cursor-pointer border-none',
                'bg-[hsl(var(--primary))] text-[hsl(var(--primary-foreground))] hover:opacity-90 transition-opacity',
                'disabled:opacity-50 disabled:cursor-not-allowed',
              )}
            >
              <Save size={14} />
              {saving ? 'Saving...' : 'Save'}
            </button>
          )}
          <button
            onClick={() => setShowDeleteConfirm(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm cursor-pointer border border-[hsl(var(--destructive))] text-[hsl(var(--destructive))] bg-white hover:bg-red-50 transition-colors"
          >
            <Trash2 size={14} />
            Delete
          </button>
        </div>
      </div>

      {saveMessage && (
        <div
          className={cn(
            'px-4 py-3 rounded-md mb-4 text-sm',
            saveMessage.type === 'success' ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-700',
          )}
        >
          {saveMessage.text}
        </div>
      )}

      {showDeleteConfirm && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4 mb-4 flex items-center justify-between">
          <span className="text-sm text-red-700">
            Delete scenario <strong>{scenario.id}</strong>? This cannot be undone.
          </span>
          <div className="flex gap-2">
            <button
              onClick={() => setShowDeleteConfirm(false)}
              className="px-3 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-white cursor-pointer hover:bg-[hsl(var(--accent))]"
            >
              Cancel
            </button>
            <button
              onClick={handleDelete}
              className="px-3 py-1.5 text-sm rounded-md border-none bg-[hsl(var(--destructive))] text-white cursor-pointer hover:opacity-90"
            >
              Confirm Delete
            </button>
          </div>
        </div>
      )}

      <div className="flex gap-1 mb-4 border-b border-[hsl(var(--border))]">
        {(['overview', 'form', 'editor'] as Tab[]).map(t => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={cn(
              'px-4 py-2 text-sm border-none bg-transparent cursor-pointer transition-colors',
              tab === t
                ? 'border-b-2 border-b-[hsl(var(--primary))] text-[hsl(var(--foreground))] font-medium -mb-px'
                : 'text-[hsl(var(--muted-foreground))] hover:text-[hsl(var(--foreground))]',
            )}
          >
            {tabLabels[t]}
          </button>
        ))}
      </div>

      {tab === 'overview' && (
        <ScenarioOverview
          scenario={scenario}
          includeFile={formParseResult?.data.response.include_file}
        />
      )}

      {tab === 'form' && formParseResult && (
        <ScenarioForm
          initialData={formParseResult.data}
          onSubmit={handleFormSave}
          submitLabel="Save"
          isSubmitting={saving}
          isNew={false}
          warnings={formParseResult.warnings.length > 0 ? formParseResult.warnings : undefined}
        />
      )}

      {tab === 'editor' && (
        <div className="border border-[hsl(var(--border))] rounded-lg overflow-hidden">
          <CodeMirror
            value={editorValue}
            onChange={setEditorValue}
            extensions={[yaml()]}
            height="600px"
            theme="light"
          />
        </div>
      )}
    </div>
  )
}

function ScenarioOverview({ scenario, includeFile }: { scenario: ScenarioDetail; includeFile?: string }) {
  return (
    <div className="grid gap-4">
      <Section title="General">
        <Field label="ID" value={scenario.id} mono />
        <Field label="Name" value={scenario.name} />
        <Field label="Priority" value={String(scenario.priority)} />
        <Field label="Source File" value={scenario.source_file} mono />
      </Section>

      <Section title="When (Request Matching)">
        <Field label="Method" value={scenario.when.method} />
        <Field label="Path" value={scenario.when.path} mono />
        {scenario.when.headers && (
          <Field
            label="Headers"
            value={Object.entries(scenario.when.headers)
              .map(([k, v]) => `${k}: ${v}`)
              .join('\n')}
            mono
            pre
          />
        )}
        {scenario.when.body && (
          <Field label="Body Matching" value={JSON.stringify(scenario.when.body, null, 2)} mono pre />
        )}
      </Section>

      <Section title="Response">
        <Field label="Status" value={String(scenario.response.status)} />
        {scenario.response.engine && <Field label="Engine" value={scenario.response.engine} />}
        {scenario.response.content_type && (
          <Field label="Content-Type" value={scenario.response.content_type} mono />
        )}
        {scenario.response.headers && (
          <Field
            label="Headers"
            value={Object.entries(scenario.response.headers)
              .map(([k, v]) => `${k}: ${v}`)
              .join('\n')}
            mono
            pre
          />
        )}
        {scenario.response.body && (
          <Field
            label="Body"
            value={scenario.response.body}
            mono
            pre
            badge={includeFile ? `!include ${includeFile}` : undefined}
          />
        )}
        {scenario.response.body_file && (
          <Field label="Body File" value={scenario.response.body_file} mono />
        )}
      </Section>

      {scenario.policy && (
        <Section title="Policy">
          {scenario.policy.rate_limit && (
            <Field
              label="Rate Limit"
              value={`${scenario.policy.rate_limit.rate} req/s, burst ${scenario.policy.rate_limit.burst}${scenario.policy.rate_limit.key ? `, key: ${scenario.policy.rate_limit.key}` : ''}`}
            />
          )}
          {scenario.policy.latency && (
            <Field
              label="Latency"
              value={`${scenario.policy.latency.fixed_ms}ms fixed + ${scenario.policy.latency.jitter_ms}ms jitter`}
            />
          )}
          {scenario.policy.pagination && (
            <Field
              label="Pagination"
              value={`${scenario.policy.pagination.style}, default ${scenario.policy.pagination.default_size}, max ${scenario.policy.pagination.max_size}`}
            />
          )}
        </Section>
      )}
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="border border-[hsl(var(--border))] rounded-lg p-4">
      <h3 className="text-sm font-medium text-[hsl(var(--muted-foreground))] mb-3">{title}</h3>
      <div className="grid gap-2">{children}</div>
    </div>
  )
}

function Field({
  label,
  value,
  mono,
  pre,
  badge,
}: {
  label: string
  value: string
  mono?: boolean
  pre?: boolean
  badge?: string
}) {
  return (
    <div className="flex gap-4">
      <span className="text-sm text-[hsl(var(--muted-foreground))] w-28 shrink-0">{label}</span>
      <div className="min-w-0 flex-1">
        {badge && (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 mb-1.5 text-xs font-mono rounded-md bg-violet-100 text-violet-700 border border-violet-200">
            {badge}
          </span>
        )}
        {pre ? (
          <pre className={cn('text-sm whitespace-pre-wrap break-all m-0', mono && 'font-mono text-xs')}>
            {value}
          </pre>
        ) : (
          <span className={cn('text-sm', mono && 'font-mono text-xs')}>{value}</span>
        )}
      </div>
    </div>
  )
}
