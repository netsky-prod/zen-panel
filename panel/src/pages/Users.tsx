import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus,
  Search,
  MoreHorizontal,
  Edit,
  Trash2,
  RefreshCw,
  Key,
  QrCode,
  Power,
  PowerOff,
} from 'lucide-react'
import { usersApi } from '../api/client'
import { useToast } from '../hooks/useToast'
import Modal from '../components/Modal'
import ConfirmDialog from '../components/ConfirmDialog'
import UserForm from '../components/UserForm'
import ConfigModal from '../components/ConfigModal'
import StatusBadge from '../components/StatusBadge'
import Dropdown, { DropdownItem, DropdownDivider } from '../components/Dropdown'
import type { User, CreateUserInput, UserConfig } from '../types'

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatDate(dateStr: string | null): string {
  if (!dateStr) return 'Never'
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function isExpired(dateStr: string | null): boolean {
  if (!dateStr) return false
  return new Date(dateStr) < new Date()
}

export default function Users() {
  const [search, setSearch] = useState('')
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [deletingUser, setDeletingUser] = useState<User | null>(null)
  const [configUser, setConfigUser] = useState<User | null>(null)
  const [userConfig, setUserConfig] = useState<UserConfig | null>(null)

  const queryClient = useQueryClient()
  const addToast = useToast((state) => state.addToast)

  const { data: users, isLoading, error } = useQuery({
    queryKey: ['users'],
    queryFn: usersApi.list,
  })

  const createMutation = useMutation({
    mutationFn: usersApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      setIsFormOpen(false)
      addToast('success', 'User created successfully')
    },
    onError: (err: Error) => addToast('error', err.message),
  })

  const updateMutation = useMutation({
    mutationFn: (data: CreateUserInput & { id: number }) => usersApi.update(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      setEditingUser(null)
      addToast('success', 'User updated successfully')
    },
    onError: (err: Error) => addToast('error', err.message),
  })

  const deleteMutation = useMutation({
    mutationFn: usersApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      setDeletingUser(null)
      addToast('success', 'User deleted successfully')
    },
    onError: (err: Error) => addToast('error', err.message),
  })

  const enableMutation = useMutation({
    mutationFn: usersApi.enable,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      addToast('success', 'User enabled')
    },
    onError: (err: Error) => addToast('error', err.message),
  })

  const disableMutation = useMutation({
    mutationFn: usersApi.disable,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      addToast('success', 'User disabled')
    },
    onError: (err: Error) => addToast('error', err.message),
  })

  const resetUUIDMutation = useMutation({
    mutationFn: usersApi.resetUUID,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      addToast('success', 'UUID reset successfully')
    },
    onError: (err: Error) => addToast('error', err.message),
  })

  const resetTrafficMutation = useMutation({
    mutationFn: usersApi.resetTraffic,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      addToast('success', 'Traffic reset successfully')
    },
    onError: (err: Error) => addToast('error', err.message),
  })

  const handleGetConfig = async (user: User) => {
    try {
      const config = await usersApi.getConfig(user.id)
      setUserConfig(config)
      setConfigUser(user)
    } catch (err) {
      addToast('error', err instanceof Error ? err.message : 'Failed to get config')
    }
  }

  const filteredUsers = users?.filter((user) =>
    user.name.toLowerCase().includes(search.toLowerCase())
  )

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-blue-600 border-t-transparent" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="rounded-lg bg-red-900/50 border border-red-700 p-4 text-red-200">
        Failed to load users: {error.message}
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Users</h1>
          <p className="mt-1 text-dark-400">Manage users and access</p>
        </div>
        <button onClick={() => setIsFormOpen(true)} className="btn-primary">
          <Plus className="h-4 w-4" />
          Add User
        </button>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-dark-400" />
        <input
          type="text"
          placeholder="Search users..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="input pl-10"
        />
      </div>

      {/* Users Table */}
      <div className="table-container">
        <table className="table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Status</th>
              <th>Traffic</th>
              <th>Expires</th>
              <th className="w-20">Actions</th>
            </tr>
          </thead>
          <tbody>
            {filteredUsers?.map((user) => (
              <tr key={user.id}>
                <td>
                  <div className="font-medium text-white">{user.name}</div>
                  <div className="text-xs text-dark-400 font-mono">
                    {user.uuid.slice(0, 8)}...
                  </div>
                </td>
                <td>
                  {!user.enabled ? (
                    <StatusBadge variant="disabled" />
                  ) : isExpired(user.expires_at) ? (
                    <StatusBadge variant="expired" />
                  ) : (
                    <StatusBadge variant="enabled" />
                  )}
                </td>
                <td>
                  <div className="text-white">{formatBytes(user.data_used)}</div>
                  <div className="text-xs text-dark-400">
                    / {user.data_limit > 0 ? formatBytes(user.data_limit) : 'Unlimited'}
                  </div>
                  {user.data_limit > 0 && (
                    <div className="mt-1 h-1.5 w-24 overflow-hidden rounded-full bg-dark-700">
                      <div
                        className="h-full bg-blue-600"
                        style={{
                          width: `${Math.min(100, (user.data_used / user.data_limit) * 100)}%`,
                        }}
                      />
                    </div>
                  )}
                </td>
                <td className="text-dark-300">{formatDate(user.expires_at)}</td>
                <td>
                  <Dropdown
                    trigger={
                      <button className="btn-ghost btn-sm">
                        <MoreHorizontal className="h-4 w-4" />
                      </button>
                    }
                  >
                    <DropdownItem onClick={() => handleGetConfig(user)}>
                      <QrCode className="h-4 w-4" />
                      Get Config
                    </DropdownItem>
                    <DropdownItem onClick={() => setEditingUser(user)}>
                      <Edit className="h-4 w-4" />
                      Edit
                    </DropdownItem>
                    <DropdownItem
                      onClick={() => {
                        if (user.enabled) {
                          disableMutation.mutate(user.id)
                        } else {
                          enableMutation.mutate(user.id)
                        }
                      }}
                    >
                      {user.enabled ? (
                        <>
                          <PowerOff className="h-4 w-4" />
                          Disable
                        </>
                      ) : (
                        <>
                          <Power className="h-4 w-4" />
                          Enable
                        </>
                      )}
                    </DropdownItem>
                    <DropdownItem onClick={() => resetUUIDMutation.mutate(user.id)}>
                      <Key className="h-4 w-4" />
                      Reset UUID
                    </DropdownItem>
                    <DropdownItem onClick={() => resetTrafficMutation.mutate(user.id)}>
                      <RefreshCw className="h-4 w-4" />
                      Reset Traffic
                    </DropdownItem>
                    <DropdownDivider />
                    <DropdownItem variant="danger" onClick={() => setDeletingUser(user)}>
                      <Trash2 className="h-4 w-4" />
                      Delete
                    </DropdownItem>
                  </Dropdown>
                </td>
              </tr>
            ))}
            {filteredUsers?.length === 0 && (
              <tr>
                <td colSpan={5} className="text-center py-8 text-dark-400">
                  No users found
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Create/Edit Modal */}
      <Modal
        isOpen={isFormOpen || !!editingUser}
        onClose={() => {
          setIsFormOpen(false)
          setEditingUser(null)
        }}
        title={editingUser ? 'Edit User' : 'Create User'}
        size="lg"
      >
        <UserForm
          user={editingUser}
          onSubmit={(data) => {
            if (editingUser) {
              updateMutation.mutate({ ...data, id: editingUser.id })
            } else {
              createMutation.mutate(data)
            }
          }}
          onCancel={() => {
            setIsFormOpen(false)
            setEditingUser(null)
          }}
          isLoading={createMutation.isPending || updateMutation.isPending}
        />
      </Modal>

      {/* Delete Confirmation */}
      <ConfirmDialog
        isOpen={!!deletingUser}
        onClose={() => setDeletingUser(null)}
        onConfirm={() => deletingUser && deleteMutation.mutate(deletingUser.id)}
        title="Delete User"
        message={`Are you sure you want to delete "${deletingUser?.name}"? This action cannot be undone.`}
        confirmText="Delete"
        isLoading={deleteMutation.isPending}
      />

      {/* Config Modal */}
      <ConfigModal
        isOpen={!!configUser}
        onClose={() => {
          setConfigUser(null)
          setUserConfig(null)
        }}
        config={userConfig}
        userName={configUser?.name || ''}
        userUUID={configUser?.uuid}
      />
    </div>
  )
}
