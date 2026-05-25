import { http, HttpResponse } from 'msw'

export const handlers = [
  http.get('/api/v1/settings', () => {
    return HttpResponse.json({
      ui_auth_enabled: false,
      ui_username: '',
      has_api_token: false,
      scheduler_cron: '',
      port: 8080,
    })
  }),
  http.get('/api/v1/instances', () => {
    return HttpResponse.json([
      {
        id: 'inst-1',
        name: 'Master',
        address: 'http://192.168.1.1',
        username: 'admin',
        is_master: true,
        tls_skip_verify: false,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
    ])
  }),
  http.get('/api/v1/sync/status', () => {
    return HttpResponse.json({
      current: null,
      last: null,
    })
  }),
  http.get('/api/v1/history', () => {
    return HttpResponse.json([])
  }),
]
