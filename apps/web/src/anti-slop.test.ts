// @vitest-environment node
import { ESLint } from 'eslint';
import { describe, expect, it } from 'vitest';

const eslint = new ESLint();

async function lintSnippet(code: string, relFile: string): Promise<string[]> {
  const results = await eslint.lintText(code, { filePath: relFile });
  return results.flatMap((r) => r.messages.map((m) => m.message));
}

describe('Web anti-slop ESLint guardrails', () => {
  it('blocks useEffect, setTimeout, raw fetch, localStorage, suppressions, and >2 line comments in pages', async () => {
    const cases: ReadonlyArray<{
      name: string;
      code: string;
      expectMsg: string;
    }> = [
      {
        name: 'useEffect',
        code: `import { useEffect } from 'react'; export function P() { useEffect(() => {}, []); return null; }`,
        expectMsg: '禁止在业务页面/模块中使用 useEffect',
      },
      {
        name: 'setTimeout patch',
        code: 'export function P() { setTimeout(() => {}, 100); return null; }',
        expectMsg: '禁止在业务页面使用 setTimeout 打时序补丁',
      },
      {
        name: 'raw fetch',
        code: `export async function P() { await fetch('/api/v1/samples'); }`,
        expectMsg: '禁止在业务页面直接调用裸 fetch',
      },
      {
        name: 'direct localStorage',
        code: `export function P() { return localStorage.getItem('k'); }`,
        expectMsg: '禁止在业务页面直接操作 localStorage',
      },
      {
        name: 'biome-ignore / eslint-disable bypass',
        code: '// biome-ignore lint/suspicious/noExplicitAny: excuse\nexport const x = 1;\n',
        expectMsg: '禁止通过注释指令',
      },
      {
        name: 'HACK marker',
        code: '// HACK: race condition workaround\nexport const x = 1;\n',
        expectMsg: '禁止遗留 TODO/FIXME/HACK/WORKAROUND',
      },
      {
        name: '>2 consecutive comment lines',
        code: '// line 1\n// line 2\n// line 3\nexport const x = 1;\n',
        expectMsg: '禁止超过 2 行的长篇解释性注释',
      },
    ];

    for (const tc of cases) {
      const msgs = await lintSnippet(tc.code, 'src/pages/GuardProbe.tsx');
      expect(
        msgs.some((m) => m.includes(tc.expectMsg)),
        `case "${tc.name}" expected "${tc.expectMsg}", got: ${JSON.stringify(msgs)}`,
      ).toBe(true);
    }
  });

  it('allows clean page with <=2 concise intent comment lines', async () => {
    const clean =
      '// 仅展示当前筛选结果\n// 权限由外层路由把关\nexport const CleanView = () => null;\n';
    const msgs = await lintSnippet(clean, 'src/pages/CleanProbe.tsx');
    expect(msgs).toEqual([]);
  });
});
