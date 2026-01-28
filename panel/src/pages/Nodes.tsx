import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus,
  Server,
  MoreHorizontal,
  Edit,
  Trash2,
  RefreshCw,
  ChevronDown,
  ChevronRight,
  Wifi,
  WifiOff,
} from 'lucide-react'
import { nodesApi, inboundsApi } from '../api/client'
import { useToast } from '../hooks/useToast'
import Modal from '../components/Modal'
import ConfirmDialog from '../components/ConfirmDialog'
import NodeForm from '../components/NodeForm'
import InboundForm from '../components/InboundForm'
import StatusBadge from '../components/StatusBadge'
import type { Node, Inbound, CreateNodeInput, CreateInboundInput } from '../types'

export default function Nodes() {
  const [isNodeFormOpen, setIsNodeFormOpen] = useState(false)
  const [editingNode, setEditingNode] = useState<Node | null>(null)
  const [deletingNode, setDeletingNode] = useState<Node | null>(null)
  const [expandedNodes, setExpandedNodes] = useState<Set<number>>(new Set())
  const [nodeInbounds, setNodeInbounds] = useState<Record<number, Inbound[]>>({})
  const [nodeStatuses, setNodeStatuses] = useState<Record<number, boolean>>({})

  // Inbound state
  const [isInboundFormOpen, setIsInboundFormOpen] = useState(false)
  const [selectedNodeId, setSelectedNodeId] = useState<number | null>(null)
  const [editingInbound, setEditingInbound] = useState<Inbound | null>(null)
  const [deletingInbound, setDeletingInbound] = useState<Inbound | null>(null)

  const [actionMenuNode, setActionMenuNode] = useState<number | null>(null)
  const [actionMenuInbound, setActionMenuInbound] = useState<number | null>(null)

  const queryClient = useQueryClient()
  const addToast = useToast((state) => state.addToast)

  const { data: nodes, isLoading, error } = useQuery({
    queryKey: ['nodes'],
    queryFn: nodesApi.list,
  })

  // Fetch node statuses
  useEffect(() => {
    if (nodes) {
      nodes.forEach(async (node: Node) => {
        try {
          const status = await nodesApi.getStatus(node.id)
          setNodeStatuses((prev) => ({ ...prev, [node.id]: status.online }))
        } catch {
          setNodeStatuses((prev) => ({ ...prev, [node.id]: false }))
        }
      })
    }
  }, [nodes])

  // Fetch inbounds for expanded nodes
  useEffect(() => {
    expandedNodes.forEach(async (nodeId) => {
      if (!nodeInbounds[nodeId]) {
        const inbounds = await inboundsApi.listByNode(nodeId)
        setNodeInbounds((prev) => ({ ...prev, [nodeId]: inbounds }))
      }
    })
  }, [expandedNodes, nodeInbounds])

  const createNodeMutation = useMutation({
    mutationFn: nodesApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nodes'] })
      setIsNodeFormOpen(false)
      addToast('success', 'Node created successfully')
    },
    onError: (err: Error) => addToast('error', err.message),
  })

  const updateNodeMutation = useMutation({
    mutationFn: (data: CreateNodeInput & { id: number }) => nodesApi.update(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nodes'] })
      setEditingNode(null)
      addToast('success', 'Node updated successfully')
    },
    onError: (err: Error) => addToast('error', err.message),
  })

  const deleteNodeMutation = useMutation({
    mutationFn: nodesApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nodes'] })
      setDeletingNode(null)
      addToast('success', 'Node deleted successfully')
    },
    onError: (err: Error) => addToast('error', err.message),
  })

  const syncNodeMutation = useMutation({
    mutationFn: nodesApi.sync,
    onSuccess: () => {
      addToast('success', 'Node synced successfully')
    },
    onError: (err: Error) => addToast('error', err.message),
  })

  const createInboundMutation = useMutation({
    mutationFn: inboundsApi.create,
    onSuccess: (_, variables) => {
      setNodeInbounds((prev) => ({ ...prev, [variables.node_id]: [] }))
      inboundsApi.listByNode(variables.node_id).then((inbounds) => {
        setNodeInbounds((prev) => ({ ...prev, [variables.node_id]: inbounds }))
      })
      setIsInboundFormOpen(false)
      setSelectedNodeId(null)
      addToast('success', 'Inbound created successfully')
    },
    onError: (err: Error) => addToast('error', err.message),
  })

  const updateInboundMutation = useMutation({
    mutationFn: (data: CreateInboundInput & { id: number }) => inboundsApi.update(data),
    onSuccess: (_, variables) => {
      inboundsApi.listByNode(variables.node_id).then((inbounds) => {
        setNodeInbounds((prev) => ({ ...prev, [variables.node_id]: inbounds }))
      })
      setEditingInbound(null)
      addToast('success', 'Inbound updated successfully')
    },
    onError: (err: Error) => addToast('error', err.message),
  })

  const deleteInboundMutation = useMutation({
    mutationFn: inboundsApi.delete,
    onSuccess: () => {
      // Refresh all inbounds
      expandedNodes.forEach(async (nodeId) => {
        const inbounds = await inboundsApi.listByNode(nodeId)
        setNodeInbounds((prev) => ({ ...prev, [nodeId]: inbounds }))
      })
      setDeletingInbound(null)
      addToast('success', 'Inbound deleted successfully')
    },
    onError: (err: Error) => addToast('error', err.message),
  })

  const toggleNode = (nodeId: number) => {
    setExpandedNodes((prev) => {
      const next = new Set(prev)
      if (next.has(nodeId)) {
        next.delete(nodeId)
      } else {
        next.add(nodeId)
      }
      return next
    })
  }

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
        Failed to load nodes: {error.message}
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Nodes</h1>
          <p className="mt-1 text-dark-400">Manage VPN server nodes</p>
        </div>
        <button onClick={() => setIsNodeFormOpen(true)} className="btn-primary">
          <Plus className="h-4 w-4" />
          Add Node
        </button>
      </div>

      {/* Nodes List */}
      <div className="space-y-4">
        {nodes?.map((node: Node) => (
          <div key={node.id} className="card p-0 overflow-hidden">
            {/* Node Header */}
            <div className="flex items-center gap-4 p-4">
              <button
                onClick={() => toggleNode(node.id)}
                className="flex-shrink-0 rounded-lg p-2 hover:bg-dark-800"
              >
                {expandedNodes.has(node.id) ? (
                  <ChevronDown className="h-5 w-5 text-dark-400" />
                ) : (
                  <ChevronRight className="h-5 w-5 text-dark-400" />
                )}
              </button>

              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-dark-800">
                <Server className="h-6 w-6 text-dark-300" />
              </div>

              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <h3 className="truncate font-medium text-white">{node.name}</h3>
                  <StatusBadge
                    variant={nodeStatuses[node.id] ? 'online' : 'offline'}
                  />
                </div>
                <p className="text-sm text-dark-400">
                  {node.address}:{node.api_port}
                </p>
              </div>

              <div className="relative">
                <button
                  onClick={() =>
                    setActionMenuNode(actionMenuNode === node.id ? null : node.id)
                  }
                  className="btn-ghost btn-sm"
                >
                  <MoreHorizontal className="h-4 w-4" />
                </button>

                {actionMenuNode === node.id && (
                  <>
                    <div
                      className="fixed inset-0 z-10"
                      onClick={() => setActionMenuNode(null)}
                    />
                    <div className="absolute right-0 z-20 mt-2 w-48 rounded-lg border border-dark-700 bg-dark-800 py-1 shadow-lg">
                      <button
                        onClick={() => {
                          syncNodeMutation.mutate(node.id)
                          setActionMenuNode(null)
                        }}
                        className="flex w-full items-center gap-2 px-4 py-2 text-sm text-dark-200 hover:bg-dark-700"
                      >
                        <RefreshCw className="h-4 w-4" />
                        Sync Config
                      </button>
                      <button
                        onClick={() => {
                          setSelectedNodeId(node.id)
                          setIsInboundFormOpen(true)
                          setActionMenuNode(null)
                        }}
                        className="flex w-full items-center gap-2 px-4 py-2 text-sm text-dark-200 hover:bg-dark-700"
                      >
                        <Plus className="h-4 w-4" />
                        Add Inbound
                      </button>
                      <button
                        onClick={() => {
                          setEditingNode(node)
                          setActionMenuNode(null)
                        }}
                        className="flex w-full items-center gap-2 px-4 py-2 text-sm text-dark-200 hover:bg-dark-700"
                      >
                        <Edit className="h-4 w-4" />
                        Edit Node
                      </button>
                      <hr className="my-1 border-dark-700" />
                      <button
                        onClick={() => {
                          setDeletingNode(node)
                          setActionMenuNode(null)
                        }}
                        className="flex w-full items-center gap-2 px-4 py-2 text-sm text-red-400 hover:bg-dark-700"
                      >
                        <Trash2 className="h-4 w-4" />
                        Delete Node
                      </button>
                    </div>
                  </>
                )}
              </div>
            </div>

            {/* Inbounds List */}
            {expandedNodes.has(node.id) && (
              <div className="border-t border-dark-800 bg-dark-800/50">
                <div className="p-4">
                  <div className="mb-3 flex items-center justify-between">
                    <h4 className="text-sm font-medium text-dark-300">Inbounds</h4>
                    <button
                      onClick={() => {
                        setSelectedNodeId(node.id)
                        setIsInboundFormOpen(true)
                      }}
                      className="btn-secondary btn-sm"
                    >
                      <Plus className="h-3 w-3" />
                      Add
                    </button>
                  </div>

                  {nodeInbounds[node.id]?.length ? (
                    <div className="space-y-2">
                      {nodeInbounds[node.id].map((inbound: Inbound) => (
                        <div
                          key={inbound.id}
                          className="flex items-center gap-4 rounded-lg border border-dark-700 bg-dark-900 p-3"
                        >
                          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-dark-800">
                            {inbound.enabled ? (
                              <Wifi className="h-5 w-5 text-green-500" />
                            ) : (
                              <WifiOff className="h-5 w-5 text-dark-500" />
                            )}
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="font-medium text-white">
                                {inbound.name}
                              </span>
                              <span className="rounded bg-dark-700 px-2 py-0.5 text-xs text-dark-300">
                                {inbound.protocol}
                              </span>
                            </div>
                            <p className="text-xs text-dark-400">
                              Port {inbound.listen_port}
                              {inbound.sni && ` / SNI: ${inbound.sni}`}
                            </p>
                          </div>
                          <div className="relative">
                            <button
                              onClick={() =>
                                setActionMenuInbound(
                                  actionMenuInbound === inbound.id
                                    ? null
                                    : inbound.id
                                )
                              }
                              className="btn-ghost btn-sm"
                            >
                              <MoreHorizontal className="h-4 w-4" />
                            </button>

                            {actionMenuInbound === inbound.id && (
                              <>
                                <div
                                  className="fixed inset-0 z-10"
                                  onClick={() => setActionMenuInbound(null)}
                                />
                                <div className="absolute right-0 z-20 mt-2 w-40 rounded-lg border border-dark-700 bg-dark-800 py-1 shadow-lg">
                                  <button
                                    onClick={() => {
                                      setEditingInbound(inbound)
                                      setSelectedNodeId(node.id)
                                      setActionMenuInbound(null)
                                    }}
                                    className="flex w-full items-center gap-2 px-4 py-2 text-sm text-dark-200 hover:bg-dark-700"
                                  >
                                    <Edit className="h-4 w-4" />
                                    Edit
                                  </button>
                                  <button
                                    onClick={() => {
                                      setDeletingInbound(inbound)
                                      setActionMenuInbound(null)
                                    }}
                                    className="flex w-full items-center gap-2 px-4 py-2 text-sm text-red-400 hover:bg-dark-700"
                                  >
                                    <Trash2 className="h-4 w-4" />
                                    Delete
                                  </button>
                                </div>
                              </>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="py-4 text-center text-sm text-dark-400">
                      No inbounds configured
                    </p>
                  )}
                </div>
              </div>
            )}
          </div>
        ))}

        {nodes?.length === 0 && (
          <div className="card py-12 text-center">
            <Server className="mx-auto h-12 w-12 text-dark-500" />
            <h3 className="mt-4 text-lg font-medium text-white">No nodes yet</h3>
            <p className="mt-2 text-dark-400">
              Add your first VPN node to get started
            </p>
            <button
              onClick={() => setIsNodeFormOpen(true)}
              className="btn-primary mt-4"
            >
              <Plus className="h-4 w-4" />
              Add Node
            </button>
          </div>
        )}
      </div>

      {/* Node Form Modal */}
      <Modal
        isOpen={isNodeFormOpen || !!editingNode}
        onClose={() => {
          setIsNodeFormOpen(false)
          setEditingNode(null)
        }}
        title={editingNode ? 'Edit Node' : 'Create Node'}
      >
        <NodeForm
          node={editingNode}
          onSubmit={(data) => {
            if (editingNode) {
              updateNodeMutation.mutate({ ...data, id: editingNode.id })
            } else {
              createNodeMutation.mutate(data)
            }
          }}
          onCancel={() => {
            setIsNodeFormOpen(false)
            setEditingNode(null)
          }}
          isLoading={createNodeMutation.isPending || updateNodeMutation.isPending}
        />
      </Modal>

      {/* Delete Node Confirmation */}
      <ConfirmDialog
        isOpen={!!deletingNode}
        onClose={() => setDeletingNode(null)}
        onConfirm={() => deletingNode && deleteNodeMutation.mutate(deletingNode.id)}
        title="Delete Node"
        message={`Are you sure you want to delete "${deletingNode?.name}"? This will also delete all inbounds on this node.`}
        confirmText="Delete"
        isLoading={deleteNodeMutation.isPending}
      />

      {/* Inbound Form Modal */}
      <Modal
        isOpen={isInboundFormOpen || !!editingInbound}
        onClose={() => {
          setIsInboundFormOpen(false)
          setEditingInbound(null)
          setSelectedNodeId(null)
        }}
        title={editingInbound ? 'Edit Inbound' : 'Create Inbound'}
        size="lg"
      >
        {(selectedNodeId || editingInbound) && (
          <InboundForm
            inbound={editingInbound}
            nodeId={selectedNodeId || editingInbound?.node_id || 0}
            onSubmit={(data) => {
              if (editingInbound) {
                updateInboundMutation.mutate({ ...data, id: editingInbound.id })
              } else {
                createInboundMutation.mutate(data)
              }
            }}
            onCancel={() => {
              setIsInboundFormOpen(false)
              setEditingInbound(null)
              setSelectedNodeId(null)
            }}
            isLoading={
              createInboundMutation.isPending || updateInboundMutation.isPending
            }
          />
        )}
      </Modal>

      {/* Delete Inbound Confirmation */}
      <ConfirmDialog
        isOpen={!!deletingInbound}
        onClose={() => setDeletingInbound(null)}
        onConfirm={() =>
          deletingInbound && deleteInboundMutation.mutate(deletingInbound.id)
        }
        title="Delete Inbound"
        message={`Are you sure you want to delete "${deletingInbound?.name}"? Users won't be able to connect through this inbound anymore.`}
        confirmText="Delete"
        isLoading={deleteInboundMutation.isPending}
      />
    </div>
  )
}
