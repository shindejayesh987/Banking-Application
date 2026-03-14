import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  Wallet,
  ArrowLeftRight,
  Database,
  Radio,
  ShieldAlert,
  Gauge,
  Copy,
  GitBranch,
  ScrollText,
  LogOut,
} from 'lucide-react'
import { useAuthStore } from '@/store/auth'
import { cn } from '@/lib/utils'

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/accounts', icon: Wallet, label: 'Accounts' },
  { to: '/transactions/new', icon: ArrowLeftRight, label: 'New Transfer' },
  { heading: 'System Design' },
  { to: '/system/redis', icon: Database, label: 'Redis Cache' },
  { to: '/system/kafka', icon: Radio, label: 'Kafka' },
  { to: '/system/circuit-breaker', icon: ShieldAlert, label: 'Circuit Breaker' },
  { to: '/system/rate-limiting', icon: Gauge, label: 'Rate Limiting' },
  { to: '/system/replication', icon: Copy, label: 'DB Replication' },
  { to: '/system/saga', icon: GitBranch, label: 'Saga' },
  { to: '/system/audit', icon: ScrollText, label: 'Audit Log' },
]

export function Sidebar() {
  const logout = useAuthStore((s) => s.logout)
  const username = useAuthStore((s) => s.username)

  return (
    <aside className="flex h-screen w-64 flex-col border-r bg-card">
      <div className="border-b p-4">
        <h1 className="text-lg font-bold tracking-tight">Banking Lab</h1>
        <p className="text-xs text-muted-foreground">System Design Explorer</p>
      </div>

      <nav className="flex-1 overflow-y-auto p-3 space-y-1">
        {navItems.map((item, i) => {
          if ('heading' in item) {
            return (
              <p key={i} className="mt-4 mb-1 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                {item.heading}
              </p>
            )
          }
          const Icon = item.icon
          return (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                  isActive ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
                )
              }
            >
              <Icon className="h-4 w-4" />
              {item.label}
            </NavLink>
          )
        })}
      </nav>

      <div className="border-t p-3">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium truncate">{username}</span>
          <button
            onClick={() => logout()}
            className="rounded-md p-2 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
          >
            <LogOut className="h-4 w-4" />
          </button>
        </div>
      </div>
    </aside>
  )
}
