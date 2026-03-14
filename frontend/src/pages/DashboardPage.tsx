export function DashboardPage() {
  return (
    <div>
      <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
      <p className="text-muted-foreground mt-2">System health at a glance</p>
      <div className="mt-6 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {['Redis Cache', 'Kafka', 'Circuit Breaker', 'Rate Limiter', 'DB Replication', 'Saga'].map((name) => (
          <div key={name} className="rounded-lg border bg-card p-6 text-card-foreground shadow-sm">
            <h3 className="font-semibold">{name}</h3>
            <p className="mt-1 text-sm text-muted-foreground">Waiting for backend connection...</p>
          </div>
        ))}
      </div>
    </div>
  )
}
