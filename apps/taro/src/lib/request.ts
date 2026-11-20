import type { ApiEndpoint, Envelope, TokenPair } from '@/api/schema';
import Taro from '@tarojs/taro';

const ACCESS_KEY = 'gs_access_token';
const REFRESH_KEY = 'gs_refresh_token';
const BASE_URL = process.env.TARO_APP_API_BASE_URL ?? 'http://localhost:8080';

export class ApiError extends Error {
  constructor(
    public readonly code: number,
    message: string,
    public readonly statusCode: number,
  ) {
    super(message);
  }
}

export const tokenStorage = {
  getAccess: () => Taro.getStorageSync<string>(ACCESS_KEY) || '',
  setTokens: (pair: TokenPair) => {
    Taro.setStorageSync(ACCESS_KEY, pair.access_token);
    Taro.setStorageSync(REFRESH_KEY, pair.refresh_token);
  },
  clear: () => {
    Taro.removeStorageSync(ACCESS_KEY);
    Taro.removeStorageSync(REFRESH_KEY);
  },
};

export async function apiRequest<T>(
  path: ApiEndpoint,
  options: {
    method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
    data?: object;
  } = {},
): Promise<T> {
  const token = tokenStorage.getAccess();
  const header: Record<string, string> = { 'Content-Type': 'application/json' };
  if (token) {
    header.Authorization = `Bearer ${token}`;
  }

  const res = await Taro.request<Envelope<T>>({
    url: `${BASE_URL}${path}`,
    method: options.method ?? 'GET',
    data: options.data,
    header,
    timeout: 15000,
  });

  if (res.statusCode === 401) {
    tokenStorage.clear();
    void Taro.navigateTo({ url: '/pages/login/index' });
    throw new ApiError(40101, '请先登录', 401);
  }

  if (res.statusCode < 200 || res.statusCode >= 300 || res.data.code !== 0) {
    throw new ApiError(
      res.data?.code ?? res.statusCode,
      res.data?.message ?? '请求失败',
      res.statusCode,
    );
  }
  return res.data.data;
}
