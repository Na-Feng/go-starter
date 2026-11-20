import type { ApiRoute, PermissionCode } from '@/api/schema';

export interface WebFeatureEntry {
  readonly id: string;
  readonly route: `/${string}`;
  readonly title: string;
  readonly menuLabel?: string;
  readonly authRequired: boolean;
  readonly permissions: readonly PermissionCode[];
  readonly apis: readonly ApiRoute[];
  readonly interactions: readonly string[];
}

export const WEB_FEATURE_MAP: readonly WebFeatureEntry[] = [
  {
    id: 'auth.login',
    route: '/login',
    title: 'Go Starter 管理后台',
    authRequired: false,
    permissions: [],
    apis: ['/api/v1/auth/login'],
    interactions: ['输入用户名与密码', '点击「登录」按钮提交并跳转至 /samples'],
  },
  {
    id: 'sample.list',
    route: '/samples',
    title: '示例业务模块（Arco Table + Upload）',
    menuLabel: '示例业务',
    authRequired: true,
    permissions: ['sample:write'],
    apis: ['/api/v1/samples', '/api/v1/files/upload'],
    interactions: [
      '分页浏览示例列表 Table',
      '点击「上传文件/图片」验证直传与压缩',
      '具备 sample:write 时输入标题点击「新增」',
    ],
  },
  {
    id: 'iam.users',
    route: '/iam/users',
    title: '用户与状态管理（RBAC）',
    menuLabel: '用户管理',
    authRequired: true,
    permissions: ['iam:user:read', 'iam:user:write'],
    apis: ['/api/v1/iam/users', '/api/v1/iam/users/{id}/status'],
    interactions: [
      '查看 RBAC 用户列表与角色状态标签',
      '具备 iam:user:write 时切换启用 Switch 更新账号状态',
    ],
  },
  {
    id: 'iam.audit-logs',
    route: '/iam/audit-logs',
    title: '操作审计日志',
    menuLabel: '审计日志',
    authRequired: true,
    permissions: ['iam:audit:read'],
    apis: ['/api/v1/iam/audit-logs'],
    interactions: ['按页查看关键写操作的审计记录（操作人、动作、目标、IP）'],
  },
];

export const WEB_MENU_FEATURES = WEB_FEATURE_MAP.filter(
  (item): item is WebFeatureEntry & { readonly menuLabel: string } =>
    Boolean(item.menuLabel),
);
