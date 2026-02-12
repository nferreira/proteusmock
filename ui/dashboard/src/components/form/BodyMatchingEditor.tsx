import { Plus, X } from 'lucide-react'
import type { BodyFormData, BodyNodeFormData, BodyNodeType } from '@/lib/types'
import { defaultBodyNode, uid } from '@/lib/yaml-parser'

const MAX_DEPTH = 10

interface Props {
  body: BodyFormData | undefined
  onChange: (body: BodyFormData | undefined) => void
}

export function BodyMatchingEditor({ body, onChange }: Props) {
  const enabled = body !== undefined

  const toggle = () => {
    if (enabled) {
      onChange(undefined)
    } else {
      onChange({ content_type: 'json', node: defaultBodyNode() })
    }
  }

  const updateContentType = (ct: string) => {
    if (!body) return
    onChange({ ...body, content_type: ct })
  }

  const updateNode = (node: BodyNodeFormData) => {
    if (!body) return
    onChange({ ...body, node })
  }

  return (
    <div>
      <label className="flex items-center gap-2 mb-2">
        <input type="checkbox" checked={enabled} onChange={toggle} className="rounded" />
        <span className="text-sm font-medium text-[hsl(var(--foreground))]">Body Matching</span>
      </label>

      {enabled && body && (
        <div className="ml-5 grid gap-3">
          <div>
            <label className="text-xs text-[hsl(var(--muted-foreground))] mb-1 block">Content Type</label>
            <select
              value={body.content_type}
              onChange={e => updateContentType(e.target.value)}
              className="px-2.5 py-1.5 text-sm rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
            >
              <option value="json">JSON (JSONPath)</option>
              <option value="xml">XML (XPath)</option>
            </select>
          </div>

          <BodyNodeEditor
            node={body.node}
            onChange={updateNode}
            contentType={body.content_type}
            depth={0}
            canDelete={false}
          />
        </div>
      )}
    </div>
  )
}

const BORDER_COLORS: Record<BodyNodeType, string> = {
  conditions: 'border-l-gray-300',
  all: 'border-l-blue-400',
  any: 'border-l-green-400',
  not: 'border-l-red-400',
}

const BADGE_COLORS: Record<BodyNodeType, string> = {
  conditions: 'bg-gray-100 text-gray-700',
  all: 'bg-blue-100 text-blue-700',
  any: 'bg-green-100 text-green-700',
  not: 'bg-red-100 text-red-700',
}

const BADGE_LABELS: Record<BodyNodeType, string> = {
  conditions: 'CONDITIONS',
  all: 'ALL (AND)',
  any: 'ANY (OR)',
  not: 'NOT',
}

interface NodeProps {
  node: BodyNodeFormData
  onChange: (node: BodyNodeFormData) => void
  contentType: string
  depth: number
  canDelete: boolean
  onDelete?: () => void
}

function BodyNodeEditor({ node, onChange, contentType, depth, canDelete, onDelete }: NodeProps) {
  if (depth >= MAX_DEPTH) {
    return (
      <div className="text-xs text-amber-600 border border-amber-200 bg-amber-50 p-2 rounded ml-4">
        Maximum nesting depth reached.
      </div>
    )
  }

  const changeType = (newType: BodyNodeType) => {
    if (newType === node.type) return
    switch (newType) {
      case 'conditions':
        onChange({ id: node.id, type: 'conditions', conditions: [{ id: uid(), extractor: '', matcher: '' }], children: [] })
        break
      case 'all':
      case 'any':
        onChange({ id: node.id, type: newType, conditions: [], children: [defaultBodyNode()] })
        break
      case 'not':
        onChange({ id: node.id, type: 'not', conditions: [], children: [], child: defaultBodyNode() })
        break
    }
  }

  return (
    <div className={`border-l-2 ${BORDER_COLORS[node.type]} pl-3 ${depth > 0 ? 'ml-4' : ''}`}>
      <div className="flex items-center gap-2 mb-2">
        <span className={`text-[10px] font-semibold px-1.5 py-0.5 rounded ${BADGE_COLORS[node.type]}`}>
          {BADGE_LABELS[node.type]}
        </span>
        <select
          value={node.type}
          onChange={e => changeType(e.target.value as BodyNodeType)}
          className="px-1.5 py-0.5 text-xs rounded border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none"
        >
          <option value="conditions">Conditions</option>
          <option value="all">ALL (AND)</option>
          <option value="any">ANY (OR)</option>
          <option value="not">NOT</option>
        </select>
        {canDelete && onDelete && (
          <button
            type="button"
            onClick={onDelete}
            aria-label="Remove clause"
            className="p-0.5 rounded text-[hsl(var(--muted-foreground))] hover:text-[hsl(var(--destructive))] cursor-pointer border-none bg-transparent"
          >
            <X size={14} />
          </button>
        )}
      </div>

      {node.type === 'conditions' && (
        <ConditionsEditor node={node} onChange={onChange} contentType={contentType} />
      )}

      {(node.type === 'all' || node.type === 'any') && (
        <ChildrenEditor node={node} onChange={onChange} contentType={contentType} depth={depth} />
      )}

      {node.type === 'not' && (
        <NotEditor node={node} onChange={onChange} contentType={contentType} depth={depth} />
      )}
    </div>
  )
}

function ConditionsEditor({
  node,
  onChange,
  contentType,
}: {
  node: BodyNodeFormData
  onChange: (node: BodyNodeFormData) => void
  contentType: string
}) {
  const addCondition = () => {
    onChange({ ...node, conditions: [...node.conditions, { id: uid(), extractor: '', matcher: '' }] })
  }

  const updateCondition = (index: number, field: 'extractor' | 'matcher', val: string) => {
    const updated = node.conditions.map((c, i) => (i === index ? { ...c, [field]: val } : c))
    onChange({ ...node, conditions: updated })
  }

  const removeCondition = (index: number) => {
    onChange({ ...node, conditions: node.conditions.filter((_, i) => i !== index) })
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-1.5">
        <label className="text-xs text-[hsl(var(--muted-foreground))]">Extractor / Matcher pairs</label>
        <button
          type="button"
          onClick={addCondition}
          className="flex items-center gap-1 text-xs text-[hsl(var(--primary))] cursor-pointer border-none bg-transparent hover:underline"
        >
          <Plus size={12} /> Add
        </button>
      </div>
      <div className="grid gap-1.5">
        {node.conditions.map((c, i) => (
          <div key={c.id} className="flex gap-1.5 items-center">
            <input
              type="text"
              value={c.extractor}
              onChange={e => updateCondition(i, 'extractor', e.target.value)}
              placeholder={contentType === 'xml' ? '//xpath/expression' : '$.json.path'}
              className="flex-1 px-2.5 py-1.5 text-sm font-mono rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
            />
            <input
              type="text"
              value={c.matcher}
              onChange={e => updateCondition(i, 'matcher', e.target.value)}
              placeholder="expected value or /regex/"
              className="flex-1 px-2.5 py-1.5 text-sm font-mono rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] focus:outline-none focus:ring-1 focus:ring-[hsl(var(--ring))]"
            />
            <button
              type="button"
              onClick={() => removeCondition(i)}
              aria-label="Remove condition"
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

function ChildrenEditor({
  node,
  onChange,
  contentType,
  depth,
}: {
  node: BodyNodeFormData
  onChange: (node: BodyNodeFormData) => void
  contentType: string
  depth: number
}) {
  const addChild = () => {
    onChange({ ...node, children: [...node.children, defaultBodyNode()] })
  }

  const updateChild = (index: number, child: BodyNodeFormData) => {
    const updated = node.children.map((c, i) => (i === index ? child : c))
    onChange({ ...node, children: updated })
  }

  const removeChild = (index: number) => {
    onChange({ ...node, children: node.children.filter((_, i) => i !== index) })
  }

  return (
    <div className="grid gap-2">
      {node.children.map((child, i) => (
        <BodyNodeEditor
          key={child.id}
          node={child}
          onChange={updated => updateChild(i, updated)}
          contentType={contentType}
          depth={depth + 1}
          canDelete={node.children.length > 1}
          onDelete={() => removeChild(i)}
        />
      ))}
      <button
        type="button"
        onClick={addChild}
        className="flex items-center gap-1 text-xs text-[hsl(var(--primary))] cursor-pointer border-none bg-transparent hover:underline w-fit"
      >
        <Plus size={12} /> Add clause
      </button>
    </div>
  )
}

function NotEditor({
  node,
  onChange,
  contentType,
  depth,
}: {
  node: BodyNodeFormData
  onChange: (node: BodyNodeFormData) => void
  contentType: string
  depth: number
}) {
  const child = node.child ?? defaultBodyNode()

  const updateChild = (updated: BodyNodeFormData) => {
    onChange({ ...node, child: updated })
  }

  return (
    <BodyNodeEditor
      node={child}
      onChange={updateChild}
      contentType={contentType}
      depth={depth + 1}
      canDelete={false}
    />
  )
}
