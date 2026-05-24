import { Link, createFileRoute, useNavigate } from '@tanstack/react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { Button } from '@/components/ui/button';
import { FormField } from '@/components/ui/form-field';
import { useAuthSession } from '@/hooks/useAuthSession';
import { ApiError } from '@/lib/api';
import { vi } from '@/locales/vi';

export const loginSchema = z.object({
  email: z.string().email(vi.auth.errors.emailInvalid),
  password: z.string().min(8, vi.auth.errors.passwordShort),
});

type LoginValues = z.infer<typeof loginSchema>;

export const Route = createFileRoute('/login')({
  component: LoginPage,
});

export function LoginPage() {
  const navigate = useNavigate();
  const { login } = useAuthSession();
  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: '', password: '' },
  });
  const submitError =
    login.error instanceof ApiError ? login.error.problem.title : login.isError ? vi.auth.errors.generic : undefined;

  async function onSubmit(values: LoginValues) {
    await login.mutateAsync(values);
    await navigate({ to: '/' });
  }

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-md flex-col justify-center gap-6 px-6 py-10">
      <header className="space-y-2">
        <h1 className="text-3xl font-bold text-foreground">{vi.auth.login.title}</h1>
        <p className="text-sm text-muted-foreground">{vi.auth.login.subtitle}</p>
      </header>

      <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)} noValidate>
        <FormField
          fieldId="login-email"
          label={vi.auth.login.emailLabel}
          type="email"
          autoComplete="email"
          error={form.formState.errors.email?.message}
          {...form.register('email')}
        />
        <FormField
          fieldId="login-password"
          label={vi.auth.login.passwordLabel}
          type="password"
          autoComplete="current-password"
          error={form.formState.errors.password?.message}
          {...form.register('password')}
        />

        {submitError ? (
          <p className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive" role="alert">
            {submitError}
          </p>
        ) : null}

        <Button type="submit" size="lg" className="w-full" disabled={login.isPending}>
          {login.isPending ? vi.auth.login.submitting : vi.auth.login.submit}
        </Button>
      </form>

      <nav className="flex flex-col gap-2 text-sm">
        <Link to="/auth/reset" className="text-primary underline-offset-4 hover:underline">
          {vi.auth.login.goReset}
        </Link>
        <Link to="/register" className="text-muted-foreground underline-offset-4 hover:text-foreground hover:underline">
          {vi.auth.login.goRegister}
        </Link>
      </nav>
    </main>
  );
}
