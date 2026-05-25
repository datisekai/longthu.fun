import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { formatMoney } from '@/lib/money';
import { vi } from '@/locales/vi';
import { apiRequest } from '@/lib/api';

export const Route = createFileRoute('/p/$playerCode')({
  component: PlayerLedgerPage,
});

interface LedgerResponse {
  player: {
    id: number;
    displayName: string;
    publicCode: string;
  };
  currentCharge?: {
    sessionId: number;
    amount: number;
    status: string;
    sessionDate: string;
    sessionTitle?: string;
  };
  charges: Array<{
    id: number;
    sessionId: number;
    amount: number;
    status: string;
    paidAt?: string;
    sessionDate: string;
    sessionTitle?: string;
    groupName: string;
  }>;
  summary: {
    totalUnpaid: number;
    totalPaid: number;
    hasUnpaid: boolean;
  };
}

function StatusBadge({ status }: { status: string }) {
  const configs: Record<string, { dot: string; label: string; color: string }> = {
    unpaid: { dot: '🔴', label: 'chưa trả', color: 'text-red-500' },
    pending_confirmation: { dot: '🟡', label: 'chờ xác nhận', color: 'text-yellow-500' },
    suspected: { dot: '🟠', label: 'nghi khớp', color: 'text-orange-500' },
    paid: { dot: '✅', label: 'đã trả', color: 'text-green-500' },
    waived: { dot: '⚪', label: 'miễn', color: 'text-gray-400' },
  };
  const cfg = configs[status] ?? { dot: '⚪', label: status, color: 'text-gray-400' };
  return (
    <span className={`inline-flex items-center gap-1 text-xs ${cfg.color}`}>
      <span aria-hidden="true">{cfg.dot}</span>
      <span>{cfg.label}</span>
    </span>
  );
}

function PlayerLedgerPage() {
  const { playerCode } = Route.useParams();

  const { data, isLoading, isError } = useQuery({
    queryKey: ['player-ledger', playerCode],
    queryFn: () => apiRequest<LedgerResponse>(`/api/v1/player-ledger/${playerCode}`),
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

  const { player, currentCharge, charges, summary } = data;
  const unpaidCharges = charges.filter(c => c.status !== 'paid' && c.status !== 'waived');
  const paidCharges = charges.filter(c => c.status === 'paid');

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-md flex-col gap-6 px-4 py-8">
      {/* Hero */}
      <header className="space-y-1 text-center">
        <p className="text-sm text-muted-foreground">{vi.app.playerLedgerGreeting}</p>
        <h1 className="text-2xl font-bold font-display text-primary">
          {player.displayName}
        </h1>
      </header>

      {/* Current charge / action */}
      {summary.hasUnpaid && currentCharge ? (
        <section className="space-y-4 rounded-md border border-primary/40 bg-primary/10 p-4">
          <div className="text-center space-y-1">
            <p className="text-sm text-muted-foreground">{vi.app.totalOwed}</p>
            <p className="text-4xl font-bold font-display text-primary">
              {formatMoney(currentCharge.amount, 'compact')}
            </p>
            <StatusBadge status={currentCharge.status} />
          </div>
          <a
            href={`/pay/${playerCode}`}
            className="flex h-12 w-full items-center justify-center rounded-md bg-primary text-primary-foreground font-bold text-lg hover:brightness-110"
            style={{ boxShadow: '0 0 24px rgba(0,255,136,0.35)' }}
          >
            {vi.app.payNow(formatMoney(currentCharge.amount))}
          </a>
        </section>
      ) : (
        <section className="space-y-2 rounded-md border border-green-500/40 bg-green-500/10 p-4 text-center">
          <p className="text-2xl">✅</p>
          <h2 className="text-xl font-bold text-green-500">{vi.app.allPaid}</h2>
        </section>
      )}

      {/* Unpaid list */}
      {unpaidCharges.length > 0 && (
        <section className="space-y-2">
          <h2 className="text-sm font-semibold text-foreground">{vi.app.unpaidList}</h2>
          <ul className="space-y-2">
            {unpaidCharges.map((charge) => (
              <li
                key={charge.id}
                className="flex items-center justify-between rounded-md border border-input bg-muted px-4 py-3"
              >
                <div>
                  <p className="font-medium text-foreground">
                    {charge.sessionTitle ?? vi.app.billDefault}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {charge.sessionDate} · {charge.groupName}
                  </p>
                </div>
                <div className="text-right">
                  <p className="font-semibold text-primary">
                    {formatMoney(charge.amount, 'compact')}
                  </p>
                  <StatusBadge status={charge.status} />
                </div>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* Paid list */}
      {paidCharges.length > 0 && (
        <details className="space-y-2">
          <summary className="cursor-pointer text-sm font-semibold text-foreground hover:text-primary">
            {vi.app.paidList(paidCharges.length)}
          </summary>
          <ul className="mt-2 space-y-2">
            {paidCharges.map((charge) => (
              <li
                key={charge.id}
                className="flex items-center justify-between rounded-md border border-input bg-muted px-4 py-3 opacity-70"
              >
                <div>
                  <p className="font-medium text-foreground">
                    {charge.sessionTitle ?? vi.app.billDefault}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {charge.sessionDate} · {charge.groupName}
                  </p>
                </div>
                <div className="text-right">
                  <p className="font-semibold text-green-500">
                    {formatMoney(charge.amount, 'compact')}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {charge.paidAt ? `đã trả ${charge.paidAt}` : 'đã trả'}
                  </p>
                </div>
              </li>
            ))}
          </ul>
        </details>
      )}
    </main>
  );
}
