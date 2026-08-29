export function githubAuthUrl(): string {
  const apiBase = (import.meta.env.VITE_API_BASE_URL || '/api/v1').replace(/\/$/, '')
  return `${apiBase}/auth/github`
}

export const githubOAuthEnabled = import.meta.env.VITE_GITHUB_OAUTH_ENABLED === 'true'
