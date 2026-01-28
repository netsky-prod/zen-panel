import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Loader2, Lock, Server, Database, Clock } from 'lucide-react'
import { authApi } from '../api/client'
import { useToast } from '../hooks/useToast'

export default function Settings() {
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [passwordError, setPasswordError] = useState('')

  const addToast = useToast((state) => state.addToast)

  const changePasswordMutation = useMutation({
    mutationFn: () => authApi.changePassword(oldPassword, newPassword),
    onSuccess: () => {
      addToast('success', 'Password changed successfully')
      setOldPassword('')
      setNewPassword('')
      setConfirmPassword('')
    },
    onError: (err: Error) => addToast('error', err.message),
  })

  const handlePasswordChange = (e: React.FormEvent) => {
    e.preventDefault()
    setPasswordError('')

    if (newPassword.length < 8) {
      setPasswordError('Password must be at least 8 characters')
      return
    }

    if (newPassword !== confirmPassword) {
      setPasswordError('Passwords do not match')
      return
    }

    changePasswordMutation.mutate()
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Settings</h1>
        <p className="mt-1 text-dark-400">Manage your account and system settings</p>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Change Password */}
        <div className="card">
          <div className="mb-6 flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-dark-800">
              <Lock className="h-5 w-5 text-dark-300" />
            </div>
            <div>
              <h2 className="font-semibold text-white">Change Password</h2>
              <p className="text-sm text-dark-400">Update your admin password</p>
            </div>
          </div>

          <form onSubmit={handlePasswordChange} className="space-y-4">
            {passwordError && (
              <div className="rounded-lg bg-red-900/50 border border-red-700 px-4 py-3 text-sm text-red-200">
                {passwordError}
              </div>
            )}

            <div>
              <label htmlFor="oldPassword" className="label">
                Current Password
              </label>
              <input
                id="oldPassword"
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                className="input"
                required
              />
            </div>

            <div>
              <label htmlFor="newPassword" className="label">
                New Password
              </label>
              <input
                id="newPassword"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                className="input"
                required
                minLength={8}
              />
            </div>

            <div>
              <label htmlFor="confirmPassword" className="label">
                Confirm New Password
              </label>
              <input
                id="confirmPassword"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="input"
                required
              />
            </div>

            <button
              type="submit"
              disabled={changePasswordMutation.isPending}
              className="btn-primary w-full"
            >
              {changePasswordMutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Changing...
                </>
              ) : (
                'Change Password'
              )}
            </button>
          </form>
        </div>

        {/* System Info */}
        <div className="card">
          <div className="mb-6 flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-dark-800">
              <Server className="h-5 w-5 text-dark-300" />
            </div>
            <div>
              <h2 className="font-semibold text-white">System Information</h2>
              <p className="text-sm text-dark-400">Server and application details</p>
            </div>
          </div>

          <div className="space-y-4">
            <div className="flex items-center justify-between rounded-lg bg-dark-800 p-4">
              <div className="flex items-center gap-3">
                <Server className="h-5 w-5 text-dark-400" />
                <span className="text-dark-300">Application</span>
              </div>
              <span className="font-mono text-white">Zen VPN Panel</span>
            </div>

            <div className="flex items-center justify-between rounded-lg bg-dark-800 p-4">
              <div className="flex items-center gap-3">
                <Database className="h-5 w-5 text-dark-400" />
                <span className="text-dark-300">Version</span>
              </div>
              <span className="font-mono text-white">1.0.0</span>
            </div>

            <div className="flex items-center justify-between rounded-lg bg-dark-800 p-4">
              <div className="flex items-center gap-3">
                <Clock className="h-5 w-5 text-dark-400" />
                <span className="text-dark-300">Server Time</span>
              </div>
              <span className="font-mono text-white">
                {new Date().toLocaleString()}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* About */}
      <div className="card">
        <h2 className="mb-4 font-semibold text-white">About Zen VPN Panel</h2>
        <p className="text-dark-300">
          Zen VPN Panel is a centralized management system for VPN infrastructure
          supporting multiple protocols including VLESS + REALITY, VLESS + WebSocket,
          and Hysteria2. It provides a modern web interface for managing users, nodes,
          and traffic monitoring.
        </p>
        <div className="mt-4 flex flex-wrap gap-2">
          <span className="rounded-full bg-dark-800 px-3 py-1 text-xs text-dark-300">
            sing-box
          </span>
          <span className="rounded-full bg-dark-800 px-3 py-1 text-xs text-dark-300">
            REALITY
          </span>
          <span className="rounded-full bg-dark-800 px-3 py-1 text-xs text-dark-300">
            Hysteria2
          </span>
          <span className="rounded-full bg-dark-800 px-3 py-1 text-xs text-dark-300">
            WebSocket
          </span>
        </div>
      </div>
    </div>
  )
}
