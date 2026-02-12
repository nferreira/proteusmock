import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import CodeMirror from '@uiw/react-codemirror'
import { yaml } from '@codemirror/lang-yaml'
import { ArrowLeft, Plus } from 'lucide-react'
import { api } from '@/lib/api'
import { cn } from '@/lib/utils'
import { defaultFormData } from '@/lib/yaml-parser'
import { ScenarioForm } from '@/components/form/ScenarioForm'

const TEMPLATE = `id: new-scenario
name: New Scenario
priority: 10
when:
  method: GET
  path: /api/v1/example
response:
  status: 200
  headers:
    Content-Type: application/json
  body: '{"message": "hello"}'
`

type EditorMode = 'form' | 'yaml'

export function CreateScenarioPage() {
  const navigate = useNavigate()
  const [mode, setMode] = useState<EditorMode>('form')
  const [yamlValue, setYamlValue] = useState(TEMPLATE)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')

  const handleYamlCreate = async () => {
    setCreating(true)
    setError('')
    try {
      await api.createScenario(yamlValue)
      navigate('/')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Create failed')
    } finally {
      setCreating(false)
    }
  }

  const handleFormCreate = async (yamlContent: string) => {
    setCreating(true)
    setError('')
    try {
      await api.createScenario(yamlContent)
      navigate('/')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Create failed')
    } finally {
      setCreating(false)
    }
  }

  const handleModeSwitch = (newMode: EditorMode) => {
    if (newMode === mode) return
    // When switching from form to YAML, we lose the form state (user should save first or accept reset)
    setMode(newMode)
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
          <h1 className="text-2xl font-semibold">New Scenario</h1>
        </div>
        {mode === 'yaml' && (
          <button
            onClick={handleYamlCreate}
            disabled={creating}
            className={cn(
              'flex items-center gap-1.5 px-4 py-2 rounded-md text-sm cursor-pointer border-none',
              'bg-[hsl(var(--primary))] text-[hsl(var(--primary-foreground))] hover:opacity-90 transition-opacity',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
          >
            <Plus size={14} />
            {creating ? 'Creating...' : 'Create Scenario'}
          </button>
        )}
      </div>

      {error && (
        <div className="bg-red-50 text-red-700 px-4 py-3 rounded-md mb-4 text-sm">{error}</div>
      )}

      <div className="flex gap-1 mb-4 border-b border-[hsl(var(--border))]">
        {(['form', 'yaml'] as EditorMode[]).map(m => (
          <button
            key={m}
            onClick={() => handleModeSwitch(m)}
            className={cn(
              'px-4 py-2 text-sm border-none bg-transparent cursor-pointer transition-colors',
              mode === m
                ? 'border-b-2 border-b-[hsl(var(--primary))] text-[hsl(var(--foreground))] font-medium -mb-px'
                : 'text-[hsl(var(--muted-foreground))] hover:text-[hsl(var(--foreground))]',
            )}
          >
            {m === 'form' ? 'Form Editor' : 'YAML Editor'}
          </button>
        ))}
      </div>

      {mode === 'form' ? (
        <ScenarioForm
          initialData={defaultFormData()}
          onSubmit={handleFormCreate}
          submitLabel="Create Scenario"
          isSubmitting={creating}
          isNew={true}
        />
      ) : (
        <div className="border border-[hsl(var(--border))] rounded-lg overflow-hidden">
          <CodeMirror
            value={yamlValue}
            onChange={setYamlValue}
            extensions={[yaml()]}
            height="600px"
            theme="light"
          />
        </div>
      )}
    </div>
  )
}
