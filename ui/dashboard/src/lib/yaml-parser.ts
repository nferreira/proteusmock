import yaml from 'js-yaml'
import type {
  ScenarioFormData,
  KeyValuePair,
  BodyFormData,
  BodyNodeFormData,
  ResponseFormData,
  PolicyFormData,
} from './types'

// Custom YAML schema that handles !include tags by preserving them as prefixed strings.
const IncludeType = new yaml.Type('!include', {
  kind: 'scalar',
  construct: (data: string) => `!include ${data}`,
})

const CUSTOM_SCHEMA = yaml.DEFAULT_SCHEMA.extend([IncludeType])

/** Prefix used to identify !include-resolved values after YAML parsing. */
const INCLUDE_PREFIX = '!include '

export interface ParseResult {
  data: ScenarioFormData
  warnings: string[]
}

function recordToKv(rec?: Record<string, string>): KeyValuePair[] {
  if (!rec) return []
  return Object.entries(rec).map(([key, value]) => ({ key, value: String(value) }))
}

let nextId = 1
export function uid(): string {
  return String(nextId++)
}

const MAX_PARSE_DEPTH = 20

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function parseBodyNode(raw: any, depth = 0): BodyNodeFormData {
  if (depth > MAX_PARSE_DEPTH || !raw || typeof raw !== 'object') {
    return defaultBodyNode()
  }

  if (Array.isArray(raw.all)) {
    return {
      id: uid(),
      type: 'all',
      conditions: [],
      children: raw.all.map((c: unknown) => parseBodyNode(c, depth + 1)),
    }
  }

  if (Array.isArray(raw.any)) {
    return {
      id: uid(),
      type: 'any',
      conditions: [],
      children: raw.any.map((c: unknown) => parseBodyNode(c, depth + 1)),
    }
  }

  if (raw.not && typeof raw.not === 'object') {
    return {
      id: uid(),
      type: 'not',
      conditions: [],
      children: [],
      child: parseBodyNode(raw.not, depth + 1),
    }
  }

  // Default: conditions leaf
  const conditions = (raw.conditions ?? []).map((c: { extractor: string; matcher: string }) => ({
    id: uid(),
    extractor: c.extractor ?? '',
    matcher: c.matcher ?? '',
  }))
  return {
    id: uid(),
    type: 'conditions',
    conditions: conditions.length > 0 ? conditions : [{ id: uid(), extractor: '', matcher: '' }],
    children: [],
  }
}

export function defaultBodyNode(): BodyNodeFormData {
  return {
    id: uid(),
    type: 'conditions',
    conditions: [{ id: uid(), extractor: '', matcher: '' }],
    children: [],
  }
}

export function yamlToFormData(yamlStr: string): ParseResult {
  const warnings: string[] = []

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const raw = yaml.load(yamlStr, { schema: CUSTOM_SCHEMA }) as Record<string, any> | null
  if (!raw || typeof raw !== 'object') {
    return {
      data: defaultFormData(),
      warnings: ['Could not parse YAML'],
    }
  }

  // Parse when clause
  const when = raw.when ?? {}
  let body: BodyFormData | undefined
  if (when.body) {
    body = {
      content_type: when.body.content_type ?? '',
      node: parseBodyNode(when.body),
    }
  }

  // Parse response — detect !include in body
  const resp = raw.response ?? {}
  const rawBody: string = resp.body ?? ''
  let includeFile = ''
  let inlineBody = rawBody

  if (typeof rawBody === 'string' && rawBody.startsWith(INCLUDE_PREFIX)) {
    includeFile = rawBody.slice(INCLUDE_PREFIX.length)
    inlineBody = '' // the body was from an include, not inline
  }

  const response: ResponseFormData = {
    status: resp.status ?? 200,
    headers: recordToKv(resp.headers),
    body: inlineBody,
    body_file: resp.body_file ?? '',
    include_file: includeFile,
    content_type: resp.content_type ?? '',
    engine: (['expr', 'jinja2'].includes(resp.engine) ? resp.engine : '') as ResponseFormData['engine'],
  }

  // Parse policy
  let policy: PolicyFormData | undefined
  if (raw.policy) {
    policy = {}
    if (raw.policy.rate_limit) {
      policy.rate_limit = {
        rate: raw.policy.rate_limit.rate ?? 0,
        burst: raw.policy.rate_limit.burst ?? 0,
        key: raw.policy.rate_limit.key ?? '',
      }
    }
    if (raw.policy.latency) {
      policy.latency = {
        fixed_ms: raw.policy.latency.fixed_ms ?? 0,
        jitter_ms: raw.policy.latency.jitter_ms ?? 0,
      }
    }
    if (raw.policy.pagination) {
      policy.pagination = {
        style: raw.policy.pagination.style === 'offset_limit' ? 'offset_limit' : 'page_size',
        default_size: raw.policy.pagination.default_size ?? 10,
        max_size: raw.policy.pagination.max_size ?? 100,
        data_path: raw.policy.pagination.data_path ?? '$',
      }
    }
  }

  return {
    data: {
      id: raw.id ?? '',
      name: raw.name ?? '',
      priority: raw.priority ?? 10,
      when: {
        method: when.method ?? 'GET',
        path: when.path ?? '/',
        headers: recordToKv(when.headers),
        body,
      },
      response,
      policy,
    },
    warnings,
  }
}

export function defaultFormData(): ScenarioFormData {
  return {
    id: 'new-scenario',
    name: 'New Scenario',
    priority: 10,
    when: {
      method: 'GET',
      path: '/api/v1/example',
      headers: [],
    },
    response: {
      status: 200,
      headers: [{ key: 'Content-Type', value: 'application/json' }],
      body: '{"message": "hello"}',
      body_file: '',
      include_file: '',
      content_type: 'application/json',
      engine: '',
    },
  }
}
