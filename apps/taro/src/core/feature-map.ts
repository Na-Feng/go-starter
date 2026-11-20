import type { ApiRoute } from '@/api/schema';

export interface TaroFeatureEntry {
  readonly id: string;
  readonly bundle: 'main' | 'subpackage';
  readonly pagePath: string;
  readonly title: string;
  readonly apis: readonly ApiRoute[];
  readonly interactions: readonly string[];
}

export const TARO_FEATURE_MAP = [
  {
    id: 'mini.home',
    bundle: 'main',
    pagePath: 'pages/index/index',
    title: 'Go Starter 小程序骨架',
    apis: [],
    interactions: [
      '点击「进入示例分包页面」跳转分包',
      '点击「账号登录」跳转登录页',
    ],
  },
  {
    id: 'mini.login',
    bundle: 'main',
    pagePath: 'pages/login/index',
    title: '账号登录',
    apis: ['/api/v1/auth/login'],
    interactions: ['输入账号密码点击「登录」写入本地令牌并返回上一页'],
  },
  {
    id: 'mini.sample',
    bundle: 'subpackage',
    pagePath: 'subpackages/sample/index',
    title: '示例分包列表与创建',
    apis: ['/api/v1/samples'],
    interactions: [
      'useDidShow 自动刷新列表',
      '通过 validateSampleTitle 校验标题后提交新增',
    ],
  },
] as const satisfies readonly TaroFeatureEntry[];
