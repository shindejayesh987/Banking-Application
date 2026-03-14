import { createBrowserRouter } from 'react-router-dom'
import { AppLayout } from '@/components/layout/AppLayout'
import { ProtectedRoute } from '@/components/layout/ProtectedRoute'
import { LoginPage } from '@/pages/auth/LoginPage'
import { RegisterPage } from '@/pages/auth/RegisterPage'
import { DashboardPage } from '@/pages/DashboardPage'
import { PlaceholderPage } from '@/pages/PlaceholderPage'

export const router = createBrowserRouter([
  {
    path: '/auth/login',
    element: <LoginPage />,
  },
  {
    path: '/auth/register',
    element: <RegisterPage />,
  },
  {
    path: '/',
    element: (
      <ProtectedRoute>
        <AppLayout />
      </ProtectedRoute>
    ),
    children: [
      { index: true, element: <DashboardPage /> },
      { path: 'accounts', element: <PlaceholderPage /> },
      { path: 'accounts/:id', element: <PlaceholderPage /> },
      { path: 'transactions/new', element: <PlaceholderPage /> },
      { path: 'transactions/:id', element: <PlaceholderPage /> },
      { path: 'system/redis', element: <PlaceholderPage /> },
      { path: 'system/kafka', element: <PlaceholderPage /> },
      { path: 'system/circuit-breaker', element: <PlaceholderPage /> },
      { path: 'system/rate-limiting', element: <PlaceholderPage /> },
      { path: 'system/replication', element: <PlaceholderPage /> },
      { path: 'system/saga', element: <PlaceholderPage /> },
      { path: 'system/audit', element: <PlaceholderPage /> },
    ],
  },
])
