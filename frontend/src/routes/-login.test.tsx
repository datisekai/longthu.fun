import { describe, expect, it } from 'vitest';
import { loginSchema } from './login';
import { vi } from '@/locales/vi';

describe('login validation', () => {
  it('rejects an invalid email before any API call is needed', () => {
    const result = loginSchema.safeParse({
      email: 'khong-phai-email',
      password: 'matkhaudai',
    });

    expect(result.success).toBe(false);
    expect(result.error?.flatten().fieldErrors.email).toContain(vi.auth.errors.emailInvalid);
  });

  it('rejects a short password before any API call is needed', () => {
    const result = loginSchema.safeParse({
      email: 'host@example.com',
      password: 'short',
    });

    expect(result.success).toBe(false);
    expect(result.error?.flatten().fieldErrors.password).toContain(vi.auth.errors.passwordShort);
  });

  it('accepts valid login input', () => {
    const result = loginSchema.safeParse({
      email: 'host@example.com',
      password: 'matkhaudai',
    });

    expect(result.success).toBe(true);
  });
});
