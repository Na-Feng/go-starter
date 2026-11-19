// 运行 `pnpm gen:api` 将自动从 Go 后端 /openapi.yaml 覆盖生成最新类型
export interface Envelope<T> {
  code: number;
  message: string;
  data: T;
  request_id?: string;
}

export interface PageResult<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface SampleItem {
  id: string;
  title: string;
  created_at: string;
}
