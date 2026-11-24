import { validateSampleTitle } from '@/core/sample';
import { api } from '@/lib/request';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Button, Input, Text, View } from '@tarojs/components';
import Taro, { useDidShow } from '@tarojs/taro';
import { useState } from 'react';

export default function SampleSubpackagePage() {
  const [title, setTitle] = useState('');
  const qc = useQueryClient();

  const listQuery = useQuery({
    queryKey: ['samples', 1],
    queryFn: () =>
      api.get('/api/v1/samples', { query: { page: 1, page_size: 20 } }),
  });

  useDidShow(() => {
    void listQuery.refetch();
  });

  const createMut = useMutation({
    mutationFn: (validTitle: string) =>
      api.post('/api/v1/samples', { title: validTitle }),
    onSuccess: () => {
      setTitle('');
      void qc.invalidateQueries({ queryKey: ['samples'] });
    },
  });

  const handleSubmit = () => {
    const check = validateSampleTitle(title);
    if (!check.ok) {
      void Taro.showToast({ title: check.error, icon: 'none' });
      return;
    }
    createMut.mutate(check.value);
  };

  return (
    <View style={{ padding: '24px' }}>
      <View style={{ display: 'flex', marginBottom: '16px' }}>
        <Input
          value={title}
          onInput={(e) => setTitle(e.detail.value)}
          placeholder="输入示例标题..."
          style={{
            flex: 1,
            border: '1px solid #ddd',
            padding: '8px',
            marginRight: '8px',
          }}
        />
        <Button
          type="primary"
          size="mini"
          loading={createMut.isPending}
          onClick={handleSubmit}
        >
          新增
        </Button>
      </View>

      {listQuery.data?.items.map((item) => (
        <View
          key={item.id}
          style={{ padding: '12px', borderBottom: '1px solid #eee' }}
        >
          <Text style={{ display: 'block', fontWeight: 'bold' }}>
            {item.title}
          </Text>
          <Text style={{ fontSize: '12px', color: '#888' }}>{item.id}</Text>
        </View>
      ))}
    </View>
  );
}
