'use client'

import { useState } from 'react'
import { DashboardNav } from '@/components/dashboard/nav'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Copy, Check } from 'lucide-react'
import { Button } from '@/components/ui/button'

export default function DevelopersPage() {
  const [copiedCode, setCopiedCode] = useState<string | null>(null)

  const copyCode = (code: string, id: string) => {
    navigator.clipboard.writeText(code)
    setCopiedCode(id)
    setTimeout(() => setCopiedCode(null), 2000)
  }

  const CodeBlock = ({ code, id, language = 'bash' }: { code: string; id: string; language?: string }) => (
    <div className="relative">
      <pre className="rounded-lg bg-zinc-950 p-4 overflow-x-auto text-sm text-zinc-100">
        <code>{code}</code>
      </pre>
      <Button
        variant="ghost"
        size="icon"
        className="absolute top-2 right-2 h-8 w-8 text-zinc-400 hover:text-zinc-100"
        onClick={() => copyCode(code, id)}
      >
        {copiedCode === id ? (
          <Check className="h-4 w-4 text-green-500" />
        ) : (
          <Copy className="h-4 w-4" />
        )}
      </Button>
    </div>
  )

  return (
    <div className="flex h-screen">
      <DashboardNav />
      <main className="flex-1 overflow-auto p-8">
        <div className="max-w-4xl">
          <div className="mb-8">
            <h1 className="text-3xl font-bold">API Documentation</h1>
            <p className="text-muted-foreground mt-2">
              Learn how to integrate Lumina Gateway into your applications
            </p>
          </div>

          {/* Getting Started */}
          <Card className="mb-6">
            <CardHeader>
              <CardTitle>Getting Started</CardTitle>
              <CardDescription>
                Lumina Gateway provides a unified API for multiple LLM providers
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <h3 className="font-semibold">1. Create a Virtual Key</h3>
                <p className="text-sm text-muted-foreground">
                  Go to the <a href="/keys" className="text-primary underline">Keys</a> page and create a new virtual key.
                  Add credentials for the providers you want to use (OpenAI, Anthropic, or both).
                </p>
              </div>
              <div className="space-y-2">
                <h3 className="font-semibold">2. Use the provider/model Format</h3>
                <p className="text-sm text-muted-foreground">
                  When making API requests, specify the model using the format{' '}
                  <code className="bg-muted px-1 rounded">provider/model</code>.
                  The gateway will automatically route your request to the correct provider.
                </p>
              </div>
              <div className="space-y-2">
                <h3 className="font-semibold">3. Make API Requests</h3>
                <p className="text-sm text-muted-foreground">
                  Use your virtual key in the Authorization header. The gateway endpoint is{' '}
                  <code className="bg-muted px-1 rounded">http://localhost:8080</code>.
                </p>
              </div>
            </CardContent>
          </Card>

          {/* API Reference */}
          <Card className="mb-6">
            <CardHeader>
              <CardTitle>API Reference</CardTitle>
              <CardDescription>
                OpenAI-compatible endpoints with multi-provider support
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Tabs defaultValue="chat">
                <TabsList className="mb-4">
                  <TabsTrigger value="chat">Chat Completions</TabsTrigger>
                  <TabsTrigger value="completions">Completions</TabsTrigger>
                  <TabsTrigger value="embeddings">Embeddings</TabsTrigger>
                </TabsList>

                <TabsContent value="chat" className="space-y-4">
                  <div className="flex items-center gap-2">
                    <Badge>POST</Badge>
                    <code className="text-sm">/v1/chat/completions</code>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Create a chat completion with any supported model.
                  </p>
                  <CodeBlock
                    id="chat-curl"
                    code={`curl -X POST http://localhost:8080/v1/chat/completions \\
  -H "Authorization: Bearer lum_your_virtual_key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "openai/gpt-4o",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "Hello!"}
    ],
    "max_tokens": 100
  }'`}
                  />
                </TabsContent>

                <TabsContent value="completions" className="space-y-4">
                  <div className="flex items-center gap-2">
                    <Badge>POST</Badge>
                    <code className="text-sm">/v1/completions</code>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Create a text completion (legacy endpoint).
                  </p>
                  <CodeBlock
                    id="completions-curl"
                    code={`curl -X POST http://localhost:8080/v1/completions \\
  -H "Authorization: Bearer lum_your_virtual_key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "openai/gpt-3.5-turbo-instruct",
    "prompt": "Say hello",
    "max_tokens": 50
  }'`}
                  />
                </TabsContent>

                <TabsContent value="embeddings" className="space-y-4">
                  <div className="flex items-center gap-2">
                    <Badge>POST</Badge>
                    <code className="text-sm">/v1/embeddings</code>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Create embeddings for text.
                  </p>
                  <CodeBlock
                    id="embeddings-curl"
                    code={`curl -X POST http://localhost:8080/v1/embeddings \\
  -H "Authorization: Bearer lum_your_virtual_key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "openai/text-embedding-3-small",
    "input": "Hello world"
  }'`}
                  />
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>

          {/* Code Examples */}
          <Card className="mb-6">
            <CardHeader>
              <CardTitle>Code Examples</CardTitle>
              <CardDescription>
                Integration examples in different languages
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Tabs defaultValue="python">
                <TabsList className="mb-4">
                  <TabsTrigger value="python">Python</TabsTrigger>
                  <TabsTrigger value="javascript">JavaScript</TabsTrigger>
                  <TabsTrigger value="go">Go</TabsTrigger>
                </TabsList>

                <TabsContent value="python" className="space-y-4">
                  <p className="text-sm text-muted-foreground">
                    Using the OpenAI Python SDK with Lumina Gateway:
                  </p>
                  <CodeBlock
                    id="python-openai"
                    language="python"
                    code={`from openai import OpenAI

client = OpenAI(
    api_key="lum_your_virtual_key",
    base_url="http://localhost:8080/v1"
)

# Use OpenAI models
response = client.chat.completions.create(
    model="openai/gpt-4o",
    messages=[
        {"role": "user", "content": "Hello!"}
    ]
)
print(response.choices[0].message.content)

# Use Anthropic models with the same client
response = client.chat.completions.create(
    model="anthropic/claude-3-5-sonnet-20241022",
    messages=[
        {"role": "user", "content": "Hello!"}
    ]
)
print(response.choices[0].message.content)`}
                  />
                </TabsContent>

                <TabsContent value="javascript" className="space-y-4">
                  <p className="text-sm text-muted-foreground">
                    Using fetch or the OpenAI JavaScript SDK:
                  </p>
                  <CodeBlock
                    id="js-fetch"
                    language="javascript"
                    code={`// Using fetch
const response = await fetch('http://localhost:8080/v1/chat/completions', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer lum_your_virtual_key',
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    model: 'openai/gpt-4o',
    messages: [{ role: 'user', content: 'Hello!' }],
  }),
});

const data = await response.json();
console.log(data.choices[0].message.content);

// Using OpenAI SDK
import OpenAI from 'openai';

const openai = new OpenAI({
  apiKey: 'lum_your_virtual_key',
  baseURL: 'http://localhost:8080/v1',
});

const completion = await openai.chat.completions.create({
  model: 'anthropic/claude-3-haiku-20240307',
  messages: [{ role: 'user', content: 'Hello!' }],
});

console.log(completion.choices[0].message.content);`}
                  />
                </TabsContent>

                <TabsContent value="go" className="space-y-4">
                  <p className="text-sm text-muted-foreground">
                    Using Go with net/http:
                  </p>
                  <CodeBlock
                    id="go-http"
                    language="go"
                    code={`package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

func main() {
    url := "http://localhost:8080/v1/chat/completions"

    payload := map[string]interface{}{
        "model": "openai/gpt-4o",
        "messages": []map[string]string{
            {"role": "user", "content": "Hello!"},
        },
    }

    body, _ := json.Marshal(payload)
    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
    req.Header.Set("Authorization", "Bearer lum_your_virtual_key")
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    fmt.Println(result)
}`}
                  />
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>

          {/* Supported Models */}
          <Card className="mb-6">
            <CardHeader>
              <CardTitle>Supported Models</CardTitle>
              <CardDescription>
                Use the provider/model format to specify which model to use
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid md:grid-cols-2 gap-6">
                <div>
                  <h3 className="font-semibold mb-3 flex items-center gap-2">
                    <Badge variant="outline">OpenAI</Badge>
                  </h3>
                  <ul className="space-y-2 text-sm">
                    <li><code className="bg-muted px-2 py-0.5 rounded">openai/gpt-4o</code></li>
                    <li><code className="bg-muted px-2 py-0.5 rounded">openai/gpt-4o-mini</code></li>
                    <li><code className="bg-muted px-2 py-0.5 rounded">openai/gpt-4-turbo</code></li>
                    <li><code className="bg-muted px-2 py-0.5 rounded">openai/gpt-3.5-turbo</code></li>
                    <li><code className="bg-muted px-2 py-0.5 rounded">openai/o1-preview</code></li>
                    <li><code className="bg-muted px-2 py-0.5 rounded">openai/o1-mini</code></li>
                    <li><code className="bg-muted px-2 py-0.5 rounded">openai/text-embedding-3-small</code></li>
                    <li><code className="bg-muted px-2 py-0.5 rounded">openai/text-embedding-3-large</code></li>
                  </ul>
                </div>
                <div>
                  <h3 className="font-semibold mb-3 flex items-center gap-2">
                    <Badge variant="outline">Anthropic</Badge>
                  </h3>
                  <ul className="space-y-2 text-sm">
                    <li><code className="bg-muted px-2 py-0.5 rounded">anthropic/claude-3-5-sonnet-20241022</code></li>
                    <li><code className="bg-muted px-2 py-0.5 rounded">anthropic/claude-3-opus-20240229</code></li>
                    <li><code className="bg-muted px-2 py-0.5 rounded">anthropic/claude-3-sonnet-20240229</code></li>
                    <li><code className="bg-muted px-2 py-0.5 rounded">anthropic/claude-3-haiku-20240307</code></li>
                  </ul>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Model Access Control */}
          <Card className="mb-6">
            <CardHeader>
              <CardTitle>Model Access Control</CardTitle>
              <CardDescription>
                Restrict which models a virtual key can access
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-muted-foreground">
                When creating a virtual key, you can specify allowed model patterns to restrict access:
              </p>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b">
                      <th className="text-left py-2 font-medium">Pattern</th>
                      <th className="text-left py-2 font-medium">Description</th>
                      <th className="text-left py-2 font-medium">Example Matches</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr className="border-b">
                      <td className="py-2"><code className="bg-muted px-1 rounded">*</code></td>
                      <td className="py-2 text-muted-foreground">All models</td>
                      <td className="py-2 text-muted-foreground">Any model</td>
                    </tr>
                    <tr className="border-b">
                      <td className="py-2"><code className="bg-muted px-1 rounded">openai/*</code></td>
                      <td className="py-2 text-muted-foreground">All OpenAI models</td>
                      <td className="py-2 text-muted-foreground">openai/gpt-4o, openai/gpt-3.5-turbo</td>
                    </tr>
                    <tr className="border-b">
                      <td className="py-2"><code className="bg-muted px-1 rounded">anthropic/claude-3-*</code></td>
                      <td className="py-2 text-muted-foreground">All Claude 3 models</td>
                      <td className="py-2 text-muted-foreground">anthropic/claude-3-opus-20240229</td>
                    </tr>
                    <tr>
                      <td className="py-2"><code className="bg-muted px-1 rounded">openai/gpt-4o</code></td>
                      <td className="py-2 text-muted-foreground">Exact match</td>
                      <td className="py-2 text-muted-foreground">Only openai/gpt-4o</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>

          {/* Error Handling */}
          <Card className="mb-6">
            <CardHeader>
              <CardTitle>Error Handling</CardTitle>
              <CardDescription>
                Common error responses and how to handle them
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="rounded-lg border p-4">
                  <div className="flex items-center gap-2 mb-2">
                    <Badge variant="destructive">401</Badge>
                    <span className="font-medium">Unauthorized</span>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Invalid or missing virtual key. Check that your key is correct and hasn&apos;t been revoked.
                  </p>
                </div>
                <div className="rounded-lg border p-4">
                  <div className="flex items-center gap-2 mb-2">
                    <Badge variant="destructive">400</Badge>
                    <span className="font-medium">Bad Request</span>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Invalid model format or provider not configured. Ensure model is in{' '}
                    <code className="bg-muted px-1 rounded">provider/model</code> format.
                  </p>
                </div>
                <div className="rounded-lg border p-4">
                  <div className="flex items-center gap-2 mb-2">
                    <Badge variant="destructive">403</Badge>
                    <span className="font-medium">Forbidden</span>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Model not allowed for this key or budget exceeded. Check your key&apos;s allowed models and spending limit.
                  </p>
                </div>
                <div className="rounded-lg border p-4">
                  <div className="flex items-center gap-2 mb-2">
                    <Badge variant="destructive">502</Badge>
                    <span className="font-medium">Bad Gateway</span>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Failed to reach upstream provider. The provider may be experiencing issues.
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Rate Limiting & Budgets */}
          <Card>
            <CardHeader>
              <CardTitle>Budgets & Cost Tracking</CardTitle>
              <CardDescription>
                Monitor and control your API spending
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-muted-foreground">
                Each virtual key can have an optional monthly budget limit. When the limit is reached,
                requests will be rejected with a 403 error.
              </p>
              <ul className="list-disc list-inside text-sm text-muted-foreground space-y-1">
                <li>Set budget limits when creating or editing keys</li>
                <li>View current spend on the Keys page</li>
                <li>Monitor usage trends on the Dashboard</li>
                <li>Search and filter requests in the Logs page</li>
              </ul>
            </CardContent>
          </Card>
        </div>
      </main>
    </div>
  )
}
