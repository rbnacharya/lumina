const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

export interface User {
  id: string
  email: string
  created_at: string
}

export interface VirtualKey {
  id: string
  user_id: string
  name: string
  allowed_models: string[]
  budget_limit: number | null
  current_spend: number
  created_at: string
  revoked_at: string | null
}

export interface CreateKeyRequest {
  name: string
  allowed_models?: string[]
  budget_limit?: number
}

export interface UpdateKeyRequest {
  name?: string
  allowed_models?: string[]
  budget_limit?: number
}

// Account-level provider configuration
export interface ProviderInfo {
  provider: 'openai' | 'anthropic'
  created_at: string
  updated_at: string
}

export interface SetProviderRequest {
  provider: 'openai' | 'anthropic'
  api_key: string
}

export interface CreateKeyResponse {
  id: string
  name: string
  allowed_models: string[]
  virtual_key: string
  created_at: string
}

export interface Overview {
  total_spend: number
  total_requests: number
  avg_latency: number
  success_rate: number
}

export interface DailyStat {
  id: string
  key_id: string
  date: string
  total_tokens: number
  total_cost: number
}

export interface LogEntry {
  trace_id: string
  timestamp: string
  virtual_key_name: string
  request: {
    model: string
    provider: string
    messages: unknown
  }
  response: {
    content: string
    usage: {
      prompt_tokens: number
      completion_tokens: number
      total_tokens: number
    }
    status_code: number
  }
  metrics: {
    latency_ms: number
    cost_usd: number
  }
}

class ApiClient {
  private async fetch<T>(path: string, options: RequestInit = {}): Promise<T> {
    const url = `${API_URL}${path}`
    const response = await fetch(url, {
      ...options,
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Unknown error' }))
      throw new Error(error.error || 'Request failed')
    }

    return response.json()
  }

  // Auth
  async login(email: string, password: string): Promise<{ user: User; token: string }> {
    return this.fetch('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    })
  }

  async register(email: string, password: string): Promise<{ user: User; token: string }> {
    return this.fetch('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    })
  }

  async logout(): Promise<void> {
    await this.fetch('/api/auth/logout', { method: 'POST' })
  }

  async me(): Promise<User> {
    return this.fetch('/api/auth/me')
  }

  // Keys
  async listKeys(): Promise<VirtualKey[]> {
    return this.fetch('/api/keys')
  }

  async createKey(data: CreateKeyRequest): Promise<CreateKeyResponse> {
    return this.fetch('/api/keys', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async revokeKey(id: string): Promise<void> {
    await this.fetch(`/api/keys/${id}`, { method: 'DELETE' })
  }

  async updateKey(id: string, data: UpdateKeyRequest): Promise<void> {
    await this.fetch(`/api/keys/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  // Provider management (account-level API keys)
  async listProviders(): Promise<ProviderInfo[]> {
    return this.fetch('/api/providers')
  }

  async setProvider(data: SetProviderRequest): Promise<void> {
    await this.fetch('/api/providers', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async removeProvider(provider: 'openai' | 'anthropic'): Promise<void> {
    await this.fetch(`/api/providers/${provider}`, { method: 'DELETE' })
  }

  // Playground - test a virtual key
  // Now uses provider/model format
  async testKey(virtualKey: string, model: string, message: string): Promise<{
    success: boolean
    response?: string
    error?: string
    latency_ms?: number
  }> {
    const startTime = Date.now()
    try {
      // Model should be in format "provider/model"
      const response = await fetch(`${API_URL}/v1/chat/completions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${virtualKey}`,
        },
        body: JSON.stringify({
          model: model, // e.g., "openai/gpt-3.5-turbo" or "anthropic/claude-3-haiku-20240307"
          messages: [{ role: 'user', content: message }],
          max_tokens: 100,
        }),
      })

      const latency_ms = Date.now() - startTime
      const data = await response.json()

      if (!response.ok) {
        return {
          success: false,
          error: data.error || 'Request failed',
          latency_ms,
        }
      }

      // Extract response content (works for both OpenAI and Anthropic format)
      let content = ''
      if (data.choices?.[0]?.message?.content) {
        content = data.choices[0].message.content
      } else if (data.content?.[0]?.text) {
        content = data.content[0].text
      }

      return {
        success: true,
        response: content,
        latency_ms,
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Network error',
        latency_ms: Date.now() - startTime,
      }
    }
  }

  // Stats
  async getOverview(): Promise<Overview> {
    return this.fetch('/api/stats/overview')
  }

  async getDailyStats(start?: string, end?: string): Promise<DailyStat[]> {
    const params = new URLSearchParams()
    if (start) params.append('start', start)
    if (end) params.append('end', end)
    const query = params.toString() ? `?${params.toString()}` : ''
    return this.fetch(`/api/stats/daily${query}`)
  }

  // Logs
  async searchLogs(params: {
    q?: string
    model?: string
    status?: number
    start?: string
    end?: string
    page?: number
    size?: number
  }): Promise<{ entries: LogEntry[]; total: number; page: number; size: number }> {
    const searchParams = new URLSearchParams()
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) searchParams.append(key, String(value))
    })
    const query = searchParams.toString() ? `?${searchParams.toString()}` : ''
    return this.fetch(`/api/logs${query}`)
  }

  async getLog(id: string): Promise<LogEntry> {
    return this.fetch(`/api/logs/${id}`)
  }
}

export const api = new ApiClient()
