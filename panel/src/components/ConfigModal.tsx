import { useState } from 'react'
import { Copy, Check } from 'lucide-react'
import Modal from './Modal'
import QRCode from './QRCode'
import type { UserConfig } from '../types'
import { clsx } from 'clsx'

interface ConfigModalProps {
  isOpen: boolean
  onClose: () => void
  config: UserConfig | null
  userName: string
}

type TabType = 'singbox' | 'url' | 'qr'

export default function ConfigModal({
  isOpen,
  onClose,
  config,
  userName,
}: ConfigModalProps) {
  const [activeTab, setActiveTab] = useState<TabType>('singbox')
  const [copied, setCopied] = useState(false)
  const [selectedUrl, setSelectedUrl] = useState(0)

  const copyToClipboard = async (text: string) => {
    await navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  if (!config) return null

  const tabs: { id: TabType; label: string }[] = [
    { id: 'singbox', label: 'sing-box JSON' },
    { id: 'url', label: 'Share URL' },
    { id: 'qr', label: 'QR Code' },
  ]

  const currentUrl = config.share_urls?.[selectedUrl]?.url || config.share_url

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
      {activeTab === 'singbox' && (
        <div className="relative">
          <pre className="max-h-96 overflow-auto rounded-lg bg-dark-800 p-4 text-sm text-dark-200">
            {JSON.stringify(config.singbox, null, 2)}
          </pre>
          <button
            onClick={() => copyToClipboard(JSON.stringify(config.singbox, null, 2))}
            className="absolute right-2 top-2 btn-secondary btn-sm"
          >
            {copied ? (
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
              onClick={() => copyToClipboard(currentUrl)}
              className="absolute right-2 top-2 btn-secondary btn-sm"
            >
              {copied ? (
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
    </Modal>
  )
}
