export interface KeyValuePair {
  key: string
  value: string
}

export interface BodyConditionFormData {
  id: string
  extractor: string
  matcher: string
}

export type BodyNodeType = 'conditions' | 'all' | 'any' | 'not'

export interface BodyNodeFormData {
  id: string
  type: BodyNodeType
  conditions: BodyConditionFormData[]  // when type='conditions'
  children: BodyNodeFormData[]         // when type='all' or 'any'
  child?: BodyNodeFormData             // when type='not'
}

export interface BodyFormData {
  content_type: string
  node: BodyNodeFormData
}

export interface WhenFormData {
  method: string
  path: string
  headers: KeyValuePair[]
  body?: BodyFormData
}

export type BodyMode = 'inline' | 'include' | 'file'

export interface ResponseFormData {
  status: number
  headers: KeyValuePair[]
  /** Inline body text */
  body: string
  /** Runtime file reference (body_file YAML field) */
  body_file: string
  /** !include file reference (resolved at YAML parse time) */
  include_file: string
  content_type: string
  engine: '' | 'expr' | 'jinja2'
}

export interface RateLimitFormData {
  rate: number
  burst: number
  key: string
}

export interface LatencyFormData {
  fixed_ms: number
  jitter_ms: number
}

export interface PaginationFormData {
  style: 'page_size' | 'offset_limit'
  default_size: number
  max_size: number
  data_path: string
}

export interface PolicyFormData {
  rate_limit?: RateLimitFormData
  latency?: LatencyFormData
  pagination?: PaginationFormData
}

export interface ScenarioFormData {
  id: string
  name: string
  priority: number
  when: WhenFormData
  response: ResponseFormData
  policy?: PolicyFormData
}
