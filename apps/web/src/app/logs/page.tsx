'use client'

import { useEffect, useState } from 'react'
import { DashboardNav } from '@/components/dashboard/nav'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { api, type LogEntry } from '@/lib/api'
import { formatDate, formatCurrency } from '@/lib/utils'
import { Search, X, ChevronLeft, ChevronRight } from 'lucide-react'

export default function LogsPage() {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(0)
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [selectedLog, setSelectedLog] = useState<LogEntry | null>(null)

  const pageSize = 20

  useEffect(() => {
    fetchLogs()
  }, [page])

  const fetchLogs = async () => {
    setLoading(true)
    try {
      const result = await api.searchLogs({
        q: search || undefined,
        page,
        size: pageSize,
      })
      setLogs(result.entries || [])
      setTotal(result.total)
    } catch (error) {
      console.error('Failed to fetch logs:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    setPage(0)
    fetchLogs()
  }

  const getStatusBadge = (statusCode: number) => {
    if (statusCode >= 200 && statusCode < 300) {
      return <Badge variant="success">{statusCode}</Badge>
    } else if (statusCode >= 400 && statusCode < 500) {
      return <Badge variant="warning">{statusCode}</Badge>
    } else if (statusCode >= 500) {
      return <Badge variant="destructive">{statusCode}</Badge>
    }
    return <Badge>{statusCode}</Badge>
  }

  const totalPages = Math.ceil(total / pageSize)

  return (
    <div className="flex h-screen">
      <DashboardNav />
      <main className="flex-1 overflow-auto p-8">
        <div className="mb-8">
          <h1 className="text-3xl font-bold">Request Logs</h1>
          <p className="text-muted-foreground">
            Search and explore your API request history
          </p>
        </div>

        {/* Search */}
        <Card className="mb-6">
          <CardContent className="pt-6">
            <form onSubmit={handleSearch} className="flex gap-4">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder="Search prompts and responses..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="pl-9"
                />
              </div>
              <Button type="submit">Search</Button>
            </form>
          </CardContent>
        </Card>

        {/* Logs Table */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>Logs ({total})</CardTitle>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="icon"
                disabled={page === 0}
                onClick={() => setPage((p) => Math.max(0, p - 1))}
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <span className="text-sm text-muted-foreground">
                Page {page + 1} of {totalPages || 1}
              </span>
              <Button
                variant="outline"
                size="icon"
                disabled={page >= totalPages - 1}
                onClick={() => setPage((p) => p + 1)}
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="text-center py-8 text-muted-foreground">
                Loading...
              </div>
            ) : logs.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                No logs found. Start making API requests to see them here.
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b text-left">
                      <th className="pb-3 font-medium">Time</th>
                      <th className="pb-3 font-medium">Key</th>
                      <th className="pb-3 font-medium">Model</th>
                      <th className="pb-3 font-medium">Status</th>
                      <th className="pb-3 font-medium">Latency</th>
                      <th className="pb-3 font-medium">Tokens</th>
                      <th className="pb-3 font-medium">Cost</th>
                    </tr>
                  </thead>
                  <tbody>
                    {logs.map((log) => (
                      <tr
                        key={log.trace_id}
                        className="border-b last:border-0 cursor-pointer hover:bg-muted/50"
                        onClick={() => setSelectedLog(log)}
                      >
                        <td className="py-4 text-sm">
                          {formatDate(log.timestamp)}
                        </td>
                        <td className="py-4 text-sm font-medium">
                          {log.virtual_key_name}
                        </td>
                        <td className="py-4">
                          <Badge variant="outline">{log.request.model}</Badge>
                        </td>
                        <td className="py-4">
                          {getStatusBadge(log.response.status_code)}
                        </td>
                        <td className="py-4 text-sm">
                          {log.metrics.latency_ms}ms
                        </td>
                        <td className="py-4 text-sm">
                          {log.response.usage.total_tokens}
                        </td>
                        <td className="py-4 text-sm">
                          {formatCurrency(log.metrics.cost_usd)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Log Detail Side Panel */}
        {selectedLog && (
          <div className="fixed inset-y-0 right-0 w-[600px] bg-background border-l shadow-lg z-50 overflow-auto">
            <div className="sticky top-0 bg-background border-b p-4 flex items-center justify-between">
              <h2 className="text-lg font-semibold">Request Details</h2>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setSelectedLog(null)}
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
            <div className="p-6 space-y-6">
              <div>
                <h3 className="text-sm font-medium text-muted-foreground mb-2">
                  Trace ID
                </h3>
                <code className="text-sm">{selectedLog.trace_id}</code>
              </div>

              <div>
                <h3 className="text-sm font-medium text-muted-foreground mb-2">
                  Metrics
                </h3>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-muted-foreground">Latency</p>
                    <p className="font-medium">
                      {selectedLog.metrics.latency_ms}ms
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Cost</p>
                    <p className="font-medium">
                      {formatCurrency(selectedLog.metrics.cost_usd)}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">
                      Prompt Tokens
                    </p>
                    <p className="font-medium">
                      {selectedLog.response.usage.prompt_tokens}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">
                      Completion Tokens
                    </p>
                    <p className="font-medium">
                      {selectedLog.response.usage.completion_tokens}
                    </p>
                  </div>
                </div>
              </div>

              <div>
                <h3 className="text-sm font-medium text-muted-foreground mb-2">
                  Request
                </h3>
                <pre className="bg-muted rounded-lg p-4 text-sm overflow-auto max-h-[300px]">
                  {JSON.stringify(selectedLog.request, null, 2)}
                </pre>
              </div>

              <div>
                <h3 className="text-sm font-medium text-muted-foreground mb-2">
                  Response
                </h3>
                <pre className="bg-muted rounded-lg p-4 text-sm overflow-auto max-h-[300px]">
                  {selectedLog.response.content || 'No content'}
                </pre>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}
