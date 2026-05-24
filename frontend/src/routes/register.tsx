import { Link, createFileRoute, useNavigate } from '@tanstack/react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { Button } from '@/components/ui/button';
import { FormField } from '@/components/ui/form-field';
import { useAuthSession } from '@/hooks/useAuthSession';
import { ApiError } from '@/lib/api';
import { vi } from '@/locales/vi';

export const registerSchema = z.object({
  email: z.string().email(vi.auth.errors.emailInvalid),
  password: z.string().min(8, vi.auth.errors.passwordShort),
  displayName: z.string().trim().min(1, vi.auth.errors.displayNameRequired),
});

type RegisterValues = z.infer<typeof registerSchema>;

export const Route = createFileRoute('/register')({
  component: RegisterPage,
});

export function RegisterPage() {
  const navigate = useNavigate();
  const { register: registerMutation } = useAuthSession();
  const form = useForm<RegisterValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: { email: '', password: '', displayName: '' },
  });
  const submitError =
    registerMutation.error instanceof ApiError
      ? registerMutation.error.problem.title
      : registerMutation.isError
        ? vi.auth.errors.generic
        : undefined;

  async function onSubmit(values: RegisterValues) {
    await registerMutation.mutateAsync(values);
    await navigate({ to: '/' });
  }

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-md flex-col justify-center gap-6 px-6 py-10">
      <header className="space-y-2">
        <h1 className="text-3xl font-bold text-foreground">{vi.auth.register.title}</h1>
        <p className="text-sm text-muted-foreground">{vi.auth.register.subtitle}</p>
      </header>

      <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)} noValidate>
        <FormField
          fieldId="register-display-name"
          label={vi.auth.register.displayNameLabel}
          placeholder={vi.auth.register.displayNamePlaceholder}
          autoComplete="name"
          error={form.formState.errors.displayName?.message}
          {...form.register('displayName')}
        />
        <FormField
          fieldId="register-email"
          label={vi.auth.register.emailLabel}
          placeholder={vi.auth.register.emailPlaceholder}
          type="email"
          autoComplete="email"
          error={form.formState.errors.email?.message}
          {...form.register('email')}
        />
        <FormField
          fieldId="register-password"
          label={vi.auth.register.passwordLabel}
          type="password"
          autoComplete="new-password"
          hint={vi.auth.register.passwordHint}
          error={form.formState.errors.password?.message}
          {...form.register('password')}
        />

        {submitError ? (
          <p className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive" role="alert">
            {submitError}
          </p>
        ) : null}

        <Button type="submit" size="lg" className="w-full" disabled={registerMutation.isPending}>
          {registerMutation.isPending ? vi.auth.register.submitting : vi.auth.register.submit}
        </Button>
      </form>

      <Link to="/login" className="text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline">
        {vi.auth.register.goLogin}
      </Link>
    </main>
  );
}
