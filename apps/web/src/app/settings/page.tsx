'use client'

import { useEffect, useState } from 'react'
import { api, ProviderInfo } from '@/lib/api'
import { DashboardNav } from '@/components/dashboard/nav'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Plus, Trash2, Key, CheckCircle } from 'lucide-react'

export default function SettingsPage() {
  const [providers, setProviders] = useState<ProviderInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [showAddDialog, setShowAddDialog] = useState(false)
  const [providerToRemove, setProviderToRemove] = useState<'openai' | 'anthropic' | null>(null)
  const [newProvider, setNewProvider] = useState<{
    provider: 'openai' | 'anthropic' | ''
    api_key: string
  }>({
    provider: '',
    api_key: '',
  })
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState('')

  const fetchProviders = async () => {
    try {
      const data = await api.listProviders()
      setProviders(data || [])
    } catch (err) {
      console.error('Failed to fetch providers:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchProviders()
  }, [])

  const handleAddProvider = async () => {
    if (!newProvider.provider || !newProvider.api_key) {
      setError('Please select a provider and enter an API key')
      return
    }

    setIsSubmitting(true)
    setError('')

    try {
      await api.setProvider({
        provider: newProvider.provider as 'openai' | 'anthropic',
        api_key: newProvider.api_key,
      })
      await fetchProviders()
      setShowAddDialog(false)
      setNewProvider({ provider: '', api_key: '' })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add provider')
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleRemoveProvider = async () => {
    if (!providerToRemove) return

    try {
      await api.removeProvider(providerToRemove)
      await fetchProviders()
      setProviderToRemove(null)
    } catch (err) {
      console.error('Failed to remove provider:', err)
    }
  }

  const isProviderConfigured = (provider: 'openai' | 'anthropic') => {
    return providers.some(p => p.provider === provider)
  }

  const getProviderDisplayName = (provider: 'openai' | 'anthropic') => {
    return provider === 'openai' ? 'OpenAI' : 'Anthropic'
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  }

  if (loading) {
    return (
      <div className="flex h-screen">
        <DashboardNav />
        <main className="flex-1 overflow-auto p-8">
          <div className="space-y-6">
            <div>
              <h1 className="text-3xl font-bold">Settings</h1>
              <p className="text-muted-foreground">Manage your account settings and API providers</p>
            </div>
            <Card>
              <CardContent className="p-6">
                <div className="animate-pulse space-y-4">
                  <div className="h-4 bg-muted rounded w-1/4"></div>
                  <div className="h-20 bg-muted rounded"></div>
                </div>
              </CardContent>
            </Card>
          </div>
        </main>
      </div>
    )
  }

  return (
    <div className="flex h-screen">
      <DashboardNav />
      <main className="flex-1 overflow-auto p-8">
        <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Settings</h1>
        <p className="text-muted-foreground">Manage your account settings and API providers</p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Key className="h-5 w-5" />
                API Providers
              </CardTitle>
              <CardDescription>
                Configure your LLM provider API keys. These keys are used by all your virtual keys.
              </CardDescription>
            </div>
            <Button onClick={() => setShowAddDialog(true)}>
              <Plus className="h-4 w-4 mr-2" />
              Add Provider
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {/* OpenAI */}
            <div className="flex items-center justify-between p-4 border rounded-lg">
              <div className="flex items-center gap-4">
                <div className="h-10 w-10 rounded-lg bg-emerald-500/10 flex items-center justify-center">
                  <span className="text-emerald-500 font-bold text-sm">OAI</span>
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium">OpenAI</span>
                    {isProviderConfigured('openai') ? (
                      <Badge variant="default" className="bg-green-500">
                        <CheckCircle className="h-3 w-3 mr-1" />
                        Configured
                      </Badge>
                    ) : (
                      <Badge variant="secondary">Not configured</Badge>
                    )}
                  </div>
                  <p className="text-sm text-muted-foreground">
                    GPT-4o, GPT-4, GPT-3.5 Turbo, and more
                  </p>
                  {providers.find(p => p.provider === 'openai') && (
                    <p className="text-xs text-muted-foreground mt-1">
                      Last updated: {formatDate(providers.find(p => p.provider === 'openai')!.updated_at)}
                    </p>
                  )}
                </div>
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setNewProvider({ provider: 'openai', api_key: '' })
                    setShowAddDialog(true)
                  }}
                >
                  {isProviderConfigured('openai') ? 'Update' : 'Configure'}
                </Button>
                {isProviderConfigured('openai') && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-destructive hover:text-destructive"
                    onClick={() => setProviderToRemove('openai')}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                )}
              </div>
            </div>

            {/* Anthropic */}
            <div className="flex items-center justify-between p-4 border rounded-lg">
              <div className="flex items-center gap-4">
                <div className="h-10 w-10 rounded-lg bg-orange-500/10 flex items-center justify-center">
                  <span className="text-orange-500 font-bold text-sm">AN</span>
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium">Anthropic</span>
                    {isProviderConfigured('anthropic') ? (
                      <Badge variant="default" className="bg-green-500">
                        <CheckCircle className="h-3 w-3 mr-1" />
                        Configured
                      </Badge>
                    ) : (
                      <Badge variant="secondary">Not configured</Badge>
                    )}
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Claude 3.5 Sonnet, Claude 3 Opus, and more
                  </p>
                  {providers.find(p => p.provider === 'anthropic') && (
                    <p className="text-xs text-muted-foreground mt-1">
                      Last updated: {formatDate(providers.find(p => p.provider === 'anthropic')!.updated_at)}
                    </p>
                  )}
                </div>
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setNewProvider({ provider: 'anthropic', api_key: '' })
                    setShowAddDialog(true)
                  }}
                >
                  {isProviderConfigured('anthropic') ? 'Update' : 'Configure'}
                </Button>
                {isProviderConfigured('anthropic') && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-destructive hover:text-destructive"
                    onClick={() => setProviderToRemove('anthropic')}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                )}
              </div>
            </div>
          </div>

          <div className="mt-6 p-4 bg-muted/50 rounded-lg">
            <h4 className="font-medium mb-2">How it works</h4>
            <ul className="text-sm text-muted-foreground space-y-1">
              <li>- Your API keys are encrypted and stored securely</li>
              <li>- All your virtual keys will use these provider API keys</li>
              <li>- Virtual keys only control access (allowed models + budget limits)</li>
              <li>- You can update or remove provider keys at any time</li>
            </ul>
          </div>
        </CardContent>
      </Card>

      {/* Add Provider Dialog */}
      <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {newProvider.provider ? `Configure ${getProviderDisplayName(newProvider.provider as 'openai' | 'anthropic')}` : 'Add API Provider'}
            </DialogTitle>
            <DialogDescription>
              Enter your API key. It will be encrypted and stored securely.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            {!newProvider.provider && (
              <div className="space-y-2">
                <Label htmlFor="provider">Provider</Label>
                <Select
                  value={newProvider.provider}
                  onValueChange={(value) => setNewProvider({ ...newProvider, provider: value as 'openai' | 'anthropic' })}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select a provider" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="openai">OpenAI</SelectItem>
                    <SelectItem value="anthropic">Anthropic</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="api_key">API Key</Label>
              <Input
                id="api_key"
                type="password"
                placeholder={newProvider.provider === 'openai' ? 'sk-...' : 'sk-ant-...'}
                value={newProvider.api_key}
                onChange={(e) => setNewProvider({ ...newProvider, api_key: e.target.value })}
              />
              <p className="text-xs text-muted-foreground">
                {newProvider.provider === 'openai'
                  ? 'Get your API key from platform.openai.com'
                  : newProvider.provider === 'anthropic'
                  ? 'Get your API key from console.anthropic.com'
                  : 'Select a provider to see instructions'}
              </p>
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => {
              setShowAddDialog(false)
              setNewProvider({ provider: '', api_key: '' })
              setError('')
            }}>
              Cancel
            </Button>
            <Button onClick={handleAddProvider} disabled={isSubmitting}>
              {isSubmitting ? 'Saving...' : 'Save'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Remove Provider Confirmation */}
      <AlertDialog open={!!providerToRemove} onOpenChange={() => setProviderToRemove(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove {providerToRemove ? getProviderDisplayName(providerToRemove) : ''} API Key?</AlertDialogTitle>
            <AlertDialogDescription>
              This will remove your API key for {providerToRemove ? getProviderDisplayName(providerToRemove) : ''}.
              Your virtual keys will no longer be able to access {providerToRemove ? getProviderDisplayName(providerToRemove) : ''} models.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleRemoveProvider} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
              Remove
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
        </div>
      </main>
    </div>
  )
}
