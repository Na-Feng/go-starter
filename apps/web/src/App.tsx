import { RequireAuth, useLogout, useMe } from '@/lib/auth';
import { AuditLogsPage } from '@/pages/AuditLogsPage';
import { LoginPage } from '@/pages/LoginPage';
import { SamplesPage } from '@/pages/SamplesPage';
import { UsersPage } from '@/pages/UsersPage';
import { Button, Layout, Menu, Space } from '@arco-design/web-react';
import { Navigate, Outlet, Route, Routes, useLocation, useNavigate } from 'react-router-dom';

function AdminLayout() {
  const { data } = useMe();
  const logout = useLogout();
  const navigate = useNavigate();
  const location = useLocation();

  return (
    <Layout className="min-h-screen">
      <Layout.Sider width={210} theme="light" className="border-r border-slate-200">
        <div className="px-5 py-4 text-base font-semibold text-slate-900">Go Starter Admin</div>
        <Menu selectedKeys={[location.pathname]} onClickMenuItem={(key) => navigate(key)}>
          <Menu.Item key="/samples">示例业务</Menu.Item>
          <Menu.Item key="/iam/users">用户管理</Menu.Item>
          <Menu.Item key="/iam/audit-logs">审计日志</Menu.Item>
        </Menu>
      </Layout.Sider>
      <Layout>
        <Layout.Header className="flex items-center justify-end border-b border-slate-200 bg-white px-6 py-3">
          <Space>
            <span className="text-xs text-slate-500">当前用户：{data?.user.username}</span>
            <Button
              type="text"
              size="small"
              onClick={() => logout.mutate(undefined, { onSettled: () => navigate('/login') })}
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
        <Route path="/samples" element={<SamplesPage />} />
        <Route path="/iam/users" element={<UsersPage />} />
        <Route path="/iam/audit-logs" element={<AuditLogsPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/samples" replace />} />
    </Routes>
  );
}
