'use client'

import { useEffect, useState } from 'react'
import { DashboardNav } from '@/components/dashboard/nav'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { api, type VirtualKey, type ProviderInfo } from '@/lib/api'
import { Play, CheckCircle, XCircle, Clock } from 'lucide-react'

// Available models in provider/model format
const AVAILABLE_MODELS = [
  { value: 'openai/gpt-4o', label: 'OpenAI GPT-4o', provider: 'openai' },
  { value: 'openai/gpt-4o-mini', label: 'OpenAI GPT-4o Mini', provider: 'openai' },
  { value: 'openai/gpt-3.5-turbo', label: 'OpenAI GPT-3.5 Turbo', provider: 'openai' },
  { value: 'anthropic/claude-3-5-sonnet-20241022', label: 'Claude 3.5 Sonnet', provider: 'anthropic' },
  { value: 'anthropic/claude-3-sonnet-20240229', label: 'Claude 3 Sonnet', provider: 'anthropic' },
  { value: 'anthropic/claude-3-haiku-20240307', label: 'Claude 3 Haiku', provider: 'anthropic' },
]

export default function PlaygroundPage() {
  const [keys, setKeys] = useState<VirtualKey[]>([])
  const [providers, setProviders] = useState<ProviderInfo[]>([])
  const [selectedKeyId, setSelectedKeyId] = useState<string>('')
  const [virtualKey, setVirtualKey] = useState('')
  const [selectedModel, setSelectedModel] = useState('openai/gpt-3.5-turbo')
  const [message, setMessage] = useState('Hello! Can you respond with a short greeting?')
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<{
    success: boolean
    response?: string
    error?: string
    latency_ms?: number
  } | null>(null)

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [keysData, providersData] = await Promise.all([
          api.listKeys(),
          api.listProviders()
        ])
        setKeys((keysData || []).filter(k => !k.revoked_at))
        setProviders(providersData || [])
      } catch (error) {
        console.error('Failed to fetch data:', error)
      }
    }
    fetchData()
  }, [])

  const selectedKey = keys.find(k => k.id === selectedKeyId)

  // Filter available models based on account-level providers
  const availableModelsForKey = providers.length > 0
    ? AVAILABLE_MODELS.filter(m =>
        providers.some(p => p.provider === m.provider)
      )
    : AVAILABLE_MODELS

  // Filter by allowed models if specified
  const filteredModels = selectedKey && selectedKey.allowed_models?.length > 0
    ? availableModelsForKey.filter(m => {
        return selectedKey.allowed_models.some(pattern => {
          if (pattern === '*') return true
          if (pattern.endsWith('*')) {
            return m.value.startsWith(pattern.slice(0, -1))
          }
          return m.value === pattern || m.value.startsWith(pattern + '/')
        })
      })
    : availableModelsForKey

  const handleTest = async () => {
    if (!virtualKey || !selectedModel) return

    setLoading(true)
    setResult(null)

    try {
      const testResult = await api.testKey(virtualKey, selectedModel, message)
      setResult(testResult)
    } catch (error) {
      setResult({
        success: false,
        error: error instanceof Error ? error.message : 'Test failed',
      })
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex h-screen">
      <DashboardNav />
      <main className="flex-1 overflow-auto p-8">
        <div className="mb-8">
          <h1 className="text-3xl font-bold">Playground</h1>
          <p className="text-muted-foreground">
            Test your virtual keys with different models
          </p>
        </div>

        <div className="grid gap-6 lg:grid-cols-2">
          {/* Configuration */}
          <Card>
            <CardHeader>
              <CardTitle>Configuration</CardTitle>
              <CardDescription>
                Select a key and model to test
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Select Key</Label>
                <Select value={selectedKeyId} onValueChange={setSelectedKeyId}>
                  <SelectTrigger>
                    <SelectValue placeholder="Choose a key to test" />
                  </SelectTrigger>
                  <SelectContent>
                    {keys.map((key) => (
                      <SelectItem key={key.id} value={key.id}>
                        {key.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {selectedKey && (
                <div className="rounded-lg bg-muted p-3 text-sm space-y-2">
                  <div className="flex items-center gap-2">
                    <span className="text-muted-foreground">Configured Providers:</span>
                    {providers.map(p => (
                      <Badge key={p.provider} variant="outline">
                        {p.provider.toUpperCase()}
                      </Badge>
                    ))}
                    {providers.length === 0 && (
                      <span className="text-muted-foreground italic">None configured</span>
                    )}
                  </div>
                  {selectedKey.allowed_models?.length > 0 && (
                    <div>
                      <span className="text-muted-foreground">Allowed models: </span>
                      <span>{selectedKey.allowed_models.join(', ')}</span>
                    </div>
                  )}
                </div>
              )}

              <div className="space-y-2">
                <Label>Select Model</Label>
                <Select value={selectedModel} onValueChange={setSelectedModel}>
                  <SelectTrigger>
                    <SelectValue placeholder="Choose a model" />
                  </SelectTrigger>
                  <SelectContent>
                    {filteredModels.map((model) => (
                      <SelectItem key={model.value} value={model.value}>
                        {model.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  Format: <code className="bg-muted px-1 rounded">provider/model</code>
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="virtualKey">Virtual Key</Label>
                <Input
                  id="virtualKey"
                  type="password"
                  placeholder="lum_..."
                  value={virtualKey}
                  onChange={(e) => setVirtualKey(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">
                  Enter the virtual key you received when creating the key
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="message">Test Message</Label>
                <Input
                  id="message"
                  placeholder="Enter a test message..."
                  value={message}
                  onChange={(e) => setMessage(e.target.value)}
                />
              </div>

              <Button
                onClick={handleTest}
                disabled={!selectedKeyId || !virtualKey || !message || loading}
                className="w-full"
              >
                {loading ? (
                  <>Testing...</>
                ) : (
                  <>
                    <Play className="mr-2 h-4 w-4" />
                    Test Key
                  </>
                )}
              </Button>
            </CardContent>
          </Card>

          {/* Results */}
          <Card>
            <CardHeader>
              <CardTitle>Results</CardTitle>
              <CardDescription>
                Response from the LLM provider
              </CardDescription>
            </CardHeader>
            <CardContent>
              {!result ? (
                <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                  <Play className="h-12 w-12 mb-4 opacity-20" />
                  <p>Run a test to see results</p>
                </div>
              ) : (
                <div className="space-y-4">
                  <div className="flex items-center gap-4">
                    {result.success ? (
                      <Badge variant="success" className="gap-1">
                        <CheckCircle className="h-3 w-3" />
                        Success
                      </Badge>
                    ) : (
                      <Badge variant="destructive" className="gap-1">
                        <XCircle className="h-3 w-3" />
                        Failed
                      </Badge>
                    )}
                    {result.latency_ms && (
                      <Badge variant="outline" className="gap-1">
                        <Clock className="h-3 w-3" />
                        {result.latency_ms}ms
                      </Badge>
                    )}
                  </div>

                  {result.error && (
                    <div className="rounded-lg bg-destructive/10 border border-destructive/20 p-4">
                      <p className="text-sm font-medium text-destructive">Error</p>
                      <p className="text-sm text-destructive/80 mt-1">{result.error}</p>
                    </div>
                  )}

                  {result.response && (
                    <div className="space-y-2">
                      <p className="text-sm font-medium">Response</p>
                      <div className="rounded-lg bg-muted p-4">
                        <p className="text-sm whitespace-pre-wrap">{result.response}</p>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Usage Example */}
        <Card className="mt-6">
          <CardHeader>
            <CardTitle>Usage Example</CardTitle>
            <CardDescription>
              How to use your virtual key in code with the provider/model format
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="rounded-lg bg-muted p-4 font-mono text-sm overflow-x-auto">
              <pre>{`curl -X POST http://localhost:8080/v1/chat/completions \\
  -H "Authorization: Bearer ${virtualKey || 'lum_your_virtual_key'}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${selectedModel}",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'`}</pre>
            </div>
            <p className="text-sm text-muted-foreground mt-4">
              The model format is <code className="bg-muted px-1 rounded">provider/model</code>.
              The gateway will route your request to the correct provider automatically.
            </p>
          </CardContent>
        </Card>
      </main>
    </div>
  )
}
