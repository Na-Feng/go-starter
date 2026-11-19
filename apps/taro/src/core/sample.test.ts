import { describe, expect, it } from 'vitest';
import { validateSampleTitle } from './sample';

describe('validateSampleTitle', () => {
  it('trims and validates valid title', () => {
    expect(validateSampleTitle('  hello taro  ')).toEqual({ ok: true, value: 'hello taro' });
  });

  it('rejects too short title', () => {
    expect(validateSampleTitle('a')).toEqual({ ok: false, error: '标题至少需要 2 个字符' });
  });
});
