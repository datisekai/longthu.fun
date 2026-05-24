/**
 * VND money formatting + parsing helpers.
 *
 * VND has no fractional unit; all amounts are integer values in `đồng`
 * (the smallest unit). See architecture.md §Implementation Patterns
 * "JSON & data formats — Money" and EXPERIENCE.md §Voice and Tone
 * for the voice rules these helpers enforce.
 */

const THOUSAND = 1_000;
const MILLION = 1_000_000;

export type MoneyMode = 'full' | 'compact';

/**
 * Render a VND amount as a Vietnamese-norm string.
 *
 * - `full` (default): dot-separated thousands + `đ` suffix → `"120.000đ"`.
 * - `compact`: bucketed to `k` / `tr` with one decimal max, no trailing zero → `"120k"`, `"1.2tr"`, `"2tr"`.
 *
 * Negative amounts are formatted with a leading `-`; in practice the app
 * never displays negatives (discounts are subtracted at split time).
 */
export function formatMoney(amount: number, mode: MoneyMode = 'full'): string {
  if (!Number.isFinite(amount)) return '';
  const sign = amount < 0 ? '-' : '';
  const abs = Math.abs(amount);

  if (mode === 'compact') {
    if (abs >= MILLION) return sign + compactUnit(abs / MILLION, 'tr');
    if (abs >= THOUSAND) return sign + compactUnit(abs / THOUSAND, 'k');
    return sign + String(abs);
  }

  // 'full' mode — Vietnamese norm uses dot as the thousands separator.
  // Intl.NumberFormat with 'vi-VN' locale uses '.' for thousands.
  return sign + abs.toLocaleString('vi-VN') + 'đ';
}

function compactUnit(value: number, suffix: string): string {
  // One decimal place; trim trailing ".0".
  const rounded = Math.floor(value * 10) / 10;
  const text = rounded.toFixed(1).replace(/\.0$/, '');
  return text + suffix;
}

/**
 * Parse a Vietnamese-form money string back to its integer VND value.
 *
 * Accepts (case-insensitive, whitespace-tolerant):
 * - Plain integer: `"120000"`
 * - Dot-thousands: `"120.000"`, `"1.234.567"`
 * - Comma-thousands: `"120,000"`
 * - Compact `k`/`nghìn`: `"120k"`, `"120 nghìn"`, `"1.5k"`
 * - Compact `tr`/`triệu`: `"1tr"`, `"1.5 triệu"`
 * - Trailing `đ`/`đồng`/`vnđ` (stripped)
 *
 * Throws on empty, garbage, or negative input.
 */
export function parseMoney(input: string): number {
  if (typeof input !== 'string') throw new Error('parseMoney: not a string');
  let s = input.trim().toLowerCase();
  if (!s) throw new Error('parseMoney: empty input');

  // Strip currency suffixes.
  s = s.replace(/\s*(đồng|vnđ|vnd|đ)\s*$/u, '');

  // Compact triệu / tr — must check before generic 'tr' since 'triệu' contains 't' followed by other chars.
  let m = s.match(/^([\d.,]+)\s*(triệu|tr)$/u);
  if (m) {
    const n = parseFloat(m[1].replace(/,/g, '.'));
    if (!Number.isFinite(n)) throw new Error(`parseMoney: not a number "${input}"`);
    return rejectNegative(Math.round(n * MILLION), input);
  }

  // Compact nghìn / k.
  m = s.match(/^([\d.,]+)\s*(nghìn|nghin|k)$/u);
  if (m) {
    const n = parseFloat(m[1].replace(/,/g, '.'));
    if (!Number.isFinite(n)) throw new Error(`parseMoney: not a number "${input}"`);
    return rejectNegative(Math.round(n * THOUSAND), input);
  }

  // Pure number with thousands separators (dot or comma).
  // Vietnamese norm: dot = thousands. We strip all dots and commas.
  if (/^-?[\d.,]+$/.test(s)) {
    const stripped = s.replace(/[.,]/g, '');
    const n = parseInt(stripped, 10);
    if (!Number.isFinite(n)) throw new Error(`parseMoney: not a number "${input}"`);
    return rejectNegative(n, input);
  }

  throw new Error(`parseMoney: cannot parse "${input}"`);
}

function rejectNegative(n: number, original: string): number {
  if (n < 0) throw new Error(`parseMoney: negative not allowed "${original}"`);
  return n;
}

/**
 * Render an amount as Vietnamese spoken form for screen-reader `aria-label`.
 *
 * Matches the COMPACT display's precision (NOT the underlying value's full
 * digit count) so the audio narration aligns with what the user sees on
 * the tile.
 *
 * Examples:
 * - `120000` → `"120 nghìn đồng"`
 * - `1234567` → `"1 triệu 200 nghìn đồng"` (rounds to 1.2tr precision)
 * - `2000000` → `"2 triệu đồng"`
 * - `1500000` → `"1 triệu 500 nghìn đồng"`
 * - `0` → `"0 đồng"`
 */
export function moneySpoken(amount: number): string {
  if (!Number.isFinite(amount) || amount < 0) return '';
  if (amount === 0) return '0 đồng';

  if (amount >= MILLION) {
    // Compact "tr" is value / 1_000_000, rounded down to 1 decimal place.
    // The decimal tenth maps to "× 100 nghìn" (so 1.2tr = 1 triệu 200 nghìn).
    const tenths = Math.floor(amount / 100_000); // 12 for 1234567
    const wholeMillions = Math.floor(tenths / 10); // 1
    const tenthDigit = tenths % 10; // 2
    const parts: string[] = [`${wholeMillions} triệu`];
    if (tenthDigit > 0) parts.push(`${tenthDigit * 100} nghìn`);
    return parts.join(' ') + ' đồng';
  }

  if (amount >= THOUSAND) {
    // Round to nearest thousand (matches the "k" compact display).
    const k = Math.round(amount / THOUSAND);
    return `${k} nghìn đồng`;
  }

  return `${amount} đồng`;
}
