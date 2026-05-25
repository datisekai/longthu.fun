import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { formatMoney } from '@/lib/money';
import { vi } from '@/locales/vi';
import { apiRequest } from '@/lib/api';
import { Button } from '@/components/ui/button';

export const Route = createFileRoute('/dashboard')({
  component: DashboardPage,
});

interface DashboardData {
  totalUnpaid: number;
  recentSessions: Array<{
    sessionId: number;
    date: string;
    title?: string;
    groupId: number;
    groupName: string;
    shareCode?: string;
    totalCost: number;
  }>;
  playersWithUnpaid: Array<{
    playerId: number;
    playerName: string;
    groupId: number;
    groupName: string;
    totalUnpaid: number;
    playerCode: string | undefined;
  }>;
  groupCount: number;
  sessionCount: number;
  suspectedCount: number;
  unmatchedCount: number;
}

interface AuthUser {
  id: number;
  email: string;
  tier: string;
  displayName: string;
}

// Action banner variants (Story 3.3)
function ActionBanner({ sessionCount, tier, suspectedCount }: {
  sessionCount: number;
  tier: string;
  suspectedCount: number;
}) {
  // No sessions yet
  if (sessionCount === 0) {
    return (
      <div className="rounded-md border border-primary/40 bg-primary/10 p-4 text-center">
        <p className="text-muted-foreground">Chưa có buổi nào hết á 🏸</p>
        <p className="mt-1 text-sm">Đánh xong nhớ tạo session liền nha</p>
        <Button variant="primary" size="lg" className="mt-3 w-full">
          + Tạo session
        </Button>
      </div>
    );
  }

  // Free tier upsell
  if (tier === 'free') {
    return (
      <div className="rounded-md border border-accent/40 bg-accent/10 p-4 text-center">
        <p className="text-muted-foreground">Còn phải tick tay từng người sao?</p>
        <p className="mt-1 text-sm font-medium text-accent">Lên PRO để app tự khớp 🚀</p>
        <Button variant="accent" size="lg" className="mt-3 w-full">
          Xem PRO 50k/tháng
        </Button>
      </div>
    );
  }

  // PRO/PRO Plus with suspected payments
  if (tier !== 'free' && suspectedCount > 0) {
    return (
      <div className="rounded-md border border-yellow-500/40 bg-yellow-500/10 p-4 text-center">
        <p className="text-muted-foreground">Có {suspectedCount} payment cần xác nhận</p>
        <p className="mt-1 text-sm">Vào xem và xử lý ngay nào</p>
        <a href="/dashboard/suspected">
          <Button variant="primary" size="lg" className="mt-3 w-full">
            Xem ngay
          </Button>
        </a>
      </div>
    );
  }

  return null;
}

// Copy reminder for a player (Story 3.5)
function copyReminder(playerName: string, amount: number, playerCode: string) {
  const msg = `Ê ${playerName} còn nợ ${formatMoney(amount)} qua nè nha 👀\nVào link trả + scan QR đi:\n${window.location.origin}/p/${playerCode}`;
  navigator.clipboard.writeText(msg).catch(() => {});
}

function copyLink(shareCode: string) {
  const url = `${window.location.origin}/g/${shareCode}`;
  navigator.clipboard.writeText(url).catch(() => {});
}

function DashboardPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['dashboard'],
    queryFn: () => apiRequest<DashboardData>('/api/v1/dashboard'),
    refetchInterval: 30000,
  });

  const { data: authData } = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: () => apiRequest<AuthUser>('/api/v1/auth/me'),
    staleTime: 60000,
  });

  if (isLoading) {
    return (
      <main className="mx-auto flex min-h-screen w-full max-w-md flex-col items-center justify-center gap-4 px-4 py-8">
        <div className="text-4xl">🏸</div>
        <p className="text-muted-foreground">Đang tải...</p>
      </main>
    );
  }

  const dash = data ?? { totalUnpaid: 0, recentSessions: [], playersWithUnpaid: [], groupCount: 0, sessionCount: 0, suspectedCount: 0, unmatchedCount: 0 };
  const tier = authData?.tier ?? 'free';

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-md flex-col gap-6 px-4 py-8">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold font-display text-primary">{vi.app.name}</h1>
        <p className="text-sm text-muted-foreground">{vi.dashboard.title}</p>
      </header>

      {/* Action banner */}
      <ActionBanner
        sessionCount={dash.sessionCount}
        tier={tier}
        suspectedCount={dash.suspectedCount}
      />

      {/* Money strip */}
      <div className="grid grid-cols-3 gap-3">
        <div className="rounded-md border border-input bg-muted p-3 text-center">
          <p className="text-2xl font-bold font-display text-primary">
            {formatMoney(dash.totalUnpaid, 'compact')}
          </p>
          <p className="text-xs text-muted-foreground">Cần thu</p>
        </div>
        <a href="/dashboard/suspected" className="rounded-md border border-input bg-muted p-3 text-center hover:border-yellow-500/50">
          <p className={`text-2xl font-bold font-display ${dash.suspectedCount > 0 ? 'text-yellow-500 animate-pulse' : 'text-muted-foreground'}`}>
            {dash.suspectedCount}
          </p>
          <p className="text-xs text-muted-foreground">Cần xác nhận</p>
        </a>
        <a href="/dashboard/unmatched" className="rounded-md border border-input bg-muted p-3 text-center hover:border-orange-500/50">
          <p className={`text-2xl font-bold font-display ${dash.unmatchedCount > 0 ? 'text-orange-500 animate-pulse' : 'text-muted-foreground'}`}>
            {dash.unmatchedCount}
          </p>
          <p className="text-xs text-muted-foreground">Chưa khớp</p>
        </a>
      </div>

      {/* Recent sessions */}
      {dash.recentSessions.length > 0 && (
        <section className="space-y-2">
          <h2 className="text-sm font-semibold text-foreground">{vi.dashboard.recentSessions}</h2>
          <ul className="space-y-2">
            {dash.recentSessions.map((s) => (
              <li key={s.sessionId} className="flex items-center justify-between rounded-md border border-input bg-muted px-4 py-3">
                <div>
                  <p className="font-medium text-foreground">{s.title ?? vi.app.billDefault}</p>
                  <p className="text-xs text-muted-foreground">{s.date} · {s.groupName}</p>
                </div>
                <div className="flex items-center gap-2">
                  {s.shareCode && (
                    <button
                      onClick={() => copyLink(s.shareCode!)}
                      className="rounded-md bg-primary/20 px-2 py-1 text-xs text-primary hover:bg-primary/30"
                    >
                      Copy link
                    </button>
                  )}
                  <p className="font-semibold text-primary">{formatMoney(s.totalCost, 'compact')}</p>
                </div>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* Players with unpaid + copy reminder */}
      {dash.playersWithUnpaid.length > 0 && (
        <section className="space-y-2">
          <h2 className="text-sm font-semibold text-foreground">{vi.dashboard.playersWithUnpaid}</h2>
          <ul className="space-y-2">
            {dash.playersWithUnpaid.map((p) => (
              <li key={p.playerId} className="flex items-center justify-between rounded-md border border-input bg-muted px-4 py-3">
                <div className="flex-1">
                  <p className="font-medium text-foreground">{p.playerName}</p>
                  <p className="text-xs text-muted-foreground">{p.groupName}</p>
                </div>
                <div className="flex items-center gap-2">
                  {p.playerCode && (
                    <button
                      onClick={() => copyReminder(p.playerName, p.totalUnpaid, p.playerCode!)}
                      className="rounded-md bg-secondary/50 px-2 py-1 text-xs text-secondary-foreground hover:bg-secondary"
                    >
                      Copy lời nhắc
                    </button>
                  )}
                  <p className="font-semibold text-red-500">{formatMoney(p.totalUnpaid, 'compact')}</p>
                </div>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* Quick actions */}
      <div className="flex gap-2">
        <Button variant="primary" size="lg" className="flex-1">
          + Tạo session
        </Button>
      </div>
    </main>
  );
}
