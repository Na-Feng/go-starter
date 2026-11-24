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
                  message: `业务单文件禁止超过 ${maxFileLines} 行（当前 ${totalLines} 行）：请拆分为独立子组件、Hooks 或纯逻辑模块，防止长任务上下文膨胀。`,
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
                    message: `业务模块禁止超过 ${maxLines} 行的长篇解释性注释（当前连续 ${span} 行）：请拆分为语义化函数/组件，或直接修复根因。`,
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
    files: ['src/features/**/*.{ts,tsx}', 'src/pages/**/*.{ts,tsx}'],
    plugins: {
      'anti-slop': antiSlopPlugin,
    },
    rules: {
      'anti-slop/comment-hygiene': [
        'error',
        { maxConsecutiveLines: 2, maxFileLines: 500 },
      ],
      'no-alert': 'error',
      'no-restricted-globals': [
        'error',
        {
          name: 'fetch',
          message:
            '禁止在业务页面直接调用裸 fetch：请统一使用 src/lib/api.ts 的 api。',
        },
        {
          name: 'localStorage',
          message:
            '禁止在业务页面直接操作 localStorage：请统一通过 src/lib/ 封装访问。',
        },
        {
          name: 'sessionStorage',
          message:
            '禁止在业务页面直接操作 sessionStorage：请统一通过 src/lib/ 封装访问。',
        },
        {
          name: 'setTimeout',
          message:
            '禁止在业务页面使用 setTimeout 打时序补丁：异步状态请交由 TanStack Query，过渡请用 CSS。',
        },
        {
          name: 'setInterval',
          message:
            '禁止在业务页面使用 setInterval 轮询：请改用 TanStack Query 的 refetchInterval。',
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
          selector:
            "CallExpression[callee.object.name='React'][callee.property.name='useEffect']",
          message:
            '禁止在业务页面/模块中使用 React.useEffect：请改用 TanStack Query 或事件驱动状态。',
        },
        {
          selector:
            "MemberExpression[object.name='window'][property.name=/^(setTimeout|setInterval|localStorage|sessionStorage|fetch)$/]",
          message:
            '禁止通过 window.* 绕过全局 API 限制：请求/缓存请走 src/lib/，异步状态请走 TanStack Query。',
        },
      ],
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              group: ['@/features/*/*'],
              message:
                '禁止跨 Feature 深层引用：请通过公共导出或提升至 src/lib/ 共享。',
            },
          ],
        },
      ],
    },
  },
);
