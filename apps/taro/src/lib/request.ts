import type {
  Envelope,
  JsonBodyOf,
  PathParamsOf,
  PathsWithMethod,
  QueryParamsOf,
  ResponseDataOf,
  TokenPair,
} from '@/api/schema';
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

function buildUrl(
  template: string,
  params?: Record<string, string | number>,
  query?: Record<string, string | number | boolean | undefined>,
): string {
  let resolved = template;
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      resolved = resolved.replace(`{${k}}`, encodeURIComponent(String(v)));
    }
  }
  if (query) {
    const pairs: string[] = [];
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined) {
        pairs.push(`${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`);
      }
    }
    if (pairs.length > 0) {
      resolved = `${resolved}?${pairs.join('&')}`;
    }
  }
  return `${BASE_URL}${resolved}`;
}

async function sendTaroEnvelope<T>(
  url: string,
  method: 'GET' | 'POST' | 'PUT' | 'DELETE',
  data?: object,
): Promise<T> {
  const token = tokenStorage.getAccess();
  const header: Record<string, string> = { 'Content-Type': 'application/json' };
  if (token) {
    header.Authorization = `Bearer ${token}`;
  }

  const res = await Taro.request<Envelope<T> | string>({
    url,
    method,
    data,
    header,
    timeout: 15000,
  });

  if (res.statusCode === 401) {
    tokenStorage.clear();
    void Taro.navigateTo({ url: '/pages/login/index' });
    throw new ApiError(40101, '请先登录', 401);
  }

  const payload =
    typeof res.data === 'object' && res.data !== null
      ? (res.data as Partial<Envelope<T>>)
      : null;

  if (
    res.statusCode < 200 ||
    res.statusCode >= 300 ||
    !payload ||
    payload.code !== 0
  ) {
    throw new ApiError(
      payload?.code ?? res.statusCode,
      payload?.message ?? `请求失败 (HTTP ${res.statusCode})`,
      res.statusCode,
    );
  }
  return payload.data as T;
}

type GetArgs<P extends PathsWithMethod<'get'>> = PathParamsOf<
  P,
  'get'
> extends undefined
  ? [options?: { query?: QueryParamsOf<P, 'get'> }]
  : [
      options: {
        params: PathParamsOf<P, 'get'>;
        query?: QueryParamsOf<P, 'get'>;
      },
    ];

type PostArgs<P extends PathsWithMethod<'post'>> = PathParamsOf<
  P,
  'post'
> extends undefined
  ? JsonBodyOf<P, 'post'> extends undefined
    ? []
    : [body: JsonBodyOf<P, 'post'>]
  : JsonBodyOf<P, 'post'> extends undefined
    ? [options: { params: PathParamsOf<P, 'post'> }]
    : [
        body: JsonBodyOf<P, 'post'>,
        options: { params: PathParamsOf<P, 'post'> },
      ];

export const api = {
  get<P extends PathsWithMethod<'get'>>(
    path: P,
    ...args: GetArgs<P>
  ): Promise<ResponseDataOf<P, 'get'>> {
    const opts = args[0] as
      | {
          params?: Record<string, string | number>;
          query?: Record<string, string | number | boolean | undefined>;
        }
      | undefined;
    return sendTaroEnvelope(buildUrl(path, opts?.params, opts?.query), 'GET');
  },

  post<P extends PathsWithMethod<'post'>>(
    path: P,
    ...args: PostArgs<P>
  ): Promise<ResponseDataOf<P, 'post'>> {
    const body = args[0] as object | undefined;
    const opts = args[1] as
      | { params?: Record<string, string | number> }
      | undefined;
    return sendTaroEnvelope(buildUrl(path, opts?.params), 'POST', body);
  },
};
