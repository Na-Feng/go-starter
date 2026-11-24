import type { User, UserStatus } from '@/api/schema';
import { api } from '@/lib/api';
import { Can } from '@/lib/auth';
import { Card, Message, Switch, Table, Tag } from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

export function UsersPage() {
  const qc = useQueryClient();
  const usersQuery = useQuery({
    queryKey: ['iam', 'users'],
    queryFn: () =>
      api.get('/api/v1/iam/users', { query: { page: 1, page_size: 20 } }),
  });

  const toggleStatus = useMutation({
    mutationFn: ({ id, status }: { id: string; status: UserStatus }) =>
      api.patch(
        '/api/v1/iam/users/{id}/status',
        { status },
        { params: { id } },
      ),
    onSuccess: () => {
      Message.success('状态已更新');
      void qc.invalidateQueries({ queryKey: ['iam', 'users'] });
    },
    onError: (err) => Message.error(err.message),
  });

  const columns: ColumnProps<User>[] = [
    { title: '用户名', dataIndex: 'username' },
    { title: '昵称', dataIndex: 'nickname' },
    {
      title: '状态',
      dataIndex: 'status',
      render: (status: string) => (
        <Tag color={status === 'active' ? 'green' : 'red'}>{status}</Tag>
      ),
    },
    {
      title: '启用',
      render: (_, record) => (
        <Can perm="iam:user:write">
          <Switch
            checked={record.status === 'active'}
            onChange={(checked) =>
              toggleStatus.mutate({
                id: record.id,
                status: checked ? 'active' : 'disabled',
              })
            }
          />
        </Can>
      ),
    },
  ];

  return (
    <Card title="用户与状态管理（RBAC）">
      <Table
        rowKey="id"
        columns={columns}
        loading={usersQuery.isLoading}
        data={usersQuery.data?.items ?? []}
        pagination={false}
      />
    </Card>
  );
}
