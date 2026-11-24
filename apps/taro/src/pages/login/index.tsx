import { api, tokenStorage } from '@/lib/request';
import { useMutation } from '@tanstack/react-query';
import { Button, Input, Text, View } from '@tarojs/components';
import Taro from '@tarojs/taro';
import { useState } from 'react';

export default function LoginPage() {
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('admin123');

  const loginMut = useMutation({
    mutationFn: () => api.post('/api/v1/auth/login', { username, password }),
    onSuccess: (pair) => {
      tokenStorage.setTokens(pair);
      void Taro.showToast({ title: '登录成功', icon: 'success' });
      void Taro.navigateBack();
    },
  });

  return (
    <View style={{ padding: '24px' }}>
      <Text
        style={{
          fontSize: '18px',
          fontWeight: 'bold',
          display: 'block',
          marginBottom: '16px',
        }}
      >
        账号登录
      </Text>
      <Input
        value={username}
        onInput={(e) => setUsername(e.detail.value)}
        placeholder="用户名"
        style={{
          border: '1px solid #ddd',
          padding: '10px',
          marginBottom: '12px',
        }}
      />
      <Input
        password
        value={password}
        onInput={(e) => setPassword(e.detail.value)}
        placeholder="密码"
        style={{
          border: '1px solid #ddd',
          padding: '10px',
          marginBottom: '16px',
        }}
      />
      {loginMut.error ? (
        <Text style={{ color: 'red', display: 'block', marginBottom: '12px' }}>
          {loginMut.error.message}
        </Text>
      ) : null}
      <Button
        type="primary"
        loading={loginMut.isPending}
        onClick={() => loginMut.mutate()}
      >
        登录
      </Button>
    </View>
  );
}
