// @vitest-environment jsdom
import { App } from '@/App';
import type { PermissionCode } from '@/api/schema';
import { WEB_FEATURE_MAP, WEB_MENU_FEATURES } from '@/feature-map';
import { tokenStore } from '@/lib/api';
import { hasPermission } from '@/lib/auth';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';

const ALL_PERMS: readonly PermissionCode[] = [
  'iam:user:read',
  'iam:user:write',
  'iam:role:read',
  'iam:role:write',
  'iam:audit:read',
  'sample:write',
];

function createPrng(seed: number): () => number {
  let state = seed >>> 0;
  return () => {
    state = (state * 1664525 + 1013904223) >>> 0;
    return state / 0x100000000;
  };
}

afterEach(() => {
  cleanup();
  localStorage.clear();
  vi.restoreAllMocks();
});

describe('WEB_FEATURE_MAP materialized memory & route-DOM sync', () => {
  it('has unique feature ids, unique routes, and non-empty API bindings', () => {
    const ids = new Set(WEB_FEATURE_MAP.map((f) => f.id));
    const routes = new Set(WEB_FEATURE_MAP.map((f) => f.route));
    expect(ids.size).toBe(WEB_FEATURE_MAP.length);
    expect(routes.size).toBe(WEB_FEATURE_MAP.length);

    for (const entry of WEB_FEATURE_MAP) {
      expect(entry.apis.length).toBeGreaterThan(0);
      expect(entry.interactions.length).toBeGreaterThan(0);
    }
    expect(WEB_MENU_FEATURES.map((f) => f.route)).toEqual([
      '/samples',
      '/iam/users',
      '/iam/audit-logs',
    ]);
  });

  it('renders every mapped route in <App /> with its declared title', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input);
      if (url.includes('/api/v1/auth/me')) {
        return new Response(
          JSON.stringify({
            code: 0,
            message: 'ok',
            data: {
              user: {
                id: 'usr_1',
                username: 'admin',
                nickname: '管理员',
                status: 'active',
                created_at: '2026-11-19T00:00:00Z',
              },
              roles: ['super_admin'],
              permissions: ALL_PERMS,
            },
          }),
          { status: 200 },
        );
      }
      return new Response(
        JSON.stringify({
          code: 0,
          message: 'ok',
          data: { items: [], total: 0, page: 1, page_size: 20 },
        }),
        { status: 200 },
      );
    });

    for (const entry of WEB_FEATURE_MAP) {
      cleanup();
      if (entry.authRequired) {
        tokenStore.setTokens({
          access_token: 'tok_access',
          refresh_token: 'tok_refresh',
          expires_in: 3600,
        });
      } else {
        tokenStore.clear();
      }

      const qc = new QueryClient({
        defaultOptions: { queries: { retry: false } },
      });
      render(
        <QueryClientProvider client={qc}>
          <MemoryRouter initialEntries={[entry.route]}>
            <App />
          </MemoryRouter>
        </QueryClientProvider>,
      );

      expect(await screen.findByText(entry.title)).toBeTruthy();
    }
  });
});

describe('hasPermission property-based invariants', () => {
  it('satisfies super_admin dominance, exact membership, and monotonicity across 250 random cases', () => {
    const rand = createPrng(20261119);
    const rolePool = ['operator', 'auditor', 'viewer', 'editor'];

    for (let i = 0; i < 250; i++) {
      const roles = rolePool.filter(() => rand() > 0.5);
      const isSuper = rand() > 0.7;
      if (isSuper) {
        roles.push('super_admin');
      }
      const granted = ALL_PERMS.filter(() => rand() > 0.5);
      const target =
        ALL_PERMS[Math.floor(rand() * ALL_PERMS.length)] ?? 'sample:write';

      const result = hasPermission(roles, granted, target);
      if (isSuper) {
        expect(result).toBe(true);
      } else {
        expect(result).toBe(granted.includes(target));
      }

      if (result) {
        expect(
          hasPermission(roles, [...granted, 'iam:role:read'], target),
        ).toBe(true);
      }
    }
  });
});
