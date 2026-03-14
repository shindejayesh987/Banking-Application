import { useLocation } from 'react-router-dom'

export function PlaceholderPage() {
  const location = useLocation()
  const title = location.pathname
    .split('/')
    .filter(Boolean)
    .map((s) => s.charAt(0).toUpperCase() + s.slice(1))
    .join(' > ')

  return (
    <div>
      <h1 className="text-3xl font-bold tracking-tight">{title || 'Page'}</h1>
      <p className="text-muted-foreground mt-2">This page will be built in a future phase.</p>
    </div>
  )
}
