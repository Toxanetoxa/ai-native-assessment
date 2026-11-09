export async function get<T>(
  path: string,
  params: Record<string, string | number> = {},
  signal?: AbortSignal,
): Promise<T> {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== "") query.set(key, String(value));
  }
  const response = await fetch(`/api${path}${query.size ? `?${query}` : ""}`, {
    signal,
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error ?? `Ошибка запроса (${response.status})`);
  }
  return response.json() as Promise<T>;
}
