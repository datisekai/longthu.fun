import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { formatMoney } from '@/lib/money';
import { vi } from '@/locales/vi';
import { apiRequest } from '@/lib/api';

export const Route = createFileRoute('/g/$shareCode')({
  component: GroupBillPage,
});

interface GroupBillResponse {
  session: {
    id: number;
    date: string;
    title?: string;
    totalCost: number;
  };
  players: PlayerRow[];
  summary: {
    totalCost: number;
    totalPaid: number;
    totalUnpaid: number;
  };
  privacyMode: string;
}

interface PlayerRow {
  playerId: number;
  displayName: string;
  publicCode: string;
  chargeAmount: number;
  chargeStatus: string;
  crossDebt?: number;
}

function StatusDot({ status }: { status: string }) {
  const configs: Record<string, { dot: string; label: string }> = {
    unpaid: { dot: '🔴', label: 'chưa trả' },
    pending_confirmation: { dot: '🟡', label: 'chờ xác nhận' },
    suspected: { dot: '🟠', label: 'nghi khớp' },
    paid: { dot: '✅', label: 'đã trả' },
    waived: { dot: '⚪', label: 'miễn' },
  };
  const cfg = configs[status] ?? { dot: '⚪', label: status };
  return (
    <span>
      <span aria-hidden="true">{cfg.dot}</span>{' '}
      <span className="text-xs">{cfg.label}</span>
    </span>
  );
}

function GroupBillPage() {
  const { shareCode } = Route.useParams();
  
  const { data, isLoading, isError } = useQuery({
    queryKey: ['group-bill', shareCode],
    queryFn: () => apiRequest<GroupBillResponse>(`/api/v1/group-bill/${shareCode}`),
    refetchInterval: 5000,
    refetchIntervalInBackground: false,
  });

  if (isLoading) {
    return (
      <main className="mx-auto flex min-h-screen w-full max-w-md flex-col items-center justify-center gap-4 px-4 py-8">
        <div className="text-4xl">🏸</div>
        <p className="text-muted-foreground">Đang tải...</p>
      </main>
    );
  }

  if (isError || !data) {
    return (
      <main className="mx-auto flex min-h-screen w-full max-w-md flex-col items-center justify-center gap-4 px-4 py-8">
        <div className="text-6xl">🏸</div>
        <h1 className="text-2xl font-bold text-foreground">Link không hợp lệ</h1>
        <p className="text-center text-muted-foreground">
          Link này không tồn tại hoặc đã hết hạn.
        </p>
      </main>
    );
  }

  const { session, players, summary } = data;

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-md flex-col gap-6 px-4 py-8">
      {/* Header */}
      <header className="space-y-1 text-center">
        <p className="text-sm text-muted-foreground">{session.date}</p>
        <h1 className="text-2xl font-bold font-display text-foreground">
          {session.title ?? vi.app.groupBillDefault}
        </h1>
      </header>

      {/* Summary hero */}
      <div className="rounded-md border border-primary/40 bg-primary/10 p-4 text-center">
        <p className="text-sm text-muted-foreground">Tổng cần trả</p>
        <p className="text-3xl font-bold font-display text-primary">
          {formatMoney(summary.totalCost, 'compact')}
        </p>
        <div className="mt-2 flex justify-center gap-4 text-sm">
          <span className="text-green-500">
            ✅ {formatMoney(summary.totalPaid, 'compact')}
          </span>
          <span className="text-red-500">
            🔴 {formatMoney(summary.totalUnpaid, 'compact')}
          </span>
        </div>
      </div>

      {/* Player list */}
      <section className="space-y-2">
        <h2 className="text-sm font-semibold text-foreground">{vi.app.playerList}</h2>
        <ul className="space-y-2">
          {players.map((player) => (
            <li key={player.playerId}>
              <a
                href={`/p/${player.publicCode}`}
                className="flex items-center justify-between rounded-md border border-input bg-muted px-4 py-3 text-foreground transition-colors hover:bg-muted/80"
              >
                <div className="flex items-center gap-2">
                  <span className="font-medium">{player.displayName}</span>
                  <StatusDot status={player.chargeStatus} />
                </div>
                <div className="text-right">
                  <p className="font-semibold text-primary">
                    {formatMoney(player.chargeAmount, 'compact')}
                  </p>
                  {player.crossDebt !== undefined && player.crossDebt > 0 && data.privacyMode === 'public' && (
                    <p className="text-xs text-muted-foreground">
                      còn nợ {formatMoney(player.crossDebt, 'compact')}
                    </p>
                  )}
                </div>
              </a>
            </li>
          ))}
        </ul>
      </section>
    </main>
  );
}
