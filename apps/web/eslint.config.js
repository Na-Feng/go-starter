import pluginQuery from '@tanstack/eslint-plugin-query';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  ...pluginQuery.configs['flat/recommended'],
  {
    files: ['src/**/*.{ts,tsx}'],
    languageOptions: {
      parser: tseslint.parser,
    },
    rules: {
      '@typescript-eslint/ban-ts-comment': [
        'error',
        {
          'ts-ignore': true,
          'ts-nocheck': true,
          'ts-expect-error': 'allow-with-description',
        },
      ],
    },
  },
  {
    files: ['src/features/**/*.{ts,tsx}', 'src/pages/**/*.{ts,tsx}'],
    rules: {
      'no-alert': 'error',
      'no-restricted-globals': [
        'error',
        {
          name: 'fetch',
          message: '禁止在业务页面直接调用裸 fetch：请统一使用 src/lib/api.ts 的 apiRequest。',
        },
      ],
      'no-restricted-syntax': [
        'error',
        {
          selector: "CallExpression[callee.name='useEffect']",
          message:
            '禁止在业务页面/模块中使用 useEffect：数据请求请用 TanStack Query (useQuery/useMutation)，交互逻辑请写在事件回调中。',
        },
        {
          selector: "CallExpression[callee.object.name='React'][callee.property.name='useEffect']",
          message:
            '禁止在业务页面/模块中使用 React.useEffect：请改用 TanStack Query 或事件驱动状态。',
        },
      ],
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['@/features/*/*'],
              message: '禁止跨 Feature 深层引用：请通过公共导出或提升至 src/lib/ 共享。',
            },
          ],
        },
      ],
    },
  },
);
