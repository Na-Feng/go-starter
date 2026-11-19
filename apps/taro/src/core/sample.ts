// 纯 TypeScript 业务逻辑层：不依赖任何 Taro / wx API，可直接用 Vitest 测试
export function validateSampleTitle(raw: string): { ok: true; value: string } | { ok: false; error: string } {
  const trimmed = raw.trim();
  if (trimmed.length < 2) {
    return { ok: false, error: '标题至少需要 2 个字符' };
  }
  if (trimmed.length > 100) {
    return { ok: false, error: '标题不能超过 100 个字符' };
  }
  return { ok: true, value: trimmed };
}
