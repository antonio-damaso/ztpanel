import axios from 'axios'

// In production the Go binary serves both the API and the built SPA
// from the same origin, so a relative base URL is all that's needed.
// In dev, Vite's proxy (see vite.config.ts) forwards /api to the Go
// server running separately.
export const api = axios.create({
  baseURL: '/api',
  withCredentials: true,
})

export interface ApiErrorBody {
  error: string
}

export function errorMessage(err: unknown, fallback = 'Something went wrong'): string {
  if (axios.isAxiosError(err)) {
    const body = err.response?.data as ApiErrorBody | undefined
    if (body?.error) return body.error
  }
  return fallback
}
