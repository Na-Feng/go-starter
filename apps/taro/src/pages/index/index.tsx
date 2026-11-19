import { Button, Text, View } from '@tarojs/components';
import Taro from '@tarojs/taro';

export default function IndexPage() {
  return (
    <View style={{ padding: '24px' }}>
      <Text style={{ fontSize: '18px', fontWeight: 'bold', display: 'block', marginBottom: '16px' }}>
        Go Starter 小程序骨架
      </Text>
      <Button type="primary" onClick={() => Taro.navigateTo({ url: '/subpackages/sample/index' })}>
        进入示例分包页面
      </Button>
      <Button style={{ marginTop: '12px' }} onClick={() => Taro.navigateTo({ url: '/pages/login/index' })}>
        账号登录
      </Button>
    </View>
  );
}
