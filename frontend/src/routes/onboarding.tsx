import { createFileRoute } from '@tanstack/react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useMemo, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { z } from 'zod';
import { AuthGuard } from '@/components/auth/AuthGuard';
import { Button } from '@/components/ui/button';
import { FormField } from '@/components/ui/form-field';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { useAuthSession } from '@/hooks/useAuthSession';
import { ApiError, apiRequest } from '@/lib/api';
import { vi } from '@/locales/vi';
import type { BankAccount, Group, Player, PublicUser } from '@/types/api';
import { SessionDraftStep } from './-onboarding-session';

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

export const groupSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, vi.onboarding.group.errors.nameRequired)
    .max(120, vi.onboarding.group.errors.nameTooLong),
});

type GroupValues = z.infer<typeof groupSchema>;

// Tier → max active Players per Group (mirrors backend constants in
// internal/players/service.go). The form caps client-side; the server
// re-enforces and is the source of truth.
const tierCaps = { free: 8, pro: 20, pro_plus: Infinity } as const;
export function capForTier(tier: PublicUser['tier'] | undefined): number {
  if (tier === 'pro') return tierCaps.pro;
  if (tier === 'pro_plus') return tierCaps.pro_plus;
  return tierCaps.free;
}

/**
 * Parse the textarea into trimmed, non-empty lines preserving Vietnamese
 * diacritics. Used by the schema + by the live count summary.
 */
export function parseNames(raw: string): string[] {
  return raw
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line.length > 0);
}

/** Factory: build the players schema with the active tier cap injected. */
export function buildPlayersSchema(cap: number) {
  return z.object({
    namesRaw: z
      .string()
      .superRefine((raw, ctx) => {
        const names = parseNames(raw);
        if (names.length === 0) {
          ctx.addIssue({ code: z.ZodIssueCode.custom, message: vi.onboarding.players.errors.namesRequired });
          return;
        }
        if (names.length > cap) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: vi.onboarding.players.errors.tooMany(cap),
          });
          return;
        }
        if (names.some((n) => n.length > 60)) {
          ctx.addIssue({ code: z.ZodIssueCode.custom, message: vi.onboarding.players.errors.nameTooLong });
          return;
        }
        const seen = new Set<string>();
        for (const n of names) {
          const key = n.toLowerCase();
          if (seen.has(key)) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              message: vi.onboarding.players.errors.duplicateInSubmit,
            });
            return;
          }
          seen.add(key);
        }
      }),
  });
}

type Step = 1 | 2 | 3 | 4;

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
  const { data: banks } = useQuery({
    queryKey: ['bank-accounts'],
    queryFn: () => apiRequest<{ bankAccounts: BankAccount[] }>('/api/v1/bank-accounts'),
  });

  // Skip bank step if user already has at least one account.
  const [step, setStep] = useState<Step>(banks?.bankAccounts?.length ? 2 : 1);
  const [group, setGroup] = useState<Group | null>(null);

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-md flex-col justify-center gap-6 px-6 py-10">
      <header className="space-y-2">
        <p className="text-sm font-medium text-primary">Bước {step}/4</p>
        <h1 className="text-3xl font-bold text-foreground">{vi.onboarding.title}</h1>
      </header>

      {step === 1 ? (
        <BankStep onSuccess={() => setStep(2)} />
      ) : step === 2 ? (
        <GroupStep
          onSuccess={(g) => {
            setGroup(g);
            setStep(3);
          }}
        />
      ) : step === 3 ? (
        group ? (
          <PlayersStep groupId={group.id} onSuccess={() => setStep(4)} />
        ) : null
      ) : group ? (
        <SessionDraftStep groupId={group.id} />
      ) : null}
    </main>
  );
}

function BankStep({ onSuccess }: { onSuccess: () => void }) {
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
    onSuccess,
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
          <Controller
            name="bankCode"
            control={form.control}
            render={({ field }: { field: { value: string; onChange: (v: string) => void } }) => (
              <Select
                value={field.value}
                onValueChange={field.onChange}
              >
                <SelectTrigger
                  id="bank-code"
                  aria-invalid={!!form.formState.errors.bankCode}
                  error={!!form.formState.errors.bankCode}
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {bankOptions.map((bank) => (
                    <SelectItem key={bank.code} value={bank.code}>
                      {bank.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          />
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
          <p
            className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive"
            role="alert"
          >
            {submitError}
          </p>
        ) : null}

        <Button type="submit" size="lg" className="w-full" disabled={createBank.isPending}>
          {createBank.isPending ? vi.onboarding.bank.submitting : vi.onboarding.bank.submit}
        </Button>
      </form>
    </section>
  );
}

export function GroupStep({ onSuccess }: { onSuccess: (group: Group) => void }) {
  const form = useForm<GroupValues>({
    resolver: zodResolver(groupSchema),
    defaultValues: { name: '' },
  });
  const createGroup = useMutation({
    mutationFn: (values: GroupValues) =>
      apiRequest<Group>('/api/v1/groups', { method: 'POST', body: values }),
    onSuccess,
  });
  const submitError =
    createGroup.error instanceof ApiError
      ? createGroup.error.problem.title
      : createGroup.isError
        ? vi.auth.errors.generic
        : undefined;

  async function onSubmit(values: GroupValues) {
    await createGroup.mutateAsync(values);
  }

  return (
    <section className="space-y-5">
      <div className="space-y-2">
        <h2 className="text-xl font-semibold text-foreground">{vi.onboarding.group.title}</h2>
        <p className="text-sm leading-6 text-muted-foreground">{vi.onboarding.group.subtitle}</p>
      </div>

      <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)} noValidate>
        <FormField
          fieldId="group-name"
          label={vi.onboarding.group.nameLabel}
          placeholder={vi.onboarding.group.namePlaceholder}
          autoComplete="off"
          hint={vi.onboarding.group.helper}
          error={form.formState.errors.name?.message}
          {...form.register('name')}
        />

        {submitError ? (
          <p
            className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive"
            role="alert"
          >
            {submitError}
          </p>
        ) : null}

        <Button type="submit" size="lg" className="w-full" disabled={createGroup.isPending}>
          {createGroup.isPending ? vi.onboarding.group.submitting : vi.onboarding.group.submit}
        </Button>
      </form>
    </section>
  );
}

interface PlayersStepProps {
  groupId: number;
  onSuccess: (players: Player[]) => void;
}

export function PlayersStep({ groupId, onSuccess }: PlayersStepProps) {
  const { user } = useAuthSession();
  const cap = capForTier(user?.tier);
  const schema = useMemo(() => buildPlayersSchema(cap), [cap]);
  type Values = z.infer<typeof schema>;

  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: { namesRaw: '' },
  });
  const raw = form.watch('namesRaw');
  const parsed = useMemo(() => parseNames(raw ?? ''), [raw]);

  const createPlayers = useMutation({
    mutationFn: (names: string[]) =>
      apiRequest<{ players: Player[] }>(`/api/v1/groups/${groupId}/players`, {
        method: 'POST',
        body: { names },
      }),
    onSuccess: (res) => onSuccess(res.players),
  });

  const submitError = (() => {
    if (createPlayers.error instanceof ApiError) {
      const fieldErr = createPlayers.error.fieldError('names');
      if (fieldErr) return fieldErr;
      return createPlayers.error.problem.detail ?? createPlayers.error.problem.title;
    }
    if (createPlayers.isError) return vi.auth.errors.generic;
    return undefined;
  })();

  async function onSubmit(_values: Values) {
    await createPlayers.mutateAsync(parsed);
  }

  return (
    <section className="space-y-5">
      <div className="space-y-2">
        <h2 className="text-xl font-semibold text-foreground">{vi.onboarding.players.title}</h2>
        <p className="text-sm leading-6 text-muted-foreground">{vi.onboarding.players.subtitle}</p>
      </div>

      <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)} noValidate>
        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium leading-none text-foreground" htmlFor="player-names">
            {vi.onboarding.players.namesLabel}
          </label>
          <textarea
            id="player-names"
            className="flex min-h-40 w-full rounded-md border border-input bg-muted px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
            placeholder={vi.onboarding.players.namesPlaceholder}
            aria-invalid={form.formState.errors.namesRaw ? 'true' : undefined}
            rows={6}
            {...form.register('namesRaw')}
          />
          <p className="text-xs text-muted-foreground">{vi.onboarding.players.helper}</p>
          <p className="text-xs text-muted-foreground">{vi.onboarding.players.tierHint(cap)}</p>
          {parsed.length > 0 ? (
            <p className="text-xs font-medium text-primary">
              {vi.onboarding.players.summary(parsed.length)}
            </p>
          ) : null}
          {form.formState.errors.namesRaw ? (
            <p className="text-xs text-destructive" role="alert">
              {form.formState.errors.namesRaw.message}
            </p>
          ) : null}
        </div>

        {submitError ? (
          <p
            className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive"
            role="alert"
          >
            {submitError}
          </p>
        ) : null}

        <Button type="submit" size="lg" className="w-full" disabled={createPlayers.isPending}>
          {createPlayers.isPending ? vi.onboarding.players.submitting : vi.onboarding.players.submit}
        </Button>
      </form>
    </section>
  );
}
