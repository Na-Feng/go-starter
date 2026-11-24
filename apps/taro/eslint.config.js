import pluginQuery from '@tanstack/eslint-plugin-query';
import tseslint from 'typescript-eslint';

const SUPPRESS_RE = /\b(eslint-disable|biome-ignore|@ts-ignore|@ts-nocheck)\b/;
const DEBT_MARKER_RE = /\b(TODO|FIXME|HACK|WORKAROUND|XXX)\b/i;

const antiSlopPlugin = {
  rules: {
    'comment-hygiene': {
      meta: {
        type: 'problem',
        schema: [
          {
            type: 'object',
            properties: {
              maxConsecutiveLines: { type: 'integer', minimum: 1 },
              maxFileLines: { type: 'integer', minimum: 1 },
            },
            additionalProperties: false,
          },
        ],
      },
      create(context) {
        const opts = context.options[0] ?? {};
        const maxLines = opts.maxConsecutiveLines ?? 0;
        const maxFileLines = opts.maxFileLines ?? 0;
        const isTestFile = /\.(test|spec)\.[cm]?[jt]sx?$/.test(
          context.filename,
        );

        return {
          Program(node) {
            if (maxFileLines > 0 && !isTestFile) {
              const totalLines = context.sourceCode.lines.length;
              if (totalLines > maxFileLines) {
                context.report({
                  loc: node.loc,
                  message: `业务单文件禁止超过 ${maxFileLines} 行（当前 ${totalLines} 行）：请拆分为独立子组件或 src/core 纯逻辑模块，防止长任务上下文膨胀。`,
                });
              }
            }

            const comments = context.sourceCode.getAllComments();
            const groups = [];

            for (const c of comments) {
              if (SUPPRESS_RE.test(c.value)) {
                context.report({
                  loc: c.loc,
                  message:
                    '禁止通过注释指令（eslint-disable / biome-ignore / @ts-ignore）豁免检查，请直接修复类型或架构根因。',
                });
              }
              if (!isTestFile && DEBT_MARKER_RE.test(c.value)) {
                context.report({
                  loc: c.loc,
                  message:
                    '禁止遗留 TODO/FIXME/HACK/WORKAROUND 补丁注释，请直接完成实现或修复根因。',
                });
              }

              const prev = groups[groups.length - 1];
              if (prev && c.loc.start.line <= prev.endLine + 1) {
                prev.endLine = Math.max(prev.endLine, c.loc.end.line);
              } else {
                groups.push({
                  loc: c.loc,
                  startLine: c.loc.start.line,
                  endLine: c.loc.end.line,
                });
              }
            }

            if (maxLines > 0 && !isTestFile) {
              for (const g of groups) {
                const span = g.endLine - g.startLine + 1;
                if (span > maxLines) {
                  context.report({
                    loc: g.loc,
                    message: `业务模块禁止超过 ${maxLines} 行的长篇解释性注释（当前连续 ${span} 行）：请拆分为语义化纯函数或直接修复根因。`,
                  });
                }
              }
            }
          },
        };
      },
    },
  },
};

export default tseslint.config(
  ...pluginQuery.configs['flat/recommended'],
  {
    files: ['src/**/*.{ts,tsx}'],
    ignores: ['src/api/openapi.gen.d.ts'],
    linterOptions: {
      noInlineConfig: true,
    },
    languageOptions: {
      parser: tseslint.parser,
      parserOptions: { ecmaFeatures: { jsx: true } },
    },
    plugins: {
      '@typescript-eslint': tseslint.plugin,
      'anti-slop': antiSlopPlugin,
    },
    rules: {
      'anti-slop/comment-hygiene': 'error',
      'no-empty': ['error', { allowEmptyCatch: false }],
      '@typescript-eslint/ban-ts-comment': [
        'error',
        {
          'ts-ignore': true,
          'ts-nocheck': true,
          'ts-expect-error': true,
        },
      ],
    },
  },
  {
    files: ['src/core/**/*.{ts,tsx}'],
    plugins: {
      'anti-slop': antiSlopPlugin,
    },
    rules: {
      'anti-slop/comment-hygiene': [
        'error',
        { maxConsecutiveLines: 2, maxFileLines: 500 },
      ],
      'no-restricted-globals': [
        'error',
        {
          name: 'wx',
          message: 'src/core 必须保持纯 TypeScript，禁止直接调用 wx 全局对象。',
        },
        {
          name: 'setTimeout',
          message: 'src/core 必须保持确定性纯函数，禁止使用 setTimeout。',
        },
        {
          name: 'setInterval',
          message: 'src/core 必须保持确定性纯函数，禁止使用 setInterval。',
        },
      ],
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['@tarojs/*'],
              message:
                'src/core 严禁导入 @tarojs/*，确保核心业务逻辑可脱离微信模拟器直接运行 Vitest。',
            },
            {
              group: ['@/subpackages/*'],
              message: 'src/core 严禁反向依赖分包页面。',
            },
          ],
        },
      ],
    },
  },
  {
    files: ['src/pages/**/*.{ts,tsx}', 'src/subpackages/**/*.{ts,tsx}'],
    plugins: {
      'anti-slop': antiSlopPlugin,
    },
    rules: {
      'anti-slop/comment-hygiene': [
        'error',
        { maxConsecutiveLines: 2, maxFileLines: 500 },
      ],
      'no-restricted-globals': [
        'error',
        {
          name: 'fetch',
          message:
            '小程序禁止使用浏览器 fetch，请统一走 src/lib/request.ts 的 api。',
        },
        {
          name: 'setTimeout',
          message:
            '禁止在小程序页面使用 setTimeout 打时序补丁，请使用生命周期或 TanStack Query。',
        },
        {
          name: 'setInterval',
          message:
            '禁止在小程序页面使用 setInterval，轮询请使用 TanStack Query refetchInterval。',
        },
      ],
      'no-restricted-syntax': [
        'error',
        {
          selector: "CallExpression[callee.name='useEffect']",
          message:
            '禁止在小程序页面使用 useEffect：数据请求请用 TanStack Query，页面曝光刷新请用 useDidShow。',
        },
        {
          selector:
            "CallExpression[callee.object.name='React'][callee.property.name='useEffect']",
          message:
            '禁止在小程序页面使用 React.useEffect：数据请求请用 TanStack Query，曝光刷新请用 useDidShow。',
        },
        {
          selector:
            "MemberExpression[object.name='Taro'][property.name='request']",
          message:
            '禁止在页面直接调用 Taro.request，请统一使用 src/lib/request.ts。',
        },
        {
          selector:
            "MemberExpression[object.name='Taro'][property.name=/^(getStorage|setStorage|removeStorage|clearStorage)(Sync)?$/]",
          message:
            '禁止在页面直接读写 Taro Storage，请统一通过 src/lib/ 封装访问。',
        },
        {
          selector:
            'JSXOpeningElement[name.name=/^(div|span|button|input|p|a|img|ul|li|form)$/]',
          message:
            '小程序禁止使用原生 HTML 标签，请改用 @tarojs/components（View/Text/Button/Input）。',
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
              message:
                '禁止主包或公共库反向引用分包代码，以守住微信小程序 2MB 主包体积限制。',
            },
          ],
        },
      ],
    },
  },
);
