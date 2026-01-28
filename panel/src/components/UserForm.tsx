import { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { nodesApi, inboundsApi } from '../api/client'
import type { User, CreateUserInput, Inbound, Node } from '../types'

interface UserFormProps {
  user?: User | null
  onSubmit: (data: CreateUserInput) => void
  onCancel: () => void
  isLoading?: boolean
}

export default function UserForm({
  user,
  onSubmit,
  onCancel,
  isLoading = false,
}: UserFormProps) {
  const [name, setName] = useState(user?.name || '')
  const [enabled, setEnabled] = useState(user?.enabled ?? true)
  const [dataLimit, setDataLimit] = useState(
    user?.data_limit ? (user.data_limit / (1024 * 1024 * 1024)).toString() : '0'
  )
  const [expiresAt, setExpiresAt] = useState(
    user?.expires_at ? user.expires_at.split('T')[0] : ''
  )
  const [selectedInbounds, setSelectedInbounds] = useState<number[]>(
    user?.inbounds?.map((i) => i.id) || []
  )

  const { data: nodes } = useQuery({
    queryKey: ['nodes'],
    queryFn: nodesApi.list,
  })

  const [nodeInbounds, setNodeInbounds] = useState<Record<number, Inbound[]>>({})

  useEffect(() => {
    if (nodes) {
      nodes.forEach(async (node: Node) => {
        const inbounds = await inboundsApi.listByNode(node.id)
        setNodeInbounds((prev) => ({ ...prev, [node.id]: inbounds }))
      })
    }
  }, [nodes])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit({
      name,
      enabled,
      data_limit: parseFloat(dataLimit) * 1024 * 1024 * 1024, // GB to bytes
      expires_at: expiresAt || null,
      inbound_ids: selectedInbounds,
    })
  }

  const toggleInbound = (inboundId: number) => {
    setSelectedInbounds((prev) =>
      prev.includes(inboundId)
        ? prev.filter((id) => id !== inboundId)
        : [...prev, inboundId]
    )
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label htmlFor="name" className="label">
          Username
        </label>
        <input
          id="name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="input"
          placeholder="Enter username"
          required
          disabled={!!user}
        />
      </div>

      <div className="flex items-center gap-3">
        <input
          id="enabled"
          type="checkbox"
          checked={enabled}
          onChange={(e) => setEnabled(e.target.checked)}
          className="h-4 w-4 rounded border-dark-600 bg-dark-800 text-blue-600 focus:ring-blue-500"
        />
        <label htmlFor="enabled" className="text-sm text-dark-200">
          Account enabled
        </label>
      </div>

      <div>
        <label htmlFor="dataLimit" className="label">
          Data Limit (GB)
        </label>
        <input
          id="dataLimit"
          type="number"
          min="0"
          step="0.1"
          value={dataLimit}
          onChange={(e) => setDataLimit(e.target.value)}
          className="input"
          placeholder="0 = unlimited"
        />
        <p className="mt-1 text-xs text-dark-400">Set to 0 for unlimited</p>
      </div>

      <div>
        <label htmlFor="expiresAt" className="label">
          Expiry Date
        </label>
        <input
          id="expiresAt"
          type="date"
          value={expiresAt}
          onChange={(e) => setExpiresAt(e.target.value)}
          className="input"
        />
        <p className="mt-1 text-xs text-dark-400">Leave empty for no expiry</p>
      </div>

      <div>
        <label className="label">Inbounds</label>
        <div className="space-y-4 rounded-lg border border-dark-700 bg-dark-800 p-4">
          {nodes && nodes.length > 0 ? (
            nodes.map((node: Node) => (
              <div key={node.id}>
                <p className="mb-2 font-medium text-dark-200">{node.name}</p>
                <div className="ml-4 space-y-2">
                  {nodeInbounds[node.id]?.map((inbound: Inbound) => (
                    <label
                      key={inbound.id}
                      className="flex items-center gap-3 cursor-pointer"
                    >
                      <input
                        type="checkbox"
                        checked={selectedInbounds.includes(inbound.id)}
                        onChange={() => toggleInbound(inbound.id)}
                        className="h-4 w-4 rounded border-dark-600 bg-dark-700 text-blue-600 focus:ring-blue-500"
                      />
                      <span className="text-sm text-dark-300">
                        {inbound.name}{' '}
                        <span className="text-dark-500">({inbound.protocol})</span>
                      </span>
                    </label>
                  )) || (
                    <p className="text-sm text-dark-500">No inbounds</p>
                  )}
                </div>
              </div>
            ))
          ) : (
            <p className="text-sm text-dark-400">No nodes available</p>
          )}
        </div>
      </div>

      <div className="flex gap-3 pt-4">
        <button
          type="button"
          onClick={onCancel}
          disabled={isLoading}
          className="btn-secondary flex-1"
        >
          Cancel
        </button>
        <button type="submit" disabled={isLoading} className="btn-primary flex-1">
          {isLoading ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              Saving...
            </>
          ) : user ? (
            'Update User'
          ) : (
            'Create User'
          )}
        </button>
      </div>
    </form>
  )
}
