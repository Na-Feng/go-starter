import { ESLint } from 'eslint';
import { describe, expect, it } from 'vitest';

const eslint = new ESLint();

async function lintSnippet(code: string, relFile: string): Promise<string[]> {
  const results = await eslint.lintText(code, { filePath: relFile });
  return results.flatMap((r) => r.messages.map((m) => m.message));
}

describe('Taro anti-slop & package isolation ESLint guardrails', () => {
  it('blocks Taro imports in src/core, main->subpackage imports, HTML tags, timers, and comment slop', async () => {
    const cases: ReadonlyArray<{
      file: string;
      code: string;
      expectMsg: string;
    }> = [
      {
        file: 'src/core/bad.ts',
        code: `import Taro from '@tarojs/taro'; export const x = Taro;`,
        expectMsg: 'src/core 严禁导入 @tarojs/*',
      },
      {
        file: 'src/core/bad.ts',
        code: 'export function f() { return wx.getSystemInfoSync(); }',
        expectMsg: '禁止直接调用 wx 全局对象',
      },
      {
        file: 'src/pages/index/bad.tsx',
        code: `import Sub from '@/subpackages/sample/index'; export const x = Sub;`,
        expectMsg: '禁止主包或公共库反向引用分包代码',
      },
      {
        file: 'src/pages/index/bad.tsx',
        code: 'export const Bad = () => <div>html</div>;',
        expectMsg: '小程序禁止使用原生 HTML 标签',
      },
      {
        file: 'src/subpackages/sample/bad.tsx',
        code: `import Taro from '@tarojs/taro'; export const f = () => Taro.getStorageSync('k');`,
        expectMsg: '禁止在页面直接读写 Taro Storage',
      },
      {
        file: 'src/subpackages/sample/bad.tsx',
        code: 'export const f = () => setTimeout(() => {}, 50);',
        expectMsg: '禁止在小程序页面使用 setTimeout',
      },
      {
        file: 'src/subpackages/sample/bad.tsx',
        code: '// line 1\n// line 2\n// line 3\nexport const x = 1;\n',
        expectMsg: '禁止超过 2 行的长篇解释性注释',
      },
      {
        file: 'src/subpackages/sample/bad.tsx',
        code: 'export function swallow() { try { JSON.parse("x"); } catch {} }',
        expectMsg: 'Empty block statement',
      },
      {
        file: 'src/core/giant.ts',
        code: 'export const n = 1;\n'.repeat(502),
        expectMsg: '业务单文件禁止超过 500 行',
      },
    ];

    for (const tc of cases) {
      const msgs = await lintSnippet(tc.code, tc.file);
      expect(
        msgs.some((m) => m.includes(tc.expectMsg)),
        `file "${tc.file}" expected "${tc.expectMsg}", got: ${JSON.stringify(msgs)}`,
      ).toBe(true);
    }
  });
});
