import { describe, expect, it } from 'vitest';
import { bankAccountSchema, buildPlayersSchema, capForTier, groupSchema, parseNames } from './onboarding';
import { vi } from '@/locales/vi';

describe('onboarding bank account validation', () => {
  it('rejects non-numeric account numbers before any API call is needed', () => {
    const result = bankAccountSchema.safeParse({
      bankCode: 'MBBANK',
      accountNumber: 'abc123',
      accountHolderName: 'NGUYEN VAN A',
    });

    expect(result.success).toBe(false);
    expect(result.error?.flatten().fieldErrors.accountNumber).toContain(
      vi.onboarding.bank.errors.accountNumberInvalid,
    );
  });

  it('rejects account numbers outside the accepted length before any API call is needed', () => {
    const result = bankAccountSchema.safeParse({
      bankCode: 'MBBANK',
      accountNumber: '123',
      accountHolderName: 'NGUYEN VAN A',
    });

    expect(result.success).toBe(false);
    expect(result.error?.flatten().fieldErrors.accountNumber).toContain(
      vi.onboarding.bank.errors.accountNumberInvalid,
    );
  });

  it('requires the account holder name', () => {
    const result = bankAccountSchema.safeParse({
      bankCode: 'MBBANK',
      accountNumber: '123456789',
      accountHolderName: '   ',
    });

    expect(result.success).toBe(false);
    expect(result.error?.flatten().fieldErrors.accountHolderName).toContain(
      vi.onboarding.bank.errors.holderRequired,
    );
  });

  it('accepts valid bank account input', () => {
    const result = bankAccountSchema.safeParse({
      bankCode: 'MBBANK',
      accountNumber: '123456789',
      accountHolderName: 'NGUYEN VAN A',
    });

    expect(result.success).toBe(true);
  });
});

describe('onboarding group validation (Story 1.8)', () => {
  it('rejects empty / whitespace-only group name before any API call is needed', () => {
    for (const name of ['', '   ', '\t\n']) {
      const result = groupSchema.safeParse({ name });
      expect(result.success).toBe(false);
      expect(result.error?.flatten().fieldErrors.name).toContain(
        vi.onboarding.group.errors.nameRequired,
      );
    }
  });

  it('rejects group name longer than 120 chars', () => {
    const tooLong = 'a'.repeat(121);
    const result = groupSchema.safeParse({ name: tooLong });

    expect(result.success).toBe(false);
    expect(result.error?.flatten().fieldErrors.name).toContain(
      vi.onboarding.group.errors.nameTooLong,
    );
  });

  it('trims surrounding whitespace and accepts a valid Vietnamese name', () => {
    const result = groupSchema.safeParse({ name: '  Tối thứ 3  ' });

    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.name).toBe('Tối thứ 3');
    }
  });

  it('accepts the maximum length name (120 chars)', () => {
    const maxName = 'a'.repeat(120);
    const result = groupSchema.safeParse({ name: maxName });

    expect(result.success).toBe(true);
  });
});

describe('onboarding players validation (Story 1.9)', () => {
  it('capForTier returns the correct cap for each tier', () => {
    expect(capForTier('free')).toBe(6);
    expect(capForTier('pro')).toBe(8);
    expect(capForTier('pro_plus')).toBe(15);
    expect(capForTier(undefined)).toBe(6); // unknown defaults to free
  });

  it('parseNames trims, drops empty lines, and preserves Vietnamese diacritics', () => {
    const raw = '  Đạt  \n\nLý\n\n  Tâm Đạt\n\t\n';
    expect(parseNames(raw)).toEqual(['Đạt', 'Lý', 'Tâm Đạt']);
  });

  it('rejects empty / whitespace-only input', () => {
    const schema = buildPlayersSchema(6);
    for (const raw of ['', '   ', '\n\n', '\t\t']) {
      const result = schema.safeParse({ namesRaw: raw });
      expect(result.success).toBe(false);
      expect(result.error?.flatten().fieldErrors.namesRaw).toContain(
        vi.onboarding.players.errors.namesRequired,
      );
    }
  });

  it('rejects submission exceeding tier cap (Free=6)', () => {
    const schema = buildPlayersSchema(6);
    const raw = ['A', 'B', 'C', 'D', 'E', 'F', 'G'].join('\n');
    const result = schema.safeParse({ namesRaw: raw });
    expect(result.success).toBe(false);
    expect(result.error?.flatten().fieldErrors.namesRaw).toContain(
      vi.onboarding.players.errors.tooMany(6),
    );
  });

  it('rejects duplicate names within submit (case-insensitive)', () => {
    const schema = buildPlayersSchema(6);
    const raw = 'Đạt\nLý\nđạt';
    const result = schema.safeParse({ namesRaw: raw });
    expect(result.success).toBe(false);
    expect(result.error?.flatten().fieldErrors.namesRaw).toContain(
      vi.onboarding.players.errors.duplicateInSubmit,
    );
  });

  it('rejects names longer than 60 chars', () => {
    const schema = buildPlayersSchema(6);
    const raw = 'Đạt\n' + 'x'.repeat(61);
    const result = schema.safeParse({ namesRaw: raw });
    expect(result.success).toBe(false);
    expect(result.error?.flatten().fieldErrors.namesRaw).toContain(
      vi.onboarding.players.errors.nameTooLong,
    );
  });

  it('accepts a valid list at the tier cap boundary', () => {
    const schema = buildPlayersSchema(8); // PRO
    const raw = ['Đạt', 'Lý', 'Tâm', 'Hùng', 'Long', 'Bảo', 'Minh', 'Phúc'].join('\n');
    const result = schema.safeParse({ namesRaw: raw });
    expect(result.success).toBe(true);
  });
});
