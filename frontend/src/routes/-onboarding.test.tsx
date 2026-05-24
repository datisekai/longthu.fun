import { describe, expect, it } from 'vitest';
import { bankAccountSchema } from './onboarding';
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
