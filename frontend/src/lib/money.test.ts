import { describe, it, expect } from 'vitest';
import { formatMoney, parseMoney, moneySpoken } from './money';

describe('formatMoney', () => {
  it('renders full (Vietnamese norm with dot thousands + đ suffix)', () => {
    expect(formatMoney(0)).toBe('0đ');
    expect(formatMoney(1_000)).toBe('1.000đ');
    expect(formatMoney(120_000)).toBe('120.000đ');
    expect(formatMoney(1_234_567)).toBe('1.234.567đ');
  });

  it('renders compact with k / tr buckets, no trailing .0', () => {
    expect(formatMoney(500, 'compact')).toBe('500');
    expect(formatMoney(1_000, 'compact')).toBe('1k');
    expect(formatMoney(1_200, 'compact')).toBe('1.2k');
    expect(formatMoney(120_000, 'compact')).toBe('120k');
    expect(formatMoney(1_000_000, 'compact')).toBe('1tr');
    expect(formatMoney(1_200_000, 'compact')).toBe('1.2tr');
    expect(formatMoney(2_000_000, 'compact')).toBe('2tr');
    expect(formatMoney(1_234_567, 'compact')).toBe('1.2tr');
  });

  it('handles non-finite / negative defensively', () => {
    expect(formatMoney(NaN)).toBe('');
    expect(formatMoney(-120_000)).toBe('-120.000đ');
  });
});

describe('parseMoney', () => {
  it('parses plain integers', () => {
    expect(parseMoney('120000')).toBe(120_000);
    expect(parseMoney('0')).toBe(0);
  });

  it('parses dot-thousands (Vietnamese norm)', () => {
    expect(parseMoney('120.000')).toBe(120_000);
    expect(parseMoney('1.234.567')).toBe(1_234_567);
  });

  it('parses comma-thousands (US convention, tolerated)', () => {
    expect(parseMoney('120,000')).toBe(120_000);
  });

  it('parses compact k / nghìn forms', () => {
    expect(parseMoney('120k')).toBe(120_000);
    expect(parseMoney('120 nghìn')).toBe(120_000);
    expect(parseMoney('120nghin')).toBe(120_000);
    expect(parseMoney('1.5k')).toBe(1_500);
  });

  it('parses compact tr / triệu forms', () => {
    expect(parseMoney('1tr')).toBe(1_000_000);
    expect(parseMoney('1.5tr')).toBe(1_500_000);
    expect(parseMoney('1.5 triệu')).toBe(1_500_000);
    expect(parseMoney('1.2tr')).toBe(1_200_000);
  });

  it('strips trailing đ / đồng / vnđ', () => {
    expect(parseMoney('120.000đ')).toBe(120_000);
    expect(parseMoney('120 nghìn đồng')).toBe(120_000);
    expect(parseMoney('120000 vnđ')).toBe(120_000);
  });

  it('is whitespace + case tolerant', () => {
    expect(parseMoney('  120 K  ')).toBe(120_000);
    expect(parseMoney('1.2 TR')).toBe(1_200_000);
  });

  it('throws on empty / garbage / negative', () => {
    expect(() => parseMoney('')).toThrow();
    expect(() => parseMoney('   ')).toThrow();
    expect(() => parseMoney('abc')).toThrow();
    expect(() => parseMoney('-1000')).toThrow();
  });
});

describe('round-trip (compact → parse)', () => {
  it('preserves round amounts exactly', () => {
    for (const n of [1_000, 120_000, 1_000_000, 1_200_000, 2_000_000]) {
      expect(parseMoney(formatMoney(n, 'compact'))).toBe(n);
    }
  });

  it('rounds 1234567 down to 1.2tr precision (compact is lossy by design)', () => {
    const formatted = formatMoney(1_234_567, 'compact');
    expect(formatted).toBe('1.2tr');
    expect(parseMoney(formatted)).toBe(1_200_000);
  });
});

describe('round-trip (full → parse)', () => {
  it('preserves any amount exactly', () => {
    for (const n of [0, 1_000, 120_000, 1_234_567, 999_999_999]) {
      expect(parseMoney(formatMoney(n))).toBe(n);
    }
  });
});

describe('moneySpoken', () => {
  it('matches AC examples from Story 1.1 Dev Notes', () => {
    expect(moneySpoken(120_000)).toBe('120 nghìn đồng');
    expect(moneySpoken(1_234_567)).toBe('1 triệu 200 nghìn đồng');
  });

  it('drops the nghìn segment when amount is exact millions', () => {
    expect(moneySpoken(1_000_000)).toBe('1 triệu đồng');
    expect(moneySpoken(2_000_000)).toBe('2 triệu đồng');
  });

  it('renders fractional millions to one-tenth precision', () => {
    expect(moneySpoken(1_500_000)).toBe('1 triệu 500 nghìn đồng');
    expect(moneySpoken(2_500_000)).toBe('2 triệu 500 nghìn đồng');
  });

  it('handles sub-thousand and zero', () => {
    expect(moneySpoken(0)).toBe('0 đồng');
    expect(moneySpoken(500)).toBe('500 đồng');
    expect(moneySpoken(999)).toBe('999 đồng');
  });

  it('rounds sub-million amounts to nearest thousand', () => {
    expect(moneySpoken(12_499)).toBe('12 nghìn đồng');
    expect(moneySpoken(12_500)).toBe('13 nghìn đồng');
  });

  it('returns empty string for invalid inputs', () => {
    expect(moneySpoken(NaN)).toBe('');
    expect(moneySpoken(-120_000)).toBe('');
  });
});
