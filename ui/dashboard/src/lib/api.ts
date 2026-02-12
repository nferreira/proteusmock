export interface ScenarioSummary {
  id: string
  name: string
  priority: number
  method: string
  path_key: string
}

export interface ScenarioDetail {
  id: string
  name: string
  priority: number
  source_file: string
  source_index: number
  source_yaml: string
  when: {
    method: string
    path: string
    headers?: Record<string, string>
    body?: Record<string, unknown>
  }
  response: {
    status: number
    headers?: Record<string, string>
    body?: string
    body_file?: string
    content_type?: string
    engine?: string
  }
  policy?: {
    rate_limit?: { rate: number; burst: number; key: string }
    latency?: { fixed_ms: number; jitter_ms: number }
    pagination?: { style: string; default_size: number; max_size: number; data_path: string }
  }
}

export interface TraceEntry {
  timestamp: string
  method: string
  path: string
  matched_id: string
  rate_limited: boolean
  candidates?: {
    scenario_id: string
    scenario_name: string
    matched: boolean
    failed_field?: string
    failed_reason?: string
  }[]
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`${res.status}: ${text}`)
  }
  return res.json() as Promise<T>
}

export const api = {
  listScenarios: (): Promise<ScenarioSummary[]> =>
    fetch('/__admin/scenarios').then(r => handleResponse<ScenarioSummary[]>(r)),

  searchScenarios: (q: string): Promise<ScenarioSummary[]> =>
    fetch(`/__admin/scenarios/search?q=${encodeURIComponent(q)}`).then(r =>
      handleResponse<ScenarioSummary[]>(r),
    ),

  getScenario: (id: string): Promise<ScenarioDetail> =>
    fetch(`/__admin/scenarios/${encodeURIComponent(id)}`).then(r =>
      handleResponse<ScenarioDetail>(r),
    ),

  updateScenario: (id: string, yaml: string): Promise<{ status: string; message: string }> =>
    fetch(`/__admin/scenarios/${encodeURIComponent(id)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'text/yaml' },
      body: yaml,
    }).then(r => handleResponse(r)),

  createScenario: (yaml: string): Promise<{ status: string; message: string }> =>
    fetch('/__admin/scenarios', {
      method: 'POST',
      headers: { 'Content-Type': 'text/yaml' },
      body: yaml,
    }).then(r => handleResponse(r)),

  deleteScenario: (id: string): Promise<void> =>
    fetch(`/__admin/scenarios/${encodeURIComponent(id)}`, { method: 'DELETE' }).then(r => {
      if (!r.ok) throw new Error(`${r.status}`)
    }),

  listFiles: (): Promise<string[]> =>
    fetch('/__admin/files').then(r => handleResponse<string[]>(r)),

  getTrace: (last = 50): Promise<TraceEntry[]> =>
    fetch(`/__admin/trace?last=${last}`).then(r => handleResponse<TraceEntry[]>(r)),

  reload: (): Promise<{ status: string; message: string }> =>
    fetch('/__admin/reload', { method: 'POST' }).then(r => handleResponse(r)),
}
