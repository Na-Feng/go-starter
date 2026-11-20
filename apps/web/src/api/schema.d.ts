import type { components, paths } from './openapi.gen';

type PathTemplate<P extends string> =
  P extends `${infer Head}/{${string}}${infer Tail}`
    ? `${Head}/${string}${PathTemplate<Tail>}`
    : P;

export type ApiRoute = keyof paths & string;
export type ApiEndpoint =
  | PathTemplate<ApiRoute>
  | `${PathTemplate<ApiRoute>}?${string}`;

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

export type UserStatus = components['schemas']['UserStatus'];
export type PermissionCode = components['schemas']['PermissionCode'];
export type TokenPair = components['schemas']['TokenPair'];
export type User = components['schemas']['User'];
export type Role = components['schemas']['Role'];
export type Permission = components['schemas']['Permission'];
export type MeResp = components['schemas']['MeResp'];
export type SampleItem = components['schemas']['SampleItem'];
export type AuditEntry = components['schemas']['AuditEntry'];
export type FileInfo = components['schemas']['FileInfo'];
export type MultipartSession = components['schemas']['MultipartSession'];
export type LoginReq = components['schemas']['LoginReq'];
export type CreateSampleReq = components['schemas']['CreateSampleReq'];
