import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-react'
import { useToast, Toast as ToastType } from '../hooks/useToast'
import { clsx } from 'clsx'

const icons = {
  success: CheckCircle,
  error: AlertCircle,
  info: Info,
  warning: AlertTriangle,
}

const colors = {
  success: 'bg-green-900/80 border-green-700 text-green-100',
  error: 'bg-red-900/80 border-red-700 text-red-100',
  info: 'bg-blue-900/80 border-blue-700 text-blue-100',
  warning: 'bg-yellow-900/80 border-yellow-700 text-yellow-100',
}

function ToastItem({ toast }: { toast: ToastType }) {
  const removeToast = useToast((state) => state.removeToast)
  const Icon = icons[toast.type]

  return (
    <div
      className={clsx(
        'flex items-center gap-3 rounded-lg border px-4 py-3 shadow-lg toast-enter',
        colors[toast.type]
      )}
    >
      <Icon className="h-5 w-5 flex-shrink-0" />
      <p className="flex-1 text-sm">{toast.message}</p>
      <button
        onClick={() => removeToast(toast.id)}
        className="flex-shrink-0 rounded p-1 hover:bg-white/10"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  )
}

export function ToastContainer() {
  const toasts = useToast((state) => state.toasts)

  if (toasts.length === 0) return null

  return (
    <div className="fixed bottom-4 right-4 z-[100] flex flex-col gap-2">
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} />
      ))}
    </div>
  )
}

export default ToastContainer
