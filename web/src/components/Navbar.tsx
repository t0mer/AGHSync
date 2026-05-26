import { Link } from 'react-router-dom'
import { Moon, Sun } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/contexts/AuthContext'
import { useTheme } from '@/contexts/ThemeContext'

export function Navbar() {
  const { authRequired, logout } = useAuth()
  const { theme, toggleTheme } = useTheme()

  return (
    <header className="border-b bg-card px-6 py-3 flex items-center justify-between shadow-sm">
      <Link to="/" className="font-semibold text-lg tracking-tight">
        AGHSync
      </Link>
      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={toggleTheme}
          aria-label="Toggle theme"
        >
          {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
        </Button>
        {authRequired && (
          <Button variant="ghost" size="sm" onClick={logout}>
            Logout
          </Button>
        )}
      </div>
    </header>
  )
}
