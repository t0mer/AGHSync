import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/contexts/AuthContext'

export function Navbar() {
  const { authRequired, logout } = useAuth()
  return (
    <header className="border-b bg-card px-6 py-3 flex items-center justify-between">
      <Link to="/" className="font-semibold text-lg tracking-tight">
        AGHSync
      </Link>
      {authRequired && (
        <Button variant="ghost" size="sm" onClick={logout}>
          Logout
        </Button>
      )}
    </header>
  )
}
