import { createFileRoute } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { formatMoney } from '@/lib/money';
import { vi } from '@/locales/vi';
import { apiRequest } from '@/lib/api';

export const Route = createFileRoute('/pay/$intentCode')({
  component: PayPage,
});

interface PaymentIntentResponse {
  code: string;
  amount: number;
  status: string;
  transferContent: string;
  expiresAt: string;
  bankInfo: {
    bankName: string;
    bankCode: string;
    accountNumber: string;
    accountHolder: string;
  };
}

function StatusBadge({ status }: { status: string }) {
  if (status === 'matched') {
    return (
      <div className="rounded-md border border-green-500/40 bg-green-500/10 p-4 text-center">
        <p className="text-4xl">✅</p>
        <h2 className="mt-2 text-xl font-bold text-green-500">
          {vi.app.paymentReceived(formatMoney(0))}
        </h2>
      </div>
    );
  }
  if (status === 'expired') {
    return (
      <div className="rounded-md border border-destructive/40 bg-destructive/10 p-4 text-center">
        <h2 className="text-xl font-bold text-destructive">{vi.app.qrExpired}</h2>
        <a href="/" className="mt-2 text-sm text-primary hover:underline">
          {vi.app.createNewQR}
        </a>
      </div>
    );
  }
  return null;
}

function PayPage() {
  const { intentCode } = Route.useParams();

  const { data, isLoading, isError } = useQuery({
    queryKey: ['payment-intent', intentCode],
    queryFn: () => apiRequest<PaymentIntentResponse>(`/api/v1/payment-intents/${intentCode}`),
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
          Link thanh toán này không tồn tại hoặc đã hết hạn.
        </p>
      </main>
    );
  }

  const { status, amount, transferContent, bankInfo } = data;
  const maskedAccount = bankInfo.accountNumber.slice(-4).padStart(bankInfo.accountNumber.length, '✱');

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-md flex-col gap-6 px-4 py-8">
      {/* Status */}
      <StatusBadge status={status} />

      {/* Amount hero */}
      {status === 'pending' && (
        <div className="space-y-4 rounded-md border border-primary/40 bg-primary/10 p-4 text-center">
          <p className="text-sm text-muted-foreground">{vi.app.amountToTransfer}</p>
          <p className="text-4xl font-bold font-display text-primary">
            {formatMoney(amount)}
          </p>
        </div>
      )}

      {/* QR placeholder */}
      {status === 'pending' && (
        <div className="mx-auto flex size-60 items-center justify-center rounded-lg border-2 border-dashed border-border bg-white">
          <p className="text-muted-foreground">QR sẽ hiển thị ở đây</p>
        </div>
      )}

      {/* Bank info */}
      {status === 'pending' && (
        <div className="space-y-3 rounded-md border border-input bg-muted p-4">
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground">{vi.app.bankName}</p>
            <p className="font-medium text-foreground">{bankInfo.bankName}</p>
          </div>
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground">{vi.app.accountNumber}</p>
            <p className="font-mono text-foreground">{maskedAccount}</p>
          </div>
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground">{vi.app.accountHolder}</p>
            <p className="font-medium text-foreground">{bankInfo.accountHolder}</p>
          </div>
        </div>
      )}

      {/* Transfer content */}
      {status === 'pending' && (
        <div className="space-y-2 rounded-md border border-input bg-muted p-4">
          <p className="text-xs text-muted-foreground">{vi.app.transferContent}</p>
          <div className="flex items-center justify-between">
            <p className="font-mono text-lg font-bold text-foreground">{transferContent}</p>
            <button
              onClick={() => navigator.clipboard.writeText(transferContent)}
              className="rounded-md bg-primary px-3 py-1 text-sm font-medium text-primary-foreground hover:brightness-110"
            >
              {vi.app.copy}
            </button>
          </div>
        </div>
      )}

      {/* "Tôi đã chuyển" button */}
      {status === 'pending' && (
        <button
          onClick={async () => {
            await apiRequest(`/api/v1/payment-intents/${intentCode}/mark-transferred`, {
              method: 'POST',
            });
          }}
          className="flex h-12 w-full items-center justify-center rounded-md bg-secondary text-secondary-foreground font-bold hover:bg-muted"
        >
          {vi.app.iTransferred}
        </button>
      )}

      {/* Pending confirmation state */}
      {status === 'pending_confirmation' && (
        <div className="rounded-md border border-yellow-500/40 bg-yellow-500/10 p-4 text-center">
          <p className="text-2xl">🟡</p>
          <h2 className="mt-2 text-lg font-bold text-yellow-500">{vi.app.pendingConfirmTitle}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{vi.app.pendingConfirmDesc}</p>
        </div>
      )}
    </main>
  );
}
