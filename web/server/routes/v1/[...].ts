export default defineEventHandler((event) => {
  const config = useRuntimeConfig()
  const apiBase = (config.apiBase || 'http://localhost:8080') as string
  const demoApiKey = (config.demoPartnerApiKey || '') as string

  const incoming = getRequestURL(event)
  const target = new URL(apiBase)

  const reqHeaders: Record<string, string> = {}
  if (demoApiKey) {
    reqHeaders.authorization = `Bearer ${demoApiKey}`
    event.node.req.headers['authorization'] = `Bearer ${demoApiKey}`
  }

  // Forward the original method, headers, body, path and query string to the
  // Go API origin. The browser stays on the same origin, which eliminates CORS.
  return proxyRequest(event, `${target.origin}${incoming.pathname}${incoming.search}`, {
    headers: reqHeaders,
  })
})
