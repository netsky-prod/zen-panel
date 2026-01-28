import { useState } from 'react'
import { Copy, Check, Link, ExternalLink } from 'lucide-react'
import Modal from './Modal'
import QRCode from './QRCode'
import type { UserConfig } from '../types'
import { clsx } from 'clsx'

interface ConfigModalProps {
  isOpen: boolean
  onClose: () => void
  config: UserConfig | null
  userName: string
  userUUID?: string
}

type TabType = 'subscription' | 'url' | 'qr' | 'singbox'

export default function ConfigModal({
  isOpen,
  onClose,
  config,
  userName,
  userUUID,
}: ConfigModalProps) {
  const [activeTab, setActiveTab] = useState<TabType>('subscription')
  const [copied, setCopied] = useState<string | null>(null)
  const [selectedUrl, setSelectedUrl] = useState(0)

  const copyToClipboard = async (text: string, key: string) => {
    await navigator.clipboard.writeText(text)
    setCopied(key)
    setTimeout(() => setCopied(null), 2000)
  }

  if (!config) return null

  const tabs: { id: TabType; label: string }[] = [
    { id: 'subscription', label: 'Subscription' },
    { id: 'url', label: 'Share URL' },
    { id: 'qr', label: 'QR Code' },
    { id: 'singbox', label: 'sing-box JSON' },
  ]

  const currentUrl = config.share_urls?.[selectedUrl]?.url || config.share_url

  // Build subscription URL (public endpoint)
  const baseUrl = window.location.origin.replace(':3000', ':8080')
  const subscriptionUrl = userUUID ? `${baseUrl}/api/sub/${userUUID}` : null
  const configUrl = userUUID ? `${baseUrl}/api/sub/${userUUID}/config` : null

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`Config for ${userName}`}
      size="lg"
    >
      {/* Tabs */}
      <div className="mb-4 flex gap-1 rounded-lg bg-dark-800 p-1">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={clsx(
              'flex-1 rounded-md px-3 py-2 text-sm font-medium transition-colors',
              activeTab === tab.id
                ? 'bg-dark-700 text-white'
                : 'text-dark-400 hover:text-white'
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Content */}
      {activeTab === 'subscription' && (
        <div className="space-y-4">
          <div className="rounded-lg bg-dark-800 p-4">
            <h3 className="text-sm font-medium text-white mb-2 flex items-center gap-2">
              <Link className="h-4 w-4 text-blue-400" />
              Subscription URL (for apps)
            </h3>
            <p className="text-xs text-dark-400 mb-3">
              Send this link to user. Works with Shadowrocket, v2rayNG, Clash, etc.
            </p>
            {subscriptionUrl ? (
              <div className="flex gap-2">
                <input
                  readOnly
                  value={subscriptionUrl}
                  className="input flex-1 font-mono text-xs"
                />
                <button
                  onClick={() => copyToClipboard(subscriptionUrl, 'sub')}
                  className="btn-secondary btn-sm"
                >
                  {copied === 'sub' ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                </button>
              </div>
            ) : (
              <p className="text-dark-500 text-sm">UUID not available</p>
            )}
          </div>

          <div className="rounded-lg bg-dark-800 p-4">
            <h3 className="text-sm font-medium text-white mb-2 flex items-center gap-2">
              <ExternalLink className="h-4 w-4 text-green-400" />
              sing-box Config URL
            </h3>
            <p className="text-xs text-dark-400 mb-3">
              Direct JSON config for sing-box clients
            </p>
            {configUrl ? (
              <div className="flex gap-2">
                <input
                  readOnly
                  value={configUrl}
                  className="input flex-1 font-mono text-xs"
                />
                <button
                  onClick={() => copyToClipboard(configUrl, 'config')}
                  className="btn-secondary btn-sm"
                >
                  {copied === 'config' ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                </button>
              </div>
            ) : (
              <p className="text-dark-500 text-sm">UUID not available</p>
            )}
          </div>

          <div className="rounded-lg bg-blue-900/30 border border-blue-700/50 p-4">
            <p className="text-sm text-blue-200">
              <strong>Tip:</strong> Send the Subscription URL to users. They can add it to their VPN app and it will auto-update when you change server settings.
            </p>
          </div>
        </div>
      )}

      {activeTab === 'url' && (
        <div className="space-y-4">
          {config.share_urls && config.share_urls.length > 1 && (
            <div>
              <label className="label">Select Inbound</label>
              <select
                value={selectedUrl}
                onChange={(e) => setSelectedUrl(Number(e.target.value))}
                className="select"
              >
                {config.share_urls.map((url, index) => (
                  <option key={index} value={index}>
                    {url.node_name} - {url.inbound_name}
                  </option>
                ))}
              </select>
            </div>
          )}
          <div className="relative">
            <textarea
              readOnly
              value={currentUrl}
              className="input h-32 resize-none font-mono text-xs"
            />
            <button
              onClick={() => copyToClipboard(currentUrl, 'url')}
              className="absolute right-2 top-2 btn-secondary btn-sm"
            >
              {copied === 'url' ? (
                <>
                  <Check className="h-4 w-4" />
                  Copied
                </>
              ) : (
                <>
                  <Copy className="h-4 w-4" />
                  Copy
                </>
              )}
            </button>
          </div>
          <p className="text-xs text-dark-400">
            This is a direct vless:// or hysteria2:// URL for importing into VPN clients
          </p>
        </div>
      )}

      {activeTab === 'qr' && (
        <div className="space-y-4">
          {config.share_urls && config.share_urls.length > 1 && (
            <div>
              <label className="label">Select Inbound</label>
              <select
                value={selectedUrl}
                onChange={(e) => setSelectedUrl(Number(e.target.value))}
                className="select"
              >
                {config.share_urls.map((url, index) => (
                  <option key={index} value={index}>
                    {url.node_name} - {url.inbound_name}
                  </option>
                ))}
              </select>
            </div>
          )}
          <div className="flex justify-center">
            <QRCode value={currentUrl} size={250} />
          </div>
          <p className="text-center text-sm text-dark-400">
            Scan this QR code with your VPN client
          </p>
        </div>
      )}

      {activeTab === 'singbox' && (
        <div className="relative">
          <pre className="max-h-96 overflow-auto rounded-lg bg-dark-800 p-4 text-sm text-dark-200">
            {JSON.stringify(config.singbox, null, 2)}
          </pre>
          <button
            onClick={() => copyToClipboard(JSON.stringify(config.singbox, null, 2), 'json')}
            className="absolute right-2 top-2 btn-secondary btn-sm"
          >
            {copied === 'json' ? (
              <>
                <Check className="h-4 w-4" />
                Copied
              </>
            ) : (
              <>
                <Copy className="h-4 w-4" />
                Copy
              </>
            )}
          </button>
        </div>
      )}
    </Modal>
  )
}
