import type { ReactNode } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { List, Plus, Activity, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'
import { api } from '@/lib/api'
import { useState } from 'react'

const navItems = [
  { to: '/', label: 'Scenarios', icon: List },
  { to: '/new', label: 'New Scenario', icon: Plus },
  { to: '/trace', label: 'Trace', icon: Activity },
]

export function Layout({ children }: { children: ReactNode }) {
  const location = useLocation()
  const [reloading, setReloading] = useState(false)

  const handleReload = async () => {
    setReloading(true)
    try {
      await api.reload()
    } catch {
      // ignore
    } finally {
      setReloading(false)
    }
  }

  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b border-[hsl(var(--border))] bg-white px-6 py-3">
        <div className="flex items-center justify-between max-w-7xl mx-auto">
          <div className="flex items-center gap-6">
            <Link to="/" className="text-lg font-semibold text-[hsl(var(--foreground))] no-underline">
              ProteusMock
            </Link>
            <nav className="flex gap-1">
              {navItems.map(item => {
                const Icon = item.icon
                const isActive =
                  item.to === '/'
                    ? location.pathname === '/'
                    : location.pathname.startsWith(item.to)
                return (
                  <Link
                    key={item.to}
                    to={item.to}
                    className={cn(
                      'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm no-underline transition-colors',
                      isActive
                        ? 'bg-[hsl(var(--primary))] text-[hsl(var(--primary-foreground))]'
                        : 'text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--accent))] hover:text-[hsl(var(--accent-foreground))]',
                    )}
                  >
                    <Icon size={16} />
                    {item.label}
                  </Link>
                )
              })}
            </nav>
          </div>
          <button
            onClick={handleReload}
            disabled={reloading}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm border border-[hsl(var(--border))] cursor-pointer',
              'bg-white text-[hsl(var(--foreground))] hover:bg-[hsl(var(--accent))] transition-colors',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
          >
            <RefreshCw size={14} className={reloading ? 'animate-spin' : ''} />
            Reload
          </button>
        </div>
      </header>
      <main className="flex-1 max-w-7xl mx-auto w-full px-6 py-6">{children}</main>
    </div>
  )
}
