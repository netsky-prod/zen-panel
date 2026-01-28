import { useState } from 'react'
import { Loader2 } from 'lucide-react'
import type { Node, CreateNodeInput } from '../types'

interface NodeFormProps {
  node?: Node | null
  onSubmit: (data: CreateNodeInput) => void
  onCancel: () => void
  isLoading?: boolean
}

export default function NodeForm({
  node,
  onSubmit,
  onCancel,
  isLoading = false,
}: NodeFormProps) {
  const [name, setName] = useState(node?.name || '')
  const [address, setAddress] = useState(node?.address || '')
  const [apiPort, setApiPort] = useState(node?.api_port?.toString() || '9090')
  const [apiToken, setApiToken] = useState(node?.api_token || '')
  const [enabled, setEnabled] = useState(node?.enabled ?? true)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit({
      name,
      address,
      api_port: parseInt(apiPort, 10),
      api_token: apiToken,
      enabled,
    })
  }

  const generateToken = () => {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
    let token = ''
    for (let i = 0; i < 32; i++) {
      token += chars.charAt(Math.floor(Math.random() * chars.length))
    }
    setApiToken(token)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label htmlFor="name" className="label">
          Node Name
        </label>
        <input
          id="name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="input"
          placeholder="e.g., Frankfurt-1"
          required
        />
      </div>

      <div>
        <label htmlFor="address" className="label">
          Address
        </label>
        <input
          id="address"
          type="text"
          value={address}
          onChange={(e) => setAddress(e.target.value)}
          className="input"
          placeholder="e.g., vpn.example.com or 1.2.3.4"
          required
        />
        <p className="mt-1 text-xs text-dark-400">
          Domain or IP address of the node
        </p>
      </div>

      <div>
        <label htmlFor="apiPort" className="label">
          API Port
        </label>
        <input
          id="apiPort"
          type="number"
          min="1"
          max="65535"
          value={apiPort}
          onChange={(e) => setApiPort(e.target.value)}
          className="input"
          required
        />
        <p className="mt-1 text-xs text-dark-400">
          Port where the node agent is running
        </p>
      </div>

      <div>
        <label htmlFor="apiToken" className="label">
          API Token
        </label>
        <div className="flex gap-2">
          <input
            id="apiToken"
            type="text"
            value={apiToken}
            onChange={(e) => setApiToken(e.target.value)}
            className="input font-mono"
            placeholder="Token for node authentication"
            required
          />
          <button
            type="button"
            onClick={generateToken}
            className="btn-secondary whitespace-nowrap"
          >
            Generate
          </button>
        </div>
        <p className="mt-1 text-xs text-dark-400">
          This token must match the one configured on the node agent
        </p>
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
          Node enabled
        </label>
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
          ) : node ? (
            'Update Node'
          ) : (
            'Create Node'
          )}
        </button>
      </div>
    </form>
  )
}
