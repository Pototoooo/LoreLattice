import { get, put } from '@/utils/request'

export interface SystemInfo {
  version: string
  commit_id?: string
  build_time?: string
  go_version?: string
}

export interface ToolDefinition {
  name: string
  label: string
  description: string
}

export interface AgentConfig {
  enabled: boolean
  max_iterations: number
  reflection_enabled: boolean
  allowed_tools: string[]
  temperature: number
  thinking_model_id: string
  rerank_model_id: string
  knowledge_bases?: string[]
  available_tools?: ToolDefinition[]  // GET 响应中包含，POST/PUT 不需要
}

export function getSystemInfo(): Promise<{ data: SystemInfo }> {
  return get('/api/v1/system/info')
}

export function getAgentConfig(): Promise<{ data: AgentConfig }> {
  return get('/api/v1/tenants/agent-config')
}

export function updateAgentConfig(config: AgentConfig): Promise<{ data: AgentConfig }> {
  return put('/api/v1/tenants/agent-config', config)
}
