import { describe, expect, it } from 'vitest';
import { computeSplit, todayInVN } from './-onboarding-session';
import type { CostItem } from '@/types/api';

function item(over: Partial<CostItem> = {}): CostItem {
  return {
    id: 1,
    sessionId: 1,
    type: 'court',
    label: 'Sân',
    amount: 100000,
    isIncludedInSplit: true,
    ...over,
  };
}

describe('todayInVN', () => {
  it('returns YYYY-MM-DD shaped string', () => {
    expect(todayInVN()).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  it('matches a known UTC instant in Asia/Ho_Chi_Minh (UTC+7)', () => {
    // 2026-05-23T23:30:00Z = 2026-05-24T06:30 in VN → 2026-05-24
    const fixed = new Date('2026-05-23T23:30:00Z');
    expect(todayInVN(fixed)).toBe('2026-05-24');
  });

  it('treats just-past-midnight UTC as the new day in VN', () => {
    // 2026-05-23T18:00:00Z = 2026-05-24T01:00 in VN → 2026-05-24
    const fixed = new Date('2026-05-23T18:00:00Z');
    expect(todayInVN(fixed)).toBe('2026-05-24');
  });
});

describe('computeSplit', () => {
  it('returns zero per-head when no participants', () => {
    const out = computeSplit([item({ amount: 360000 })], 0);
    expect(out.perHead).toBe(0);
    expect(out.splittable).toBe(360000);
  });

  it('returns zero per-head when no items', () => {
    const out = computeSplit([], 7);
    expect(out.perHead).toBe(0);
  });

  it('splits 600.000đ across 7 → 85.714đ rounded', () => {
    const items = [
      item({ id: 1, type: 'court', amount: 360000 }),
      item({ id: 2, type: 'shuttle', amount: 200000 }),
      item({ id: 3, type: 'water', amount: 40000 }),
    ];
    const out = computeSplit(items, 7);
    expect(out.splittable).toBe(600000);
    // 600000 / 7 = 85714.28... → rounded 85714
    expect(out.perHead).toBe(85714);
  });

  it('discount reduces splittable total when included', () => {
    const items = [
      item({ id: 1, amount: 600000 }),
      item({ id: 2, type: 'discount', label: 'Voucher', amount: -50000 }),
    ];
    const out = computeSplit(items, 5);
    expect(out.splittable).toBe(550000);
    expect(out.perHead).toBe(110000);
  });

  it('non-split items appear in total but not splittable', () => {
    const items = [
      item({ id: 1, amount: 100000, isIncludedInSplit: true }),
      item({ id: 2, amount: 50000, isIncludedInSplit: false }),
    ];
    const out = computeSplit(items, 2);
    expect(out.total).toBe(150000);
    expect(out.splittable).toBe(100000);
    expect(out.perHead).toBe(50000);
  });
});
