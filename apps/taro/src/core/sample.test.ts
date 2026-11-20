import { describe, expect, it } from 'vitest';
import { TARO_FEATURE_MAP } from './feature-map';
import { validateSampleTitle } from './sample';

function createPrng(seed: number): () => number {
  let state = seed >>> 0;
  return () => {
    state = (state * 1664525 + 1013904223) >>> 0;
    return state / 0x100000000;
  };
}

describe('validateSampleTitle unit & property invariants', () => {
  it('trims and validates valid title', () => {
    expect(validateSampleTitle('  hello taro  ')).toEqual({
      ok: true,
      value: 'hello taro',
    });
  });

  it('rejects too short or too long titles', () => {
    expect(validateSampleTitle('a')).toEqual({
      ok: false,
      error: '标题至少需要 2 个字符',
    });
    expect(validateSampleTitle('x'.repeat(101))).toEqual({
      ok: false,
      error: '标题不能超过 100 个字符',
    });
  });

  it('holds whitespace-padding invariance and trim idempotence across 250 random inputs', () => {
    const rand = createPrng(42);
    const chars = 'abcdefghijklmnopqrstuvwxyz0123456789中文标题-_';
    const ws = [' ', '\t', '\n', '  '];

    for (let i = 0; i < 250; i++) {
      const len = Math.floor(rand() * 130);
      let core = '';
      for (let j = 0; j < len; j++) {
        core += chars[Math.floor(rand() * chars.length)] ?? 'a';
      }
      const padLeft = ws[Math.floor(rand() * ws.length)] ?? '';
      const padRight = ws[Math.floor(rand() * ws.length)] ?? '';

      const resRaw = validateSampleTitle(core);
      const resPadded = validateSampleTitle(`${padLeft}${core}${padRight}`);
      expect(resPadded).toEqual(resRaw);

      if (core.length >= 2 && core.length <= 100) {
        expect(resPadded).toEqual({ ok: true, value: core });
        if (resPadded.ok) {
          expect(validateSampleTitle(resPadded.value)).toEqual(resPadded);
        }
      } else {
        expect(resPadded.ok).toBe(false);
      }
    }
  });
});

describe('TARO_FEATURE_MAP route sync with app.config.ts', () => {
  it('matches main pages and subpackages registered in app.config.ts', async () => {
    Object.assign(globalThis, { defineAppConfig: <T>(cfg: T) => cfg });
    const { default: appConfig } = await import('../app.config');
    const mainPages = appConfig.pages ?? [];
    const subPages = (appConfig.subpackages ?? []).flatMap((sp) =>
      sp.pages.map((p) => `${sp.root}/${p}`),
    );

    const mapMain = TARO_FEATURE_MAP.filter((f) => f.bundle === 'main').map(
      (f) => f.pagePath,
    );
    const mapSub = TARO_FEATURE_MAP.filter(
      (f) => f.bundle === 'subpackage',
    ).map((f) => f.pagePath);

    expect(mapMain).toEqual(mainPages);
    expect(mapSub).toEqual(subPages);
    for (const p of mapMain) {
      expect(p.startsWith('subpackages/')).toBe(false);
    }
  });
});
