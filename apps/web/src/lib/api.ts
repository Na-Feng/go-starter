import type { Envelope, TokenPair } from '@/api/schema';

const ACCESS_KEY = 'gs_access_token';
const REFRESH_KEY = 'gs_refresh_token';

export class ApiError extends Error {
  constructor(
    public readonly code: number,
    message: string,
    public readonly status: number,
  ) {
    super(message);
  }
}

export const tokenStore = {
  getAccess: () => localStorage.getItem(ACCESS_KEY),
  getRefresh: () => localStorage.getItem(REFRESH_KEY),
  setTokens: (pair: TokenPair) => {
    localStorage.setItem(ACCESS_KEY, pair.access_token);
    localStorage.setItem(REFRESH_KEY, pair.refresh_token);
  },
  clear: () => {
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
  },
};

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '';

export async function apiRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  const token = tokenStore.getAccess();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  if (!(init.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const res = await fetch(`${BASE_URL}${path}`, { ...init, headers });
  const json = (await res.json()) as Envelope<T>;

  if (!res.ok || json.code !== 0) {
    if (res.status === 401) {
      tokenStore.clear();
    }
    throw new ApiError(json.code ?? res.status, json.message ?? '请求失败', res.status);
  }
  return json.data;
}
