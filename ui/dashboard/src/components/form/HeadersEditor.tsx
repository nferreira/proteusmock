import { Plus, X } from 'lucide-react'
import type { KeyValuePair } from '@/lib/types'

interface Props {
  label: string
  headers: KeyValuePair[]
  onChange: (headers: KeyValuePair[]) => void
  keyPlaceholder?: string
  valuePlaceholder?: string
}

export function HeadersEditor({ label, headers, onChange, keyPlaceholder = 'Header name', valuePlaceholder = 'Value' }: Props) {
  const addRow = () => onChange([...headers, { key: '', value: '' }])

  const updateRow = (index: number, field: 'key' | 'value', val: string) => {
    const updated = headers.map((h, i) => (i === index ? { ...h, [field]: val } : h))
    onChange(updated)
  }

  const removeRow = (index: number) => onChange(headers.filter((_, i) => i !== index))

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <label className="text-sm font-medium text-[hsl(var(--foreground))]">{label}</label>
        <button
          type="button"
          onClick={addRow}
          className="flex items-center gap-1 text-xs text-[hsl(var(--primary))] cursor-pointer border-none bg-transparent hover:underline"
        >
          <Plus size={12} /> Add
        </button>
      </div>
      {headers.length === 0 && (
        <p className="text-xs text-[hsl(var(--muted-foreground))] italic">None</p>
      )}
      <div className="grid gap-1.5">
        {headers.map((h, i) => (
          <div key={i} className="flex gap-1.5 items-center">
            <input
              type="text"
              value={h.key}
              onChange={e => updateRow(i, 'key', e.target.value)}
              placeholder={keyPlaceholder}
              className="flex-1 px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
            />
            <input
              type="text"
              value={h.value}
              onChange={e => updateRow(i, 'value', e.target.value)}
              placeholder={valuePlaceholder}
              className="flex-1 px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
            />
            <button
              type="button"
              onClick={() => removeRow(i)}
              className="p-1 rounded-md text-[hsl(var(--muted-foreground))] hover:text-[hsl(var(--destructive))] cursor-pointer border-none bg-transparent"
            >
              <X size={14} />
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
