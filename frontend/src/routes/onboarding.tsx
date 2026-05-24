import { createFileRoute } from '@tanstack/react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { AuthGuard } from '@/components/auth/AuthGuard';
import { Button } from '@/components/ui/button';
import { FormField } from '@/components/ui/form-field';
import { ApiError, apiRequest } from '@/lib/api';
import { vi } from '@/locales/vi';
import type { BankAccount } from '@/types/api';

export const bankOptions = [
  { code: 'MBBANK', label: vi.onboarding.bank.bankOptions.mbbank },
  { code: 'VCB', label: vi.onboarding.bank.bankOptions.vcb },
  { code: 'TPB', label: vi.onboarding.bank.bankOptions.tpb },
] as const;

const bankCodes = bankOptions.map((bank) => bank.code) as [string, ...string[]];

export const bankAccountSchema = z.object({
  bankCode: z.enum(bankCodes, { message: vi.onboarding.bank.errors.bankRequired }),
  accountNumber: z.string().regex(/^[0-9]{8,16}$/, vi.onboarding.bank.errors.accountNumberInvalid),
  accountHolderName: z.string().trim().min(1, vi.onboarding.bank.errors.holderRequired),
});

type BankAccountValues = z.infer<typeof bankAccountSchema>;

export const Route = createFileRoute('/onboarding')({
  component: OnboardingRoute,
});

function OnboardingRoute() {
  return (
    <AuthGuard>
      <OnboardingPage />
    </AuthGuard>
  );
}

export function OnboardingPage() {
  const [step, setStep] = useState<1 | 2>(1);
  const form = useForm<BankAccountValues>({
    resolver: zodResolver(bankAccountSchema),
    defaultValues: {
      bankCode: 'MBBANK',
      accountNumber: '',
      accountHolderName: '',
    },
  });
  const createBank = useMutation({
    mutationFn: (values: BankAccountValues) =>
      apiRequest<BankAccount>('/api/v1/bank-accounts', { method: 'POST', body: values }),
    onSuccess: () => setStep(2),
  });
  const submitError =
    createBank.error instanceof ApiError
      ? createBank.error.problem.title
      : createBank.isError
        ? vi.auth.errors.generic
        : undefined;

  async function onSubmit(values: BankAccountValues) {
    await createBank.mutateAsync(values);
  }

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-md flex-col justify-center gap-6 px-6 py-10">
      <header className="space-y-2">
        <p className="text-sm font-medium text-primary">{vi.onboarding.stepLabel}</p>
        <h1 className="text-3xl font-bold text-foreground">{vi.onboarding.title}</h1>
      </header>

      {step === 1 ? (
        <section className="space-y-5">
          <div className="space-y-2">
            <h2 className="text-xl font-semibold text-foreground">{vi.onboarding.bank.title}</h2>
            <p className="text-sm leading-6 text-muted-foreground">{vi.onboarding.bank.subtitle}</p>
          </div>

          <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)} noValidate>
            <div className="flex flex-col gap-1.5">
              <label className="text-sm font-medium leading-none text-foreground" htmlFor="bank-code">
                {vi.onboarding.bank.bankLabel}
              </label>
              <select
                id="bank-code"
                className="flex h-11 w-full rounded-md border border-input bg-muted px-3 py-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                aria-invalid={form.formState.errors.bankCode ? 'true' : undefined}
                {...form.register('bankCode')}
              >
                {bankOptions.map((bank) => (
                  <option key={bank.code} value={bank.code}>
                    {bank.label}
                  </option>
                ))}
              </select>
              {form.formState.errors.bankCode ? (
                <p className="text-xs text-destructive" role="alert">
                  {form.formState.errors.bankCode.message}
                </p>
              ) : null}
            </div>

            <FormField
              fieldId="account-number"
              label={vi.onboarding.bank.accountNumberLabel}
              placeholder={vi.onboarding.bank.accountNumberPlaceholder}
              inputMode="numeric"
              autoComplete="off"
              error={form.formState.errors.accountNumber?.message}
              {...form.register('accountNumber')}
            />
            <FormField
              fieldId="account-holder-name"
              label={vi.onboarding.bank.accountHolderLabel}
              placeholder={vi.onboarding.bank.accountHolderPlaceholder}
              autoComplete="name"
              hint={vi.onboarding.bank.helper}
              error={form.formState.errors.accountHolderName?.message}
              {...form.register('accountHolderName')}
            />

            {submitError ? (
              <p className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive" role="alert">
                {submitError}
              </p>
            ) : null}

            <Button type="submit" size="lg" className="w-full" disabled={createBank.isPending}>
              {createBank.isPending ? vi.onboarding.bank.submitting : vi.onboarding.bank.submit}
            </Button>
          </form>
        </section>
      ) : (
        <section className="space-y-3">
          <h2 className="text-xl font-semibold text-foreground">{vi.onboarding.step2.title}</h2>
          <p className="text-sm leading-6 text-muted-foreground">{vi.onboarding.step2.body}</p>
        </section>
      )}
    </main>
  );
}
