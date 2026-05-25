import { useState, useEffect, useRef } from 'react';
import { QRCodeSVG } from 'qrcode.react';
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

interface VietQRData {
  bankCode: string;
  accountNumber: string;
  amount: string;
  content: string;
}

// VietQR EMVCo format for Vietnam domestic transfers
function buildVietQR(data: VietQRData): string {
  const bankBinMap: Record<string, string> = {
    mbbank: '970422',
    mb: '970422',
    vietcombank: '970436',
    vcb: '970436',
    techcombank: '970407',
    tcb: '970407',
    tpbank: '970452',
    tpb: '970452',
    acb: '970416',
    bidv: '970418',
    ocb: '970448',
    kienlongbank: '970452',
  };

  const bin = bankBinMap[data.bankCode.toLowerCase()] || data.bankCode;
  const cleanAmount = data.amount.replace(/\D/g, '');
  const content = data.content || 'PAY';

  // VietQR EMVCo format
  let raw = '000201010211';
  raw += '38580011';
  raw += 'A000000727012000015';
  raw += '0201QRIBOLTT';
  raw += `0201${bin}`;
  raw += `0216${data.accountNumber}`;
  raw += `0301${cleanAmount || '0'}`;
  raw += '52045953580';
  raw += '5802VN';
  raw += '6008VND';
  raw += `0801${content}`;

  // Add CRC16
  const crc16 = calculateCRC16(raw + '6304');
  return raw + '6304' + crc16.toString(16).toUpperCase().padStart(4, '0');
}

function calculateCRC16(data: string): number {
  let crc = 0xffff;
  const polynomial = 0x1021;
  for (let i = 0; i < data.length; i++) {
    crc ^= data.charCodeAt(i) << 8;
    for (let j = 0; j < 8; j++) {
      if (crc & 0x8000) {
        crc = ((crc << 1) ^ polynomial) & 0xffff;
      } else {
        crc = (crc << 1) & 0xffff;
      }
    }
  }
  return crc & 0xffff;
}

function StatusBadge({ status, amount }: { status: string; amount: number }) {
  if (status === 'matched') {
    return (
      <div className="rounded-md border border-green-500/40 bg-green-500/10 p-4 text-center">
        <p className="text-4xl">✅</p>
        <h2 className="mt-2 text-xl font-bold text-green-500">
          {vi.app.paymentReceived(formatMoney(amount))}
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

function Confetti({ show }: { show: boolean }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    if (!show || !canvasRef.current) return;

    const canvas = canvasRef.current;
    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const colors = ['#00FF88', '#C084FC', '#FBBF24', '#F472B6', '#60A5FA'];
    const particles = Array.from({ length: 100 }, () => ({
      x: canvas.width / 2,
      y: canvas.height / 2,
      vx: (Math.random() - 0.5) * 20,
      vy: (Math.random() - 0.5) * 20 - 5,
      color: colors[Math.floor(Math.random() * colors.length)],
      life: 1,
    }));

    let frame = 0;
    const maxFrames = 60;

    const draw = () => {
      if (frame >= maxFrames) {
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        return;
      }
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      particles.forEach((p) => {
        p.x += p.vx;
        p.y += p.vy;
        p.vy += 0.5;
        p.life -= 1 / maxFrames;
        if (p.life > 0) {
          ctx.globalAlpha = p.life;
          ctx.fillStyle = p.color;
          ctx.beginPath();
          ctx.arc(p.x, p.y, 4, 0, Math.PI * 2);
          ctx.fill();
        }
      });
      frame++;
      requestAnimationFrame(draw);
    };

    draw();
  }, [show]);

  if (!show) return null;
  return (
    <canvas
      ref={canvasRef}
      className="pointer-events-none fixed inset-0 z-50"
      aria-hidden="true"
    />
  );
}

function PayPage() {
  const { intentCode } = Route.useParams();
  const [showConfetti, setShowConfetti] = useState(false);
  const [prevStatus, setPrevStatus] = useState<string>('');

  const { data, isLoading, isError } = useQuery({
    queryKey: ['payment-intent', intentCode],
    queryFn: () => apiRequest<PaymentIntentResponse>(`/api/v1/payment-intents/${intentCode}`),
    refetchInterval: 5000,
    refetchIntervalInBackground: false,
  });

  useEffect(() => {
    if (data?.status === 'matched' && prevStatus !== 'matched') {
      setShowConfetti(true);
      setTimeout(() => setShowConfetti(false), 3000);
    }
    setPrevStatus(data?.status || '');
  }, [data?.status, prevStatus]);

  if (isLoading) {
    return (
      <main className="mx-auto flex min-h-screen w-full max-w-md flex-col items-center justify-center gap-4 px-4 py-8">
        <div className="text-4xl animate-spin">🏸</div>
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
      <Confetti show={showConfetti} />

      <StatusBadge status={status} amount={amount} />

      {status === 'pending' && (
        <div className="space-y-4 rounded-md border border-primary/40 bg-primary/10 p-4 text-center">
          <p className="text-sm text-muted-foreground">{vi.app.amountToTransfer}</p>
          <p className="text-4xl font-bold font-display text-primary">
            {formatMoney(amount)}
          </p>
        </div>
      )}

      {status === 'pending' && (
        <div className="mx-auto overflow-hidden rounded-lg border-4 border-white bg-white p-4">
          <QRCodeSVG
            value={buildVietQR({
              bankCode: bankInfo.bankCode,
              accountNumber: bankInfo.accountNumber,
              amount: amount.toString(),
              content: transferContent,
            })}
            size={220}
            level="M"
            includeMargin={false}
          />
        </div>
      )}

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
