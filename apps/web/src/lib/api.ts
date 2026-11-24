import type {
  Envelope,
  FileInfo,
  JsonBodyOf,
  PathParamsOf,
  PathsWithMethod,
  QueryParamsOf,
  ResponseDataOf,
  TokenPair,
} from '@/api/schema';

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
    const usp = new URLSearchParams();
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined) {
        usp.set(k, String(v));
      }
    }
    const qs = usp.toString();
    if (qs) {
      resolved = `${resolved}?${qs}`;
    }
  }
  return `${BASE_URL}${resolved}`;
}

async function sendEnvelope<T>(url: string, init: RequestInit): Promise<T> {
  const headers = new Headers(init.headers);
  const token = tokenStore.getAccess();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  if (
    init.body !== undefined &&
    !(init.body instanceof FormData) &&
    !headers.has('Content-Type')
  ) {
    headers.set('Content-Type', 'application/json');
  }

  const res = await fetch(url, { ...init, headers });
  const rawText = await res.text();

  let json: Partial<Envelope<T>> | null = null;
  if (rawText) {
    try {
      json = JSON.parse(rawText) as Partial<Envelope<T>>;
    } catch {
      if (res.status === 401) {
        tokenStore.clear();
      }
      throw new ApiError(
        res.status,
        `服务响应格式异常 (HTTP ${res.status})`,
        res.status,
      );
    }
  }

  if (!res.ok || !json || json.code !== 0) {
    if (res.status === 401) {
      tokenStore.clear();
    }
    throw new ApiError(
      json?.code ?? res.status,
      json?.message ?? `请求失败 (HTTP ${res.status})`,
      res.status,
    );
  }
  return json.data as T;
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

type MutateArgs<
  M extends 'post' | 'put' | 'patch',
  P extends PathsWithMethod<M>,
> = PathParamsOf<P, M> extends undefined
  ? JsonBodyOf<P, M> extends undefined
    ? []
    : [body: JsonBodyOf<P, M>]
  : JsonBodyOf<P, M> extends undefined
    ? [options: { params: PathParamsOf<P, M> }]
    : [body: JsonBodyOf<P, M>, options: { params: PathParamsOf<P, M> }];

function runMutate<
  M extends 'post' | 'put' | 'patch',
  P extends PathsWithMethod<M>,
>(method: M, path: P, args: unknown[]): Promise<ResponseDataOf<P, M>> {
  const hasTemplateParam = path.includes('{');
  if (
    hasTemplateParam &&
    args.length === 1 &&
    typeof args[0] === 'object' &&
    args[0] !== null &&
    'params' in args[0]
  ) {
    const opts = args[0] as { params: Record<string, string | number> };
    return sendEnvelope(buildUrl(path, opts.params), {
      method: method.toUpperCase(),
    });
  }
  const body = args[0];
  const opts = args[1] as
    | { params?: Record<string, string | number> }
    | undefined;
  return sendEnvelope(buildUrl(path, opts?.params), {
    method: method.toUpperCase(),
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
}

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
    return sendEnvelope(buildUrl(path, opts?.params, opts?.query), {
      method: 'GET',
    });
  },

  post<P extends PathsWithMethod<'post'>>(
    path: P,
    ...args: MutateArgs<'post', P>
  ): Promise<ResponseDataOf<P, 'post'>> {
    return runMutate('post', path, args);
  },

  put<P extends PathsWithMethod<'put'>>(
    path: P,
    ...args: MutateArgs<'put', P>
  ): Promise<ResponseDataOf<P, 'put'>> {
    return runMutate('put', path, args);
  },

  patch<P extends PathsWithMethod<'patch'>>(
    path: P,
    ...args: MutateArgs<'patch', P>
  ): Promise<ResponseDataOf<P, 'patch'>> {
    return runMutate('patch', path, args);
  },

  upload(path: '/api/v1/files/upload', formData: FormData): Promise<FileInfo> {
    return sendEnvelope<FileInfo>(buildUrl(path), {
      method: 'POST',
      body: formData,
    });
  },
};
