import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { formatMoney } from '@/lib/money';
import { apiRequest } from '@/lib/api';
import { Button } from '@/components/ui/button';

export const Route = createFileRoute('/dashboard/unmatched')({
  component: UnmatchedPage,
});

interface UnmatchedPayment {
  id: number;
  amount: number;
  bankDescription: string;
  receivedAt: string;
}

function UnmatchedPage() {
  const { data: payments, isLoading } = useQuery({
    queryKey: ['dashboard', 'unmatched'],
    queryFn: () => apiRequest<{ payments: UnmatchedPayment[] }>('/api/v1/dashboard/unmatched'),
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

  const paymentList = payments?.payments ?? [];

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-md flex-col gap-6 px-4 py-8">
      <header className="flex items-center gap-3">
        <a href="/dashboard" className="text-muted-foreground">←</a>
        <h1 className="text-xl font-bold">Chưa khớp</h1>
      </header>

      {paymentList.length === 0 ? (
        <div className="rounded-md border border-primary/40 bg-primary/10 p-8 text-center">
          <p className="text-4xl">✅</p>
          <p className="mt-2 text-muted-foreground">Không có payment nào chưa khớp</p>
        </div>
      ) : (
        <ul className="space-y-3">
          {paymentList.map((payment) => (
            <li key={payment.id} className="rounded-md border border-orange-500/30 bg-orange-500/10 p-4">
              <div className="flex items-start justify-between">
                <div>
                  <p className="text-2xl font-bold text-orange-500">{formatMoney(payment.amount)}</p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {payment.receivedAt}
                  </p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    Nội dung: {payment.bankDescription}
                  </p>
                </div>
              </div>
              <div className="mt-3">
                <Button
                  variant="outline"
                  size="sm"
                  className="w-full"
                  onClick={() => {
                    // TODO: Open modal to select player and charges
                    alert('Tính năng khớp thủ công - cần chọn người và các khoản');
                  }}
                >
                  Khớp thủ công
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
