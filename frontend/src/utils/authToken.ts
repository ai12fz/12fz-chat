// 将外部注入的 token(go.12fz.com ChatPanel 传来的 marketplace token)
// 通过 /api/sso/exchange 换成 chat 本地 JWT。设备 token(d_ 前缀)直通。
export async function resolveChatToken(raw: string): Promise<string> {
  if (!raw) return ''
  if (raw.startsWith('d_')) return raw // device token passthrough
  try {
    const resp = await fetch('/api/sso/exchange', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: raw }),
    })
    const data = await resp.json()
    if (resp.ok && data.token) return data.token
  } catch { /* fall through */ }
  return raw // 保留原 token,由 whoami 401 流程兜底
}
