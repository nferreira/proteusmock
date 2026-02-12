import yaml from 'js-yaml'
import type { ScenarioFormData, KeyValuePair, BodyMode, BodyNodeFormData } from './types'

/** Wrapper class so js-yaml can serialize !include tags. */
class IncludeRef {
  path: string
  constructor(path: string) {
    this.path = path
  }
}

const IncludeType = new yaml.Type('!include', {
  kind: 'scalar',
  instanceOf: IncludeRef,
  construct: (data: string) => new IncludeRef(data),
  represent: (ref: object) => (ref as IncludeRef).path,
})

const CUSTOM_SCHEMA = yaml.DEFAULT_SCHEMA.extend([IncludeType])

function kvToRecord(pairs: KeyValuePair[]): Record<string, string> | undefined {
  const filtered = pairs.filter(p => p.key.trim() !== '')
  if (filtered.length === 0) return undefined
  const result: Record<string, string> = {}
  for (const { key, value } of filtered) {
    result[key] = value
  }
  return result
}

/**
 * Determines which body mode is active based on the form data.
 * Priority: include_file > body_file > inline body.
 */
export function detectBodyMode(data: ScenarioFormData): BodyMode {
  if (data.response.include_file) return 'include'
  if (data.response.body_file) return 'file'
  return 'inline'
}

export function formDataToYaml(data: ScenarioFormData, activeBodyMode?: BodyMode): string {
  const obj: Record<string, unknown> = {
    id: data.id,
    name: data.name,
    priority: data.priority,
    when: buildWhen(data),
    response: buildResponse(data, activeBodyMode),
  }

  const policy = buildPolicy(data)
  if (policy && Object.keys(policy).length > 0) {
    obj.policy = policy
  }

  return yaml.dump(obj, { lineWidth: 120, noRefs: true, sortKeys: false, schema: CUSTOM_SCHEMA })
}

function serializeBodyNode(node: BodyNodeFormData): Record<string, unknown> | null {
  switch (node.type) {
    case 'conditions': {
      const conditions = node.conditions.filter(
        c => c.extractor.trim() !== '' || c.matcher.trim() !== '',
      )
      if (conditions.length === 0) return null
      return {
        conditions: conditions.map(c => ({
          extractor: c.extractor,
          matcher: c.matcher,
        })),
      }
    }
    case 'all':
    case 'any': {
      const children = node.children
        .map(serializeBodyNode)
        .filter((c): c is Record<string, unknown> => c !== null)
      if (children.length === 0) return null
      return { [node.type]: children }
    }
    case 'not': {
      if (!node.child) return null
      const child = serializeBodyNode(node.child)
      if (!child) return null
      return { not: child }
    }
  }
}

function buildWhen(data: ScenarioFormData): Record<string, unknown> {
  const when: Record<string, unknown> = {
    method: data.when.method,
    path: data.when.path,
  }

  const headers = kvToRecord(data.when.headers)
  if (headers) {
    when.headers = headers
  }

  if (data.when.body) {
    const body: Record<string, unknown> = {}
    if (data.when.body.content_type) {
      body.content_type = data.when.body.content_type
    }
    const nodeData = serializeBodyNode(data.when.body.node)
    if (nodeData) {
      Object.assign(body, nodeData)
    }
    if (Object.keys(body).length > 0) {
      when.body = body
    }
  }

  return when
}

function buildResponse(data: ScenarioFormData, activeBodyMode?: BodyMode): Record<string, unknown> {
  const resp: Record<string, unknown> = {
    status: data.response.status,
  }

  const headers = kvToRecord(data.response.headers)
  if (headers) {
    resp.headers = headers
  }

  if (data.response.content_type) {
    resp.content_type = data.response.content_type
  }

  if (data.response.engine) {
    resp.engine = data.response.engine
  }

  // Serialize the active body mode only.
  const mode = activeBodyMode ?? detectBodyMode(data)
  switch (mode) {
    case 'include':
      if (data.response.include_file) {
        resp.body = new IncludeRef(data.response.include_file)
      }
      break
    case 'file':
      if (data.response.body_file) {
        resp.body_file = data.response.body_file
      }
      break
    case 'inline':
      if (data.response.body) {
        resp.body = data.response.body
      }
      break
  }

  return resp
}

function buildPolicy(data: ScenarioFormData): Record<string, unknown> | undefined {
  if (!data.policy) return undefined

  const policy: Record<string, unknown> = {}

  if (data.policy.rate_limit) {
    const rl: Record<string, unknown> = {
      rate: data.policy.rate_limit.rate,
      burst: data.policy.rate_limit.burst,
    }
    if (data.policy.rate_limit.key) {
      rl.key = data.policy.rate_limit.key
    }
    policy.rate_limit = rl
  }

  if (data.policy.latency) {
    const lat: Record<string, unknown> = {}
    if (data.policy.latency.fixed_ms > 0) lat.fixed_ms = data.policy.latency.fixed_ms
    if (data.policy.latency.jitter_ms > 0) lat.jitter_ms = data.policy.latency.jitter_ms
    if (Object.keys(lat).length > 0) {
      policy.latency = lat
    }
  }

  if (data.policy.pagination) {
    const pg: Record<string, unknown> = {
      style: data.policy.pagination.style,
    }
    if (data.policy.pagination.default_size > 0) pg.default_size = data.policy.pagination.default_size
    if (data.policy.pagination.max_size > 0) pg.max_size = data.policy.pagination.max_size
    if (data.policy.pagination.data_path) pg.data_path = data.policy.pagination.data_path
    policy.pagination = pg
  }

  return Object.keys(policy).length > 0 ? policy : undefined
}
