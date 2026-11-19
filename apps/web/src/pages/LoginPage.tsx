import { useLogin } from '@/lib/auth';
import { Button, Card, Form, Input, Message, Typography } from '@arco-design/web-react';
import { useNavigate } from 'react-router-dom';

export function LoginPage() {
  const navigate = useNavigate();
  const login = useLogin();

  const handleSubmit = (values: { username: string; password: string }) => {
    login.mutate(values, {
      onSuccess: () => {
        Message.success('登录成功');
        void navigate('/samples', { replace: true });
      },
      onError: (err) => {
        Message.error(err.message);
      },
    });
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-100 p-4">
      <Card className="w-full max-w-sm" title="Go Starter 管理后台">
        <Typography.Text type="secondary" className="mb-4 block text-xs">
          默认账号：admin / admin123
        </Typography.Text>
        <Form
          layout="vertical"
          initialValues={{ username: 'admin', password: 'admin123' }}
          onSubmit={handleSubmit}
        >
          <Form.Item label="用户名" field="username" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input placeholder="请输入用户名" />
          </Form.Item>
          <Form.Item label="密码" field="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password placeholder="请输入密码" />
          </Form.Item>
          <Form.Item className="mb-0">
            <Button type="primary" htmlType="submit" long loading={login.isPending}>
              登录
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
