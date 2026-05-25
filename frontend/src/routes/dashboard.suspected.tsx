import { createFileRoute } from '@tanstack/react-router';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { formatMoney } from '@/lib/money';
import { apiRequest } from '@/lib/api';
import { Button } from '@/components/ui/button';

export const Route = createFileRoute('/dashboard/suspected')({
  component: SuspectedPage,
});

interface SuspectedPayment {
  id: number;
  amount: number;
  bankDescription: string;
  receivedAt: string;
  intentCode?: string;
  intentAmount?: number;
  playerName?: string;
}

function SuspectedPage() {
  const queryClient = useQueryClient();

  const { data: payments, isLoading } = useQuery({
    queryKey: ['dashboard', 'suspected'],
    queryFn: () => apiRequest<{ payments: SuspectedPayment[] }>('/api/v1/dashboard/suspected'),
    refetchInterval: 30000,
  });

  const confirmMutation = useMutation({
    mutationFn: (paymentId: number) =>
      apiRequest(`/api/v1/payments/${paymentId}/confirm`, { method: 'POST' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dashboard', 'suspected'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard'] });
    },
  });

  const rejectMutation = useMutation({
    mutationFn: (paymentId: number) =>
      apiRequest(`/api/v1/payments/${paymentId}/reject`, { method: 'POST' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dashboard', 'suspected'] });
    },
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
        <h1 className="text-xl font-bold">Cần xác nhận</h1>
      </header>

      {paymentList.length === 0 ? (
        <div className="rounded-md border border-primary/40 bg-primary/10 p-8 text-center">
          <p className="text-4xl">✅</p>
          <p className="mt-2 text-muted-foreground">Không có payment nào cần xác nhận</p>
        </div>
      ) : (
        <ul className="space-y-3">
          {paymentList.map((payment) => (
            <li key={payment.id} className="rounded-md border border-yellow-500/30 bg-yellow-500/10 p-4">
              <div className="flex items-start justify-between">
                <div>
                  <p className="text-2xl font-bold text-yellow-500">{formatMoney(payment.amount)}</p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {payment.receivedAt} · {payment.bankDescription}
                  </p>
                  {payment.playerName && (
                    <p className="mt-1 text-sm text-foreground">
                      Người được đề xuất: {payment.playerName}
                    </p>
                  )}
                  {payment.intentCode && (
                    <p className="text-xs text-muted-foreground">
                      Intent: {payment.intentCode} ({formatMoney(payment.intentAmount ?? 0)})
                    </p>
                  )}
                </div>
              </div>
              <div className="mt-3 flex gap-2">
                <Button
                  variant="primary"
                  size="sm"
                  className="flex-1"
                  disabled={confirmMutation.isPending}
                  onClick={() => confirmMutation.mutate(payment.id)}
                >
                  Xác nhận khớp
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="flex-1"
                  disabled={rejectMutation.isPending}
                  onClick={() => rejectMutation.mutate(payment.id)}
                >
                  Không khớp
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
