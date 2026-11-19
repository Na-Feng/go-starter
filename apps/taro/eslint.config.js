import pluginQuery from '@tanstack/eslint-plugin-query';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  ...pluginQuery.configs['flat/recommended'],
  {
    files: ['src/core/**/*.{ts,tsx}'],
    languageOptions: {
      parser: tseslint.parser,
    },
    rules: {
      'no-restricted-globals': [
        'error',
        { name: 'wx', message: 'src/core 必须保持纯 TypeScript，禁止直接调用 wx 全局对象。' },
      ],
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['@tarojs/*'],
              message: 'src/core 严禁导入 @tarojs/*，确保核心业务逻辑可脱离微信模拟器运行 Vitest。',
            },
          ],
        },
      ],
    },
  },
  {
    files: ['src/pages/**/*.{ts,tsx}', 'src/subpackages/**/*.{ts,tsx}'],
    languageOptions: {
      parser: tseslint.parser,
      parserOptions: { ecmaFeatures: { jsx: true } },
    },
    rules: {
      'no-restricted-globals': [
        'error',
        { name: 'fetch', message: '小程序禁止使用浏览器 fetch，请统一走 src/lib/request.ts 的 apiRequest。' },
      ],
      'no-restricted-syntax': [
        'error',
        {
          selector: "CallExpression[callee.name='useEffect']",
          message: '禁止在小程序页面使用 useEffect：数据请求请用 TanStack Query，页面曝光刷新请用 useDidShow。',
        },
        {
          selector: "MemberExpression[object.name='Taro'][property.name='request']",
          message: '禁止在页面直接调用 Taro.request，请统一使用 src/lib/request.ts。',
        },
        {
          selector: "JSXOpeningElement[name.name=/^(div|span|button|input|p|a)$/]",
          message: '小程序禁止使用原生 HTML 标签，请改用 @tarojs/components（View/Text/Button/Input）。',
        },
      ],
    },
  },
  {
    files: ['src/pages/**/*.{ts,tsx}', 'src/lib/**/*.{ts,tsx}'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['@/subpackages/*'],
              message: '禁止主包或公共库反向引用分包代码，以守住微信小程序 2MB 主包体积限制。',
            },
          ],
        },
      ],
    },
  },
);
