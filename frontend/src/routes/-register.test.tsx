import { describe, expect, it } from 'vitest';
import { registerSchema } from './register';
import { vi } from '@/locales/vi';

describe('register validation', () => {
  it('rejects an invalid email before any API call is needed', () => {
    const result = registerSchema.safeParse({
      email: 'sai-email',
      password: 'matkhaudai',
      displayName: 'Anh Hung',
    });

    expect(result.success).toBe(false);
    expect(result.error?.flatten().fieldErrors.email).toContain(vi.auth.errors.emailInvalid);
  });

  it('rejects a short password before any API call is needed', () => {
    const result = registerSchema.safeParse({
      email: 'host@example.com',
      password: 'short',
      displayName: 'Anh Hung',
    });

    expect(result.success).toBe(false);
    expect(result.error?.flatten().fieldErrors.password).toContain(vi.auth.errors.passwordShort);
  });

  it('requires a display name', () => {
    const result = registerSchema.safeParse({
      email: 'host@example.com',
      password: 'matkhaudai',
      displayName: '   ',
    });

    expect(result.success).toBe(false);
    expect(result.error?.flatten().fieldErrors.displayName).toContain(vi.auth.errors.displayNameRequired);
  });

  it('accepts valid register input', () => {
    const result = registerSchema.safeParse({
      email: 'host@example.com',
      password: 'matkhaudai',
      displayName: 'Anh Hung',
    });

    expect(result.success).toBe(true);
  });
});
