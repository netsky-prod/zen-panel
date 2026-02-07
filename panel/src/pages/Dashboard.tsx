import { useQuery } from '@tanstack/react-query'
import { Users, Activity, Download, Upload, Server } from 'lucide-react'
import { dashboardApi } from '../api/client'
import StatusBadge from '../components/StatusBadge'
import StatsChart from '../components/StatsChart'

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

interface StatCardProps {
  title: string
  value: string | number
  icon: React.ReactNode
  subtitle?: string
}

function StatCard({ title, value, icon, subtitle }: StatCardProps) {
  return (
    <div className="card">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-sm font-medium text-dark-400">{title}</p>
          <p className="mt-2 text-3xl font-bold text-white">{value}</p>
          {subtitle && (
            <p className="mt-1 text-sm text-dark-400">{subtitle}</p>
          )}
        </div>
        <div className="rounded-lg bg-dark-800 p-3">{icon}</div>
      </div>
    </div>
  )
}

export default function Dashboard() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['dashboard'],
    queryFn: dashboardApi.get,
    refetchInterval: 30000,
  })

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
        Failed to load dashboard data: {error.message}
      </div>
    )
  }

  if (!data) return null

  const { users, traffic, node_status } = data

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Dashboard</h1>
        <p className="mt-1 text-dark-400">Overview of your infrastructure</p>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Total Users"
          value={users.total}
          icon={<Users className="h-6 w-6 text-blue-500" />}
        />
        <StatCard
          title="Active Users"
          value={users.active}
          icon={<Activity className="h-6 w-6 text-green-500" />}
          subtitle="Enabled & not expired"
        />
        <StatCard
          title="Upload"
          value={formatBytes(traffic.total_upload)}
          icon={<Upload className="h-6 w-6 text-purple-500" />}
        />
        <StatCard
          title="Download"
          value={formatBytes(traffic.total_download)}
          icon={<Download className="h-6 w-6 text-cyan-500" />}
        />
      </div>

      {/* Nodes Status */}
      <div className="card">
        <h2 className="mb-4 text-lg font-semibold text-white">Nodes Status</h2>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {node_status.map((node) => (
            <div
              key={node.id}
              className="flex items-center gap-4 rounded-lg border border-dark-700 bg-dark-800 p-4"
            >
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-dark-700">
                <Server className="h-6 w-6 text-dark-300" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="truncate font-medium text-white">{node.name}</p>
                  <StatusBadge variant={node.online ? 'online' : 'offline'} />
                </div>
                <p className="mt-1 text-sm text-dark-400">{node.address}</p>
                <p className="text-xs text-dark-500">
                  {node.user_count} users / {node.inbound_count} services
                </p>
              </div>
            </div>
          ))}
          {node_status.length === 0 && (
            <div className="col-span-full py-8 text-center text-dark-400">
              No nodes configured yet
            </div>
          )}
        </div>
      </div>

      {/* Traffic Chart */}
      <div className="card">
        <h2 className="mb-4 text-lg font-semibold text-white">
          Traffic (Last 7 Days)
        </h2>
        {traffic.weekly_history && traffic.weekly_history.length > 0 ? (
          <>
            <div className="mb-4 flex items-center gap-6">
              <div className="flex items-center gap-2">
                <div className="h-3 w-3 rounded-full bg-blue-500" />
                <span className="text-sm text-dark-400">Upload</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="h-3 w-3 rounded-full bg-green-500" />
                <span className="text-sm text-dark-400">Download</span>
              </div>
            </div>
            <StatsChart data={traffic.weekly_history} />
          </>
        ) : (
          <div className="flex h-64 items-center justify-center text-dark-400">
            No traffic data available
          </div>
        )}
      </div>
    </div>
  )
}
