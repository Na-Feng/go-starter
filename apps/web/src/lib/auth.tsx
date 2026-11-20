import type { MeResp, PermissionCode, TokenPair } from '@/api/schema';
import { apiRequest, tokenStore } from '@/lib/api';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { Navigate } from 'react-router-dom';

export function useMe() {
  return useQuery({
    queryKey: ['auth', 'me'],
    queryFn: () => apiRequest<MeResp>('/api/v1/auth/me'),
    enabled: Boolean(tokenStore.getAccess()),
    retry: false,
  });
}

export function useLogin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { username: string; password: string }) =>
      apiRequest<TokenPair>('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    onSuccess: (pair) => {
      tokenStore.setTokens(pair);
      void qc.invalidateQueries({ queryKey: ['auth', 'me'] });
    },
  });
}

export function useLogout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiRequest('/api/v1/auth/logout', { method: 'POST' }),
    onSettled: () => {
      tokenStore.clear();
      qc.clear();
    },
  });
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const { data, isLoading, isError } = useMe();
  if (!tokenStore.getAccess() || isError) {
    return <Navigate to="/login" replace />;
  }
  if (isLoading || !data) {
    return <div className="p-8 text-sm text-slate-500">正在校验登录态...</div>;
  }
  return <>{children}</>;
}

export function hasPermission(
  roles: readonly string[],
  permissions: readonly string[],
  perm: PermissionCode,
): boolean {
  return roles.includes('super_admin') || permissions.includes(perm);
}

export function Can({
  perm,
  children,
}: { perm: PermissionCode; children: ReactNode }) {
  const { data } = useMe();
  if (!data) return null;
  if (hasPermission(data.roles, data.permissions, perm)) {
    return <>{children}</>;
  }
  return null;
}
