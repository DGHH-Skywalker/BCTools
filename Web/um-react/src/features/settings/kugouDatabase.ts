export interface KugouDatabaseStatus {
  canSelect: boolean;
  custom: boolean;
  keyCount: number;
}

export function hasLocalDatabaseApi() {
  const host = location.hostname;
  return (
    !!window.BASE_URL &&
    window.BASE_URL !== '/' &&
    (['localhost', '127.0.0.1', '[::1]', 'wails.localhost'].includes(host) || host.startsWith('wails.localhost.'))
  );
}

export async function kugouDatabaseRequest<T>(path: string, method = 'GET', body?: string, signal?: AbortSignal) {
  const response = await fetch(`/api/v1/decrypt/kugou/${path}`, {
    method,
    cache: 'no-store',
    headers: { 'X-BCTools-Local': '1', ...(body ? { 'Content-Type': 'application/json' } : {}) },
    body,
    signal,
  });
  if (response.status === 204) return undefined;
  if (!response.ok) {
    const result = await response.json().catch(() => undefined);
    throw new Error(result?.error?.message ?? '无法访问酷狗数据库');
  }
  return (await response.json()) as T;
}

export async function findAutomaticKugouKey(audioHash: string) {
  if (!hasLocalDatabaseApi()) return undefined;
  try {
    const result = await kugouDatabaseRequest<{ key: string }>(
      'key',
      'POST',
      JSON.stringify({ audioHash }),
      AbortSignal.timeout(5000),
    );
    return result?.key;
  } catch {
    return undefined;
  }
}
