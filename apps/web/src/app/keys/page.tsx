'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { DashboardNav } from '@/components/dashboard/nav'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { api, type VirtualKey, type CreateKeyResponse } from '@/lib/api'
import { formatCurrency, formatDate } from '@/lib/utils'
import { Plus, Trash2, Copy, Check, Pencil, X, Settings } from 'lucide-react'

export default function KeysPage() {
  const [keys, setKeys] = useState<VirtualKey[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [newKey, setNewKey] = useState<CreateKeyResponse | null>(null)
  const [copied, setCopied] = useState(false)
  const [editingKey, setEditingKey] = useState<VirtualKey | null>(null)

  // Create form state
  const [name, setName] = useState('')
  const [allowedModels, setAllowedModels] = useState('')
  const [budgetLimit, setBudgetLimit] = useState('')
  const [creating, setCreating] = useState(false)

  // Edit form state
  const [editName, setEditName] = useState('')
  const [editAllowedModels, setEditAllowedModels] = useState('')
  const [editBudgetLimit, setEditBudgetLimit] = useState('')
  const [updating, setUpdating] = useState(false)

  useEffect(() => {
    fetchKeys()
  }, [])

  const fetchKeys = async () => {
    try {
      const data = await api.listKeys()
      setKeys(data || [])
    } catch (error) {
      console.error('Failed to fetch keys:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreating(true)

    try {
      const response = await api.createKey({
        name,
        allowed_models: allowedModels ? allowedModels.split(',').map(m => m.trim()) : undefined,
        budget_limit: budgetLimit ? parseFloat(budgetLimit) : undefined,
      })
      setNewKey(response)
      fetchKeys()
      resetCreateForm()
      setShowCreateForm(false)
    } catch (error) {
      console.error('Failed to create key:', error)
    } finally {
      setCreating(false)
    }
  }

  const resetCreateForm = () => {
    setName('')
    setAllowedModels('')
    setBudgetLimit('')
  }

  const handleRevoke = async (id: string) => {
    if (!confirm('Are you sure you want to revoke this key?')) return

    try {
      await api.revokeKey(id)
      fetchKeys()
    } catch (error) {
      console.error('Failed to revoke key:', error)
    }
  }

  const startEdit = (key: VirtualKey) => {
    setEditingKey(key)
    setEditName(key.name)
    setEditAllowedModels(key.allowed_models?.join(', ') || '')
    setEditBudgetLimit(key.budget_limit?.toString() || '')
  }

  const cancelEdit = () => {
    setEditingKey(null)
    setEditName('')
    setEditAllowedModels('')
    setEditBudgetLimit('')
  }

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingKey) return

    setUpdating(true)

    try {
      await api.updateKey(editingKey.id, {
        name: editName !== editingKey.name ? editName : undefined,
        allowed_models: editAllowedModels ? editAllowedModels.split(',').map(m => m.trim()) : undefined,
        budget_limit: editBudgetLimit ? parseFloat(editBudgetLimit) : undefined,
      })
      fetchKeys()
      cancelEdit()
    } catch (error) {
      console.error('Failed to update key:', error)
    } finally {
      setUpdating(false)
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="flex h-screen">
      <DashboardNav />
      <main className="flex-1 overflow-auto p-8">
        <div className="mb-8 flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">API Keys</h1>
            <p className="text-muted-foreground">
              Create virtual keys to control access to your LLM providers
            </p>
          </div>
          <Button onClick={() => setShowCreateForm(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Create Key
          </Button>
        </div>

        {/* Provider Setup Notice */}
        <Card className="mb-6 bg-muted/50">
          <CardContent className="py-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Settings className="h-5 w-5 text-muted-foreground" />
                <div>
                  <p className="font-medium">Provider API Keys</p>
                  <p className="text-sm text-muted-foreground">
                    Configure your OpenAI and Anthropic API keys in Settings. All virtual keys will use those credentials.
                  </p>
                </div>
              </div>
              <Link href="/settings">
                <Button variant="outline" size="sm">
                  Configure Providers
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>

        {/* New Key Created Alert */}
        {newKey && (
          <Card className="mb-6 border-green-500">
            <CardHeader>
              <CardTitle className="text-green-600">
                Key Created Successfully!
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground mb-2">
                Copy your virtual key now. You won&apos;t be able to see it again!
              </p>
              <div className="flex items-center gap-2">
                <code className="flex-1 rounded bg-muted px-3 py-2 font-mono text-sm">
                  {newKey.virtual_key}
                </code>
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => copyToClipboard(newKey.virtual_key)}
                >
                  {copied ? (
                    <Check className="h-4 w-4 text-green-600" />
                  ) : (
                    <Copy className="h-4 w-4" />
                  )}
                </Button>
              </div>
              <Button
                variant="ghost"
                className="mt-4"
                onClick={() => setNewKey(null)}
              >
                Dismiss
              </Button>
            </CardContent>
          </Card>
        )}

        {/* Create Key Form */}
        {showCreateForm && (
          <Card className="mb-6">
            <CardHeader>
              <CardTitle>Create New Virtual Key</CardTitle>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleCreate} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="name">Key Name</Label>
                  <Input
                    id="name"
                    placeholder="Production App"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    required
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="allowedModels">Allowed Models (optional)</Label>
                  <Input
                    id="allowedModels"
                    placeholder="openai/*, anthropic/claude-3-*"
                    value={allowedModels}
                    onChange={(e) => setAllowedModels(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    Comma-separated patterns. Use * for wildcards. Leave empty to allow all models.
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="budget">Monthly Budget (USD, optional)</Label>
                  <Input
                    id="budget"
                    type="number"
                    step="0.01"
                    placeholder="100.00"
                    value={budgetLimit}
                    onChange={(e) => setBudgetLimit(e.target.value)}
                  />
                </div>

                <div className="flex gap-2">
                  <Button type="submit" disabled={creating}>
                    {creating ? 'Creating...' : 'Create Key'}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      resetCreateForm()
                      setShowCreateForm(false)
                    }}
                  >
                    Cancel
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        )}

        {/* Edit Key Form */}
        {editingKey && (
          <Card className="mb-6 border-blue-500">
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle>Edit Key: {editingKey.name}</CardTitle>
              <Button variant="ghost" size="icon" onClick={cancelEdit}>
                <X className="h-4 w-4" />
              </Button>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleUpdate} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="editName">Key Name</Label>
                  <Input
                    id="editName"
                    placeholder="Production App"
                    value={editName}
                    onChange={(e) => setEditName(e.target.value)}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="editAllowedModels">Allowed Models</Label>
                  <Input
                    id="editAllowedModels"
                    placeholder="openai/*, anthropic/claude-3-*"
                    value={editAllowedModels}
                    onChange={(e) => setEditAllowedModels(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    Comma-separated patterns. Use * for wildcards. Leave empty to allow all models.
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="editBudget">Budget Limit (USD)</Label>
                  <Input
                    id="editBudget"
                    type="number"
                    step="0.01"
                    placeholder="100.00"
                    value={editBudgetLimit}
                    onChange={(e) => setEditBudgetLimit(e.target.value)}
                  />
                </div>

                <div className="flex gap-2">
                  <Button type="submit" disabled={updating}>
                    {updating ? 'Updating...' : 'Update Key'}
                  </Button>
                  <Button type="button" variant="outline" onClick={cancelEdit}>
                    Cancel
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        )}

        {/* Keys Table */}
        <Card>
          <CardHeader>
            <CardTitle>Your Keys</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="text-center py-8 text-muted-foreground">
                Loading...
              </div>
            ) : keys.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                No keys yet. Create your first virtual key to get started.
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b text-left">
                      <th className="pb-3 font-medium">Name</th>
                      <th className="pb-3 font-medium">Allowed Models</th>
                      <th className="pb-3 font-medium">Budget</th>
                      <th className="pb-3 font-medium">Spend</th>
                      <th className="pb-3 font-medium">Status</th>
                      <th className="pb-3 font-medium">Created</th>
                      <th className="pb-3 font-medium">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {keys.map((key) => (
                      <tr key={key.id} className="border-b last:border-0">
                        <td className="py-4 font-medium">{key.name}</td>
                        <td className="py-4">
                          <div className="max-w-[200px] truncate text-sm text-muted-foreground">
                            {key.allowed_models?.length > 0
                              ? key.allowed_models.join(', ')
                              : 'All models'}
                          </div>
                        </td>
                        <td className="py-4">
                          {key.budget_limit
                            ? formatCurrency(key.budget_limit)
                            : 'Unlimited'}
                        </td>
                        <td className="py-4">
                          {formatCurrency(key.current_spend)}
                        </td>
                        <td className="py-4">
                          {key.revoked_at ? (
                            <Badge variant="destructive">Revoked</Badge>
                          ) : (
                            <Badge variant="success">Active</Badge>
                          )}
                        </td>
                        <td className="py-4 text-muted-foreground">
                          {formatDate(key.created_at)}
                        </td>
                        <td className="py-4">
                          {!key.revoked_at && (
                            <div className="flex gap-1">
                              <Button
                                variant="ghost"
                                size="icon"
                                onClick={() => startEdit(key)}
                              >
                                <Pencil className="h-4 w-4" />
                              </Button>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="text-destructive"
                                onClick={() => handleRevoke(key.id)}
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </div>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Usage Instructions */}
        <Card className="mt-6">
          <CardHeader>
            <CardTitle>API Usage</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground mb-4">
              Use the <code className="bg-muted px-1 rounded">provider/model</code> format when making requests:
            </p>
            <div className="rounded-lg bg-muted p-4 font-mono text-sm overflow-x-auto">
              <pre>{`curl -X POST http://localhost:8080/v1/chat/completions \\
  -H "Authorization: Bearer lum_your_virtual_key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "openai/gpt-4o",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'`}</pre>
            </div>
            <p className="text-sm text-muted-foreground mt-4">
              Supported model formats:
            </p>
            <ul className="text-sm text-muted-foreground mt-2 list-disc list-inside space-y-1">
              <li><code className="bg-muted px-1 rounded">openai/gpt-4o</code></li>
              <li><code className="bg-muted px-1 rounded">openai/gpt-3.5-turbo</code></li>
              <li><code className="bg-muted px-1 rounded">anthropic/claude-3-sonnet-20240229</code></li>
              <li><code className="bg-muted px-1 rounded">anthropic/claude-3-haiku-20240307</code></li>
            </ul>
          </CardContent>
        </Card>
      </main>
    </div>
  )
}
