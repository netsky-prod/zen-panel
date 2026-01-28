import { clsx } from 'clsx'

type BadgeVariant = 'online' | 'offline' | 'enabled' | 'disabled' | 'expired' | 'active'

interface StatusBadgeProps {
  variant: BadgeVariant
  children?: React.ReactNode
}

const variants: Record<BadgeVariant, string> = {
  online: 'bg-green-900/50 text-green-400 border-green-700',
  offline: 'bg-red-900/50 text-red-400 border-red-700',
  enabled: 'bg-green-900/50 text-green-400 border-green-700',
  disabled: 'bg-dark-700 text-dark-400 border-dark-600',
  expired: 'bg-yellow-900/50 text-yellow-400 border-yellow-700',
  active: 'bg-blue-900/50 text-blue-400 border-blue-700',
}

const labels: Record<BadgeVariant, string> = {
  online: 'Online',
  offline: 'Offline',
  enabled: 'Enabled',
  disabled: 'Disabled',
  expired: 'Expired',
  active: 'Active',
}

export default function StatusBadge({ variant, children }: StatusBadgeProps) {
  return (
    <span
      className={clsx(
        'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium',
        variants[variant]
      )}
    >
      <span
        className={clsx(
          'h-1.5 w-1.5 rounded-full',
          variant === 'online' || variant === 'enabled' || variant === 'active'
            ? 'bg-green-400'
            : variant === 'offline'
            ? 'bg-red-400'
            : variant === 'expired'
            ? 'bg-yellow-400'
            : 'bg-dark-400'
        )}
      />
      {children || labels[variant]}
    </span>
  )
}
