import { useState } from 'react'
import { Loader2, Key } from 'lucide-react'
import { inboundsApi } from '../api/client'
import { useToast } from '../hooks/useToast'
import type { Inbound, CreateInboundInput } from '../types'

interface InboundFormProps {
  inbound?: Inbound | null
  nodeId: number
  onSubmit: (data: CreateInboundInput) => void
  onCancel: () => void
  isLoading?: boolean
}

type Protocol = 'reality' | 'ws-tls' | 'hysteria2'

export default function InboundForm({
  inbound,
  nodeId,
  onSubmit,
  onCancel,
  isLoading = false,
}: InboundFormProps) {
  const addToast = useToast((state) => state.addToast)
  const [generatingKeys, setGeneratingKeys] = useState(false)

  const [name, setName] = useState(inbound?.name || '')
  const [protocol, setProtocol] = useState<Protocol>(inbound?.protocol || 'reality')
  const [listenPort, setListenPort] = useState(inbound?.listen_port?.toString() || '443')
  const [enabled, setEnabled] = useState(inbound?.enabled ?? true)

  // REALITY fields
  const [sni, setSni] = useState(inbound?.sni || '')
  const [fallbackAddr, setFallbackAddr] = useState(inbound?.fallback_addr || '127.0.0.1')
  const [fallbackPort, setFallbackPort] = useState(inbound?.fallback_port?.toString() || '8443')
  const [privateKey, setPrivateKey] = useState(inbound?.private_key || '')
  const [publicKey, setPublicKey] = useState(inbound?.public_key || '')
  const [shortId, setShortId] = useState(inbound?.short_id || '')
  const [fingerprint, setFingerprint] = useState(inbound?.fingerprint || 'chrome')

  // WS fields
  const [wsPath, setWsPath] = useState(inbound?.ws_path || '/ws')

  // Hysteria2 fields
  const [upMbps, setUpMbps] = useState(inbound?.up_mbps?.toString() || '100')
  const [downMbps, setDownMbps] = useState(inbound?.down_mbps?.toString() || '100')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    const base: CreateInboundInput = {
      node_id: nodeId,
      name,
      protocol,
      listen_port: parseInt(listenPort, 10),
      enabled,
    }

    if (protocol === 'reality') {
      onSubmit({
        ...base,
        sni,
        fallback_addr: fallbackAddr,
        fallback_port: parseInt(fallbackPort, 10),
        private_key: privateKey,
        public_key: publicKey,
        short_id: shortId,
        fingerprint,
      })
    } else if (protocol === 'ws-tls') {
      onSubmit({
        ...base,
        sni,
        ws_path: wsPath,
      })
    } else if (protocol === 'hysteria2') {
      onSubmit({
        ...base,
        up_mbps: parseInt(upMbps, 10),
        down_mbps: parseInt(downMbps, 10),
      })
    }
  }

  const generateKeys = async () => {
    if (!inbound?.id) {
      addToast('error', 'Save the inbound first to generate keys')
      return
    }
    setGeneratingKeys(true)
    try {
      const keys = await inboundsApi.generateKeys(inbound.id)
      setPrivateKey(keys.private_key)
      setPublicKey(keys.public_key)
      setShortId(keys.short_id)
      addToast('success', 'REALITY keys generated')
    } catch (err) {
      addToast('error', err instanceof Error ? err.message : 'Failed to generate keys')
    } finally {
      setGeneratingKeys(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label htmlFor="name" className="label">
          Inbound Name
        </label>
        <input
          id="name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="input"
          placeholder="e.g., REALITY-443"
          required
        />
      </div>

      <div>
        <label htmlFor="protocol" className="label">
          Protocol
        </label>
        <select
          id="protocol"
          value={protocol}
          onChange={(e) => setProtocol(e.target.value as Protocol)}
          className="select"
          disabled={!!inbound}
        >
          <option value="reality">VLESS + REALITY</option>
          <option value="ws-tls">VLESS + WebSocket + TLS</option>
          <option value="hysteria2">Hysteria2</option>
        </select>
      </div>

      <div>
        <label htmlFor="listenPort" className="label">
          Listen Port
        </label>
        <input
          id="listenPort"
          type="number"
          min="1"
          max="65535"
          value={listenPort}
          onChange={(e) => setListenPort(e.target.value)}
          className="input"
          required
        />
      </div>

      {/* REALITY specific fields */}
      {protocol === 'reality' && (
        <>
          <div>
            <label htmlFor="sni" className="label">
              SNI (Server Name Indication)
            </label>
            <input
              id="sni"
              type="text"
              value={sni}
              onChange={(e) => setSni(e.target.value)}
              className="input"
              placeholder="e.g., www.google.com"
              required
            />
            <p className="mt-1 text-xs text-dark-400">
              Domain to impersonate for anti-DPI
            </p>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label htmlFor="fallbackAddr" className="label">
                Fallback Address
              </label>
              <input
                id="fallbackAddr"
                type="text"
                value={fallbackAddr}
                onChange={(e) => setFallbackAddr(e.target.value)}
                className="input"
                required
              />
            </div>
            <div>
              <label htmlFor="fallbackPort" className="label">
                Fallback Port
              </label>
              <input
                id="fallbackPort"
                type="number"
                min="1"
                max="65535"
                value={fallbackPort}
                onChange={(e) => setFallbackPort(e.target.value)}
                className="input"
                required
              />
            </div>
          </div>

          <div>
            <label htmlFor="fingerprint" className="label">
              TLS Fingerprint
            </label>
            <select
              id="fingerprint"
              value={fingerprint}
              onChange={(e) => setFingerprint(e.target.value)}
              className="select"
            >
              <option value="chrome">Chrome</option>
              <option value="firefox">Firefox</option>
              <option value="safari">Safari</option>
              <option value="edge">Edge</option>
              <option value="random">Random</option>
            </select>
          </div>

          <div className="space-y-4 rounded-lg border border-dark-700 bg-dark-800 p-4">
            <div className="flex items-center justify-between">
              <span className="font-medium text-dark-200">REALITY Keys</span>
              {inbound?.id && (
                <button
                  type="button"
                  onClick={generateKeys}
                  disabled={generatingKeys}
                  className="btn-secondary btn-sm"
                >
                  {generatingKeys ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Key className="h-4 w-4" />
                  )}
                  Generate Keys
                </button>
              )}
            </div>
            <div>
              <label htmlFor="privateKey" className="label">
                Private Key
              </label>
              <input
                id="privateKey"
                type="text"
                value={privateKey}
                onChange={(e) => setPrivateKey(e.target.value)}
                className="input font-mono text-xs"
                placeholder="Base64 private key"
              />
            </div>
            <div>
              <label htmlFor="publicKey" className="label">
                Public Key
              </label>
              <input
                id="publicKey"
                type="text"
                value={publicKey}
                onChange={(e) => setPublicKey(e.target.value)}
                className="input font-mono text-xs"
                placeholder="Base64 public key"
              />
            </div>
            <div>
              <label htmlFor="shortId" className="label">
                Short ID
              </label>
              <input
                id="shortId"
                type="text"
                value={shortId}
                onChange={(e) => setShortId(e.target.value)}
                className="input font-mono"
                placeholder="e.g., abc123"
                maxLength={16}
              />
            </div>
          </div>
        </>
      )}

      {/* WS-TLS specific fields */}
      {protocol === 'ws-tls' && (
        <>
          <div>
            <label htmlFor="sni" className="label">
              SNI (Server Name Indication)
            </label>
            <input
              id="sni"
              type="text"
              value={sni}
              onChange={(e) => setSni(e.target.value)}
              className="input"
              placeholder="e.g., vpn.example.com"
              required
            />
          </div>
          <div>
            <label htmlFor="wsPath" className="label">
              WebSocket Path
            </label>
            <input
              id="wsPath"
              type="text"
              value={wsPath}
              onChange={(e) => setWsPath(e.target.value)}
              className="input"
              placeholder="/ws"
              required
            />
          </div>
        </>
      )}

      {/* Hysteria2 specific fields */}
      {protocol === 'hysteria2' && (
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label htmlFor="upMbps" className="label">
              Upload Speed (Mbps)
            </label>
            <input
              id="upMbps"
              type="number"
              min="1"
              value={upMbps}
              onChange={(e) => setUpMbps(e.target.value)}
              className="input"
              required
            />
          </div>
          <div>
            <label htmlFor="downMbps" className="label">
              Download Speed (Mbps)
            </label>
            <input
              id="downMbps"
              type="number"
              min="1"
              value={downMbps}
              onChange={(e) => setDownMbps(e.target.value)}
              className="input"
              required
            />
          </div>
        </div>
      )}

      <div className="flex items-center gap-3">
        <input
          id="enabled"
          type="checkbox"
          checked={enabled}
          onChange={(e) => setEnabled(e.target.checked)}
          className="h-4 w-4 rounded border-dark-600 bg-dark-800 text-blue-600 focus:ring-blue-500"
        />
        <label htmlFor="enabled" className="text-sm text-dark-200">
          Inbound enabled
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
          ) : inbound ? (
            'Update Inbound'
          ) : (
            'Create Inbound'
          )}
        </button>
      </div>
    </form>
  )
}
