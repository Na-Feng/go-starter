import type { AuditEntry } from '@/api/schema';
import { api } from '@/lib/api';
import { Card, Table } from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table';
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';

const columns: ColumnProps<AuditEntry>[] = [
  { title: '操作人', dataIndex: 'actor_id', width: 180 },
  { title: '动作', dataIndex: 'action', width: 180 },
  { title: '目标类型', dataIndex: 'target_type', width: 120 },
  { title: '目标 ID', dataIndex: 'target_id', width: 180 },
  { title: 'IP', dataIndex: 'ip', width: 140 },
  { title: '时间', dataIndex: 'created_at' },
];

export function AuditLogsPage() {
  const [page, setPage] = useState(1);
  const query = useQuery({
    queryKey: ['iam', 'audit-logs', page],
    queryFn: () =>
      api.get('/api/v1/iam/audit-logs', { query: { page, page_size: 20 } }),
  });

  return (
    <Card title="操作审计日志">
      <Table
        rowKey="id"
        columns={columns}
        loading={query.isLoading}
        data={query.data?.items ?? []}
        pagination={{
          current: page,
          pageSize: 20,
          total: query.data?.total ?? 0,
          onChange: setPage,
        }}
      />
    </Card>
  );
}
