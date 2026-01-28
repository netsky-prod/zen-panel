import { useState, useRef, useEffect, ReactNode } from 'react'
import { createPortal } from 'react-dom'

interface DropdownProps {
  trigger: ReactNode
  children: ReactNode
  align?: 'left' | 'right'
}

export default function Dropdown({ trigger, children, align = 'right' }: DropdownProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [position, setPosition] = useState({ top: 0, left: 0 })
  const triggerRef = useRef<HTMLDivElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (isOpen && triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect()
      const dropdownWidth = 192 // w-48 = 12rem = 192px

      let left = align === 'right'
        ? rect.right - dropdownWidth
        : rect.left

      // Ensure dropdown doesn't go off-screen
      if (left < 8) left = 8
      if (left + dropdownWidth > window.innerWidth - 8) {
        left = window.innerWidth - dropdownWidth - 8
      }

      setPosition({
        top: rect.bottom + 4,
        left,
      })
    }
  }, [isOpen, align])

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node) &&
        triggerRef.current &&
        !triggerRef.current.contains(e.target as Node)
      ) {
        setIsOpen(false)
      }
    }

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setIsOpen(false)
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      document.addEventListener('keydown', handleEscape)
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [isOpen])

  return (
    <>
      <div ref={triggerRef} onClick={() => setIsOpen(!isOpen)}>
        {trigger}
      </div>

      {isOpen &&
        createPortal(
          <div
            ref={dropdownRef}
            className="fixed z-50 w-48 rounded-lg border border-dark-700 bg-dark-800 py-1 shadow-xl"
            style={{ top: position.top, left: position.left }}
          >
            <div onClick={() => setIsOpen(false)}>{children}</div>
          </div>,
          document.body
        )}
    </>
  )
}

interface DropdownItemProps {
  onClick?: () => void
  children: ReactNode
  variant?: 'default' | 'danger'
}

export function DropdownItem({ onClick, children, variant = 'default' }: DropdownItemProps) {
  return (
    <button
      onClick={onClick}
      className={`flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-dark-700 ${
        variant === 'danger' ? 'text-red-400' : 'text-dark-200'
      }`}
    >
      {children}
    </button>
  )
}

export function DropdownDivider() {
  return <hr className="my-1 border-dark-700" />
}
