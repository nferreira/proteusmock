import { useState } from 'react'
import { Save, Plus, AlertTriangle } from 'lucide-react'
import type { ScenarioFormData, BodyMode } from '@/lib/types'
import { formDataToYaml, detectBodyMode } from '@/lib/yaml-serializer'
import { WhenSection } from './WhenSection'
import { ResponseSection } from './ResponseSection'
import { PolicySection } from './PolicySection'
import { cn } from '@/lib/utils'

interface Props {
  initialData: ScenarioFormData
  onSubmit: (yamlContent: string) => Promise<void>
  submitLabel: string
  isSubmitting: boolean
  isNew: boolean
  warnings?: string[]
}

export function ScenarioForm({ initialData, onSubmit, submitLabel, isSubmitting, isNew, warnings }: Props) {
  const [data, setData] = useState<ScenarioFormData>(initialData)
  const [bodyMode, setBodyMode] = useState<BodyMode>(() => {
    const detected = detectBodyMode(initialData)
    return detected === 'file' ? 'inline' : detected
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const yamlContent = formDataToYaml(data, bodyMode)
    await onSubmit(yamlContent)
  }

  return (
    <form onSubmit={handleSubmit} className="grid gap-4">
      {warnings && warnings.length > 0 && (
        <div className="flex items-start gap-2 bg-amber-50 border border-amber-200 text-amber-800 px-4 py-3 rounded-md text-sm">
          <AlertTriangle size={16} className="mt-0.5 shrink-0" />
          <div>
            {warnings.map((w, i) => (
              <p key={i} className="m-0">{w}</p>
            ))}
            <p className="m-0 mt-1 font-medium">Use the YAML Editor if you need to adjust these manually.</p>
          </div>
        </div>
      )}

      {/* General fields */}
      <div className="border border-[hsl(var(--border))] rounded-lg p-4">
        <h3 className="text-sm font-medium text-[hsl(var(--muted-foreground))] mb-3">General</h3>
        <div className="grid grid-cols-3 gap-3">
          <div>
            <label className="text-sm font-medium text-[hsl(var(--foreground))] mb-1 block">ID</label>
            <input
              type="text"
              value={data.id}
              onChange={e => setData({ ...data, id: e.target.value })}
              readOnly={!isNew}
              placeholder="my-scenario-id"
              className={cn(
                'w-full px-2.5 py-1.5 text-sm font-mono rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]',
                !isNew && 'bg-[hsl(var(--muted))] text-[hsl(var(--muted-foreground))] cursor-not-allowed',
              )}
            />
          </div>
          <div>
            <label className="text-sm font-medium text-[hsl(var(--foreground))] mb-1 block">Name</label>
            <input
              type="text"
              value={data.name}
              onChange={e => setData({ ...data, name: e.target.value })}
              placeholder="My Scenario"
              className="w-full px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
            />
          </div>
          <div>
            <label className="text-sm font-medium text-[hsl(var(--foreground))] mb-1 block">Priority</label>
            <input
              type="number"
              value={data.priority}
              onChange={e => setData({ ...data, priority: parseInt(e.target.value) || 10 })}
              min={0}
              className="w-full px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
            />
          </div>
        </div>
      </div>

      <WhenSection when={data.when} onChange={when => setData({ ...data, when })} />
      <ResponseSection
        response={data.response}
        onChange={response => setData({ ...data, response })}
        bodyMode={bodyMode}
        onBodyModeChange={setBodyMode}
      />
      <PolicySection policy={data.policy} onChange={policy => setData({ ...data, policy })} />

      <div className="flex justify-end">
        <button
          type="submit"
          disabled={isSubmitting}
          className={cn(
            'flex items-center gap-1.5 px-4 py-2 rounded-md text-sm cursor-pointer border-none',
            'bg-[hsl(var(--primary))] text-[hsl(var(--primary-foreground))] hover:opacity-90 transition-opacity',
            'disabled:opacity-50 disabled:cursor-not-allowed',
          )}
        >
          {isNew ? <Plus size={14} /> : <Save size={14} />}
          {isSubmitting ? 'Saving...' : submitLabel}
        </button>
      </div>
    </form>
  )
}
