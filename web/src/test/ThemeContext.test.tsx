import React from 'react'
import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { ThemeProvider, useTheme } from '@/contexts/ThemeContext'

// jsdom doesn't implement matchMedia — provide a minimal mock.
function mockMatchMedia(matches: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
  })
}

function wrapper({ children }: { children: React.ReactNode }) {
  return <ThemeProvider>{children}</ThemeProvider>
}

describe('ThemeContext', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.classList.remove('dark')
    mockMatchMedia(false)
  })

  it('throws when used outside provider', () => {
    expect(() => renderHook(() => useTheme())).toThrow(
      'useTheme must be used within ThemeProvider'
    )
  })

  it('defaults to light when system is light and nothing stored', () => {
    mockMatchMedia(false)
    const { result } = renderHook(() => useTheme(), { wrapper })
    expect(result.current.theme).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('defaults to dark when system is dark and nothing stored', () => {
    mockMatchMedia(true)
    const { result } = renderHook(() => useTheme(), { wrapper })
    expect(result.current.theme).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('reads stored dark preference from localStorage over system light', () => {
    localStorage.setItem('aghsync_theme', 'dark')
    mockMatchMedia(false)
    const { result } = renderHook(() => useTheme(), { wrapper })
    expect(result.current.theme).toBe('dark')
  })

  it('reads stored light preference from localStorage over system dark', () => {
    localStorage.setItem('aghsync_theme', 'light')
    mockMatchMedia(true)
    const { result } = renderHook(() => useTheme(), { wrapper })
    expect(result.current.theme).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('toggleTheme switches light→dark, persists to localStorage, applies class', () => {
    mockMatchMedia(false)
    const { result } = renderHook(() => useTheme(), { wrapper })
    expect(result.current.theme).toBe('light')
    act(() => result.current.toggleTheme())
    expect(result.current.theme).toBe('dark')
    expect(localStorage.getItem('aghsync_theme')).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('toggleTheme switches dark→light, persists to localStorage, removes class', () => {
    localStorage.setItem('aghsync_theme', 'dark')
    mockMatchMedia(false)
    const { result } = renderHook(() => useTheme(), { wrapper })
    act(() => result.current.toggleTheme())
    expect(result.current.theme).toBe('light')
    expect(localStorage.getItem('aghsync_theme')).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('ignores invalid localStorage values and falls back to system preference', () => {
    localStorage.setItem('aghsync_theme', 'invalid-value')
    mockMatchMedia(true)
    const { result } = renderHook(() => useTheme(), { wrapper })
    expect(result.current.theme).toBe('dark')
  })
})
