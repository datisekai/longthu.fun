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
  }>;
  groupCount: number;
  sessionCount: number;
}

function ActionBanner({ sessionCount, groupCount }: { sessionCount: number; groupCount: number }) {
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
  if (groupCount === 0) {
    return (
      <div className="rounded-md border border-primary/40 bg-primary/10 p-4 text-center">
        <p className="text-muted-foreground">Tạo group đầu tiên để bắt đầu</p>
        <Button variant="primary" size="lg" className="mt-3 w-full">
          + Tạo group
        </Button>
      </div>
    );
  }
  return null;
}

function DashboardPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['dashboard'],
    queryFn: () => apiRequest<DashboardData>('/api/v1/dashboard'),
    refetchInterval: 30000,
  });

  if (isLoading) {
    return (
      <main className="mx-auto flex min-h-screen w-full max-w-md flex-col items-center justify-center gap-4 px-4 py-8">
        <div className="text-4xl">🏸</div>
        <p className="text-muted-foreground">Đang tải...</p>
      </main>
    );
  }

  const dash = data ?? { totalUnpaid: 0, recentSessions: [], playersWithUnpaid: [], groupCount: 0, sessionCount: 0 };

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-md flex-col gap-6 px-4 py-8">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold font-display text-primary">{vi.app.name}</h1>
        <p className="text-sm text-muted-foreground">{vi.dashboard.title}</p>
      </header>

      {/* Action banner */}
      <ActionBanner sessionCount={dash.sessionCount} groupCount={dash.groupCount} />

      {/* Money strip */}
      <div className="grid grid-cols-3 gap-3">
        <div className="rounded-md border border-input bg-muted p-3 text-center">
          <p className="text-2xl font-bold font-display text-primary">
            {formatMoney(dash.totalUnpaid, 'compact')}
          </p>
          <p className="text-xs text-muted-foreground">Cần thu</p>
        </div>
        <div className="rounded-md border border-input bg-muted p-3 text-center">
          <p className="text-2xl font-bold font-display text-yellow-500">0</p>
          <p className="text-xs text-muted-foreground">Cần xác nhận</p>
        </div>
        <div className="rounded-md border border-input bg-muted p-3 text-center">
          <p className="text-2xl font-bold font-display text-orange-500">0</p>
          <p className="text-xs text-muted-foreground">Chưa khớp</p>
        </div>
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
                <p className="font-semibold text-primary">{formatMoney(s.totalCost, 'compact')}</p>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* Players with unpaid */}
      {dash.playersWithUnpaid.length > 0 && (
        <section className="space-y-2">
          <h2 className="text-sm font-semibold text-foreground">{vi.dashboard.playersWithUnpaid}</h2>
          <ul className="space-y-2">
            {dash.playersWithUnpaid.map((p) => (
              <li key={p.playerId} className="flex items-center justify-between rounded-md border border-input bg-muted px-4 py-3">
                <div>
                  <p className="font-medium text-foreground">{p.playerName}</p>
                  <p className="text-xs text-muted-foreground">{p.groupName}</p>
                </div>
                <p className="font-semibold text-red-500">{formatMoney(p.totalUnpaid, 'compact')}</p>
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
