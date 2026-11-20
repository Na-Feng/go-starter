import type { PageResult, SampleItem } from '@/api/schema';
import { apiRequest } from '@/lib/api';
import { Can } from '@/lib/auth';
import {
  Button,
  Card,
  Form,
  Input,
  Message,
  Space,
  Table,
  Upload,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';

const columns: ColumnProps<SampleItem>[] = [
  { title: 'ID', dataIndex: 'id', width: 200 },
  { title: '标题', dataIndex: 'title' },
  { title: '创建时间', dataIndex: 'created_at', width: 240 },
];

export function SamplesPage() {
  const [page, setPage] = useState(1);
  const [form] = Form.useForm<{ title: string }>();
  const qc = useQueryClient();

  const listQuery = useQuery({
    queryKey: ['samples', page],
    queryFn: () =>
      apiRequest<PageResult<SampleItem>>(
        `/api/v1/samples?page=${page}&page_size=10`,
      ),
  });

  const createMut = useMutation({
    mutationFn: (title: string) =>
      apiRequest<SampleItem>('/api/v1/samples', {
        method: 'POST',
        body: JSON.stringify({ title }),
      }),
    onSuccess: () => {
      form.resetFields();
      Message.success('创建成功');
      void qc.invalidateQueries({ queryKey: ['samples'] });
    },
    onError: (err) => Message.error(err.message),
  });

  return (
    <Card
      title="示例业务模块（Arco Table + Upload）"
      extra={
        <Space>
          <Upload
            showUploadList={false}
            customRequest={async (option) => {
              const fd = new FormData();
              fd.append('file', option.file);
              try {
                const res = await apiRequest<{ url: string }>(
                  '/api/v1/files/upload',
                  {
                    method: 'POST',
                    body: fd,
                  },
                );
                Message.success(`上传成功：${res.url}`);
                option.onSuccess(res);
              } catch (err) {
                Message.error(err instanceof Error ? err.message : '上传失败');
              }
            }}
          >
            <Button>上传文件/图片</Button>
          </Upload>
          <Can perm="sample:write">
            <Form
              form={form}
              layout="inline"
              onSubmit={(values) => {
                if (values.title?.trim()) createMut.mutate(values.title.trim());
              }}
            >
              <Form.Item
                field="title"
                rules={[{ required: true, message: '请输入标题' }]}
              >
                <Input placeholder="输入标题..." />
              </Form.Item>
              <Form.Item>
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={createMut.isPending}
                >
                  新增
                </Button>
              </Form.Item>
            </Form>
          </Can>
        </Space>
      }
    >
      <Table
        rowKey="id"
        columns={columns}
        loading={listQuery.isLoading}
        data={listQuery.data?.items ?? []}
        pagination={{
          current: page,
          pageSize: 10,
          total: listQuery.data?.total ?? 0,
          onChange: setPage,
        }}
      />
    </Card>
  );
}
