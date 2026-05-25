import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { apiRequest } from '@/lib/api';
import { Button } from '@/components/ui/button';

export const Route = createFileRoute('/settings')({
  component: SettingsPage,
});

interface AuthUser {
  id: number;
  email: string;
  tier: string;
  displayName: string;
}

interface BankAccount {
  id: number;
  bankName: string;
  accountNumber: string;
  accountHolderName: string;
  isDefault: boolean;
}

function TierBadge({ tier }: { tier: string }) {
  const configs: Record<string, { label: string; color: string; bg: string }> = {
    free: { label: 'Free', color: 'text-muted-foreground', bg: 'bg-muted' },
    pro: { label: 'PRO', color: 'text-accent', bg: 'bg-accent/20' },
    pro_plus: { label: 'PRO Plus', color: 'text-purple-400', bg: 'bg-purple-500/20' },
  };
  const cfg = configs[tier] ?? configs.free;
  return (
    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${cfg.color} ${cfg.bg}`}>
      {cfg.label}
    </span>
  );
}

function SettingsPage() {
  const { data: user } = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: () => apiRequest<AuthUser>('/api/v1/auth/me'),
    staleTime: 60000,
  });

  const { data: banks } = useQuery({
    queryKey: ['bank-accounts'],
    queryFn: () => apiRequest<{ bankAccounts: BankAccount[] }>('/api/v1/bank-accounts'),
  });

  const tier = user?.tier ?? 'free';

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-md flex-col gap-6 px-4 py-8">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold font-display text-primary">Cài đặt</h1>
        <p className="text-sm text-muted-foreground">Quản lý tài khoản và gói dịch vụ</p>
      </header>

      {/* Account info */}
      <section className="space-y-2">
        <h2 className="text-sm font-semibold text-foreground">Tài khoản</h2>
        <div className="rounded-md border border-input bg-muted p-4 space-y-2">
          <div className="flex justify-between">
            <span className="text-sm text-muted-foreground">Email</span>
            <span className="text-sm font-medium">{user?.email ?? '...'}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-sm text-muted-foreground">Tên hiển thị</span>
            <span className="text-sm font-medium">{user?.displayName ?? '...'}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-sm text-muted-foreground">Gói</span>
            <TierBadge tier={tier} />
          </div>
        </div>
      </section>

      {/* Bank accounts */}
      <section className="space-y-2">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-foreground">Tài khoản ngân hàng</h2>
          <Button variant="outline" size="sm">+ Thêm</Button>
        </div>
        {banks?.bankAccounts && banks.bankAccounts.length > 0 ? (
          <ul className="space-y-2">
            {banks.bankAccounts.map((bank) => (
              <li key={bank.id} className="rounded-md border border-input bg-muted p-4">
                <div className="flex justify-between">
                  <span className="font-medium">{bank.bankName}</span>
                  {bank.isDefault && (
                    <span className="text-xs text-primary">Mặc định</span>
                  )}
                </div>
                <p className="text-sm text-muted-foreground">
                  {bank.accountNumber} · {bank.accountHolderName}
                </p>
              </li>
            ))}
          </ul>
        ) : (
          <div className="rounded-md border border-dashed border-input p-4 text-center text-sm text-muted-foreground">
            Chưa có tài khoản ngân hàng
          </div>
        )}
      </section>

      {/* Upgrade section */}
      {tier === 'free' && (
        <section className="space-y-2">
          <h2 className="text-sm font-semibold text-foreground">Nâng cấp PRO</h2>
          <div className="rounded-md border border-accent/40 bg-accent/10 p-4 space-y-3">
            <div className="space-y-1">
              <p className="font-medium text-accent">PRO - 50k/tháng</p>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>Auto-detect: không cần tick tay từng người</li>
                <li>Tối đa 20 người/group</li>
                <li>Hỗ trợ ưu tiên</li>
              </ul>
            </div>
            <Button variant="accent" size="lg" className="w-full">
              Nâng cấp PRO
            </Button>
          </div>
        </section>
      )}

      {/* PRO Plus */}
      {tier === 'pro' && (
        <section className="space-y-2">
          <h2 className="text-sm font-semibold text-foreground">Nâng cấp PRO Plus</h2>
          <div className="rounded-md border border-purple-500/40 bg-purple-500/10 p-4 space-y-3">
            <div className="space-y-1">
              <p className="font-medium text-purple-400">PRO Plus - 100k/tháng</p>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>Auto-detect: không cần tick tay từng người</li>
                <li>Không giới hạn người chơi</li>
                <li>Hỗ trợ ưu tiên cao cấp</li>
              </ul>
            </div>
            <Button variant="secondary" size="lg" className="w-full">
              Nâng cấp PRO Plus
            </Button>
          </div>
        </section>
      )}
    </main>
  );
}
