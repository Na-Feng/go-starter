import {
  WEB_MENU_FEATURES,
  WEB_PROTECTED_FEATURES,
  type WebFeatureId,
} from '@/feature-map';
import {
  RequireAuth,
  RequirePermission,
  hasPermission,
  useLogout,
  useMe,
} from '@/lib/auth';
import { AuditLogsPage } from '@/pages/AuditLogsPage';
import { LoginPage } from '@/pages/LoginPage';
import { SamplesPage } from '@/pages/SamplesPage';
import { UsersPage } from '@/pages/UsersPage';
import { Button, Layout, Menu, Space } from '@arco-design/web-react';
import type { ComponentType } from 'react';
import {
  Navigate,
  Outlet,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from 'react-router-dom';

const FEATURE_COMPONENTS: Record<WebFeatureId, ComponentType> = {
  'auth.login': LoginPage,
  'sample.list': SamplesPage,
  'iam.users': UsersPage,
  'iam.audit-logs': AuditLogsPage,
};

function AdminLayout() {
  const { data } = useMe();
  const logout = useLogout();
  const navigate = useNavigate();
  const location = useLocation();

  const visibleMenus = WEB_MENU_FEATURES.filter(
    (item) =>
      !item.routePerm ||
      (data && hasPermission(data.roles, data.permissions, item.routePerm)),
  );

  return (
    <Layout className="min-h-screen">
      <Layout.Sider
        width={210}
        theme="light"
        className="border-r border-slate-200"
      >
        <div className="px-5 py-4 text-base font-semibold text-slate-900">
          Go Starter Admin
        </div>
        <Menu
          selectedKeys={[location.pathname]}
          onClickMenuItem={(key) => navigate(key)}
        >
          {visibleMenus.map((item) => (
            <Menu.Item key={item.route}>{item.menuLabel}</Menu.Item>
          ))}
        </Menu>
      </Layout.Sider>
      <Layout>
        <Layout.Header className="flex items-center justify-end border-b border-slate-200 bg-white px-6 py-3">
          <Space>
            <span className="text-xs text-slate-500">
              当前用户：{data?.user.username}
            </span>
            <Button
              type="text"
              size="small"
              onClick={() =>
                logout.mutate(undefined, {
                  onSettled: () => navigate('/login'),
                })
              }
            >
              退出登录
            </Button>
          </Space>
        </Layout.Header>
        <Layout.Content className="bg-slate-50 p-6">
          <Outlet />
        </Layout.Content>
      </Layout>
    </Layout>
  );
}

export function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        element={
          <RequireAuth>
            <AdminLayout />
          </RequireAuth>
        }
      >
        {WEB_PROTECTED_FEATURES.map((entry) => {
          const PageComp = FEATURE_COMPONENTS[entry.id];
          return (
            <Route
              key={entry.id}
              path={entry.route}
              element={
                <RequirePermission perm={entry.routePerm}>
                  <PageComp />
                </RequirePermission>
              }
            />
          );
        })}
      </Route>
      <Route path="*" element={<Navigate to="/samples" replace />} />
    </Routes>
  );
}
