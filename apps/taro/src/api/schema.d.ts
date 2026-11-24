import type { components, paths } from './openapi.gen';

export type ApiRoute = keyof paths & string;
export type HttpMethod = 'get' | 'post' | 'put' | 'patch' | 'delete';

export type PathsWithMethod<M extends HttpMethod> = {
  [P in keyof paths]: [NonNullable<paths[P][M]>] extends [never] ? never : P;
}[keyof paths] &
  string;

type OperationOf<P extends keyof paths, M extends HttpMethod> = NonNullable<
  paths[P][M]
>;

export type PathParamsOf<
  P extends keyof paths,
  M extends HttpMethod,
> = OperationOf<P, M> extends {
  parameters: { path: infer PathP };
}
  ? [NonNullable<PathP>] extends [never]
    ? undefined
    : NonNullable<PathP>
  : undefined;

export type QueryParamsOf<
  P extends keyof paths,
  M extends HttpMethod,
> = OperationOf<P, M> extends {
  parameters: { query?: infer QueryP };
}
  ? [NonNullable<QueryP>] extends [never]
    ? undefined
    : NonNullable<QueryP>
  : undefined;

export type JsonBodyOf<
  P extends keyof paths,
  M extends HttpMethod,
> = OperationOf<P, M> extends {
  requestBody: { content: { 'application/json': infer B } };
}
  ? [NonNullable<B>] extends [never]
    ? undefined
    : B
  : undefined;

type SuccessCode = 200 | 201 | 202;

type RawJsonSuccess<Op> = Op extends { responses: infer R }
  ? {
      [S in Extract<keyof R, SuccessCode>]: R[S] extends {
        content: { 'application/json': infer C };
      }
        ? C
        : never;
    }[Extract<keyof R, SuccessCode>]
  : never;

export type ResponseDataOf<
  P extends keyof paths,
  M extends HttpMethod,
> = RawJsonSuccess<OperationOf<P, M>> extends { data: infer D } ? D : never;

export interface Envelope<T> {
  code: number;
  message: string;
  data: T;
  request_id?: string;
}

export type TokenPair = components['schemas']['TokenPair'];
export type SampleItem = components['schemas']['SampleItem'];
export type LoginReq = components['schemas']['LoginReq'];
export type CreateSampleReq = components['schemas']['CreateSampleReq'];
