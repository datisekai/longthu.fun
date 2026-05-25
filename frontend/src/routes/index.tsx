import { createFileRoute } from '@tanstack/react-router';
import { useAuthSession } from '@/hooks/useAuthSession';
import { vi } from '@/locales/vi';
import { Button } from '@/components/ui/button';

export const Route = createFileRoute('/')({
  component: HomePage,
});

function HomePage() {
  const { user, isLoading } = useAuthSession();

  if (isLoading) {
    return (
      <main className="flex min-h-screen items-center justify-center">
        <div className="text-4xl">🏸</div>
      </main>
    );
  }

  if (user) {
    window.location.href = '/dashboard';
    return (
      <main className="flex min-h-screen items-center justify-center">
        <div className="text-4xl">⏳</div>
      </main>
    );
  }

  return <LandingPage />;
}

function LandingPage() {
  return (
    <div className="min-h-screen bg-background">
      {/* Navbar */}
      <header className="fixed top-0 left-0 right-0 z-50 border-b bg-background/80 backdrop-blur-sm">
        <div className="mx-auto flex h-16 max-w-5xl items-center justify-between px-4">
          <div className="flex items-center gap-2">
            <span className="text-2xl">🏸</span>
            <span className="text-xl font-bold text-primary">{vi.app.name}</span>
          </div>
          <div className="flex items-center gap-3">
            <Button variant="ghost" size="sm" onClick={() => window.location.href = '/login'}>
              Đăng nhập
            </Button>
            <Button variant="primary" size="sm" onClick={() => window.location.href = '/register'}>
              Bắt đầu
            </Button>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="pt-32 pb-20 text-center">
        <div className="mx-auto max-w-4xl px-4">
          <div className="mb-6 inline-block rounded-full border border-primary/30 bg-primary/10 px-4 py-1 text-sm text-primary">
            🎉 Miễn phí cho nhóm cầu lông của bạn
          </div>
          <h1 className="mb-6 text-5xl font-bold leading-tight">
            Chia bill cầu lông
            <br />
            <span className="text-primary">Không cần nhắc nhở</span>
          </h1>
          <p className="mx-auto mb-10 max-w-2xl text-xl text-muted-foreground">
            Host tạo buổi chơi → chia tiền tự động → Player chuyển tiền qua QR.
            Mọi thứ tự động, không lo thiếu tiền.
          </p>
          <div className="flex flex-col items-center gap-4 sm:flex-row sm:justify-center">
            <Button variant="primary" size="lg" className="text-lg px-8" onClick={() => window.location.href = '/register'}>
              Tạo tài khoản miễn phí
            </Button>
            <Button variant="outline" size="lg" className="text-lg px-8" onClick={() => window.location.href = '/login'}>
              Đã có tài khoản?
            </Button>
          </div>
        </div>
      </section>

      {/* Demo Mockup */}
      <section className="bg-muted/50 py-16">
        <div className="mx-auto max-w-5xl px-4">
          <div className="mx-auto max-w-sm rounded-2xl border bg-card p-6 shadow-2xl">
            <div className="mb-4 flex items-center gap-3 border-b pb-4">
              <div className="h-10 w-10 rounded-full bg-primary/20" />
              <div>
                <div className="font-semibold">Nhóm Cầu CN</div>
                <div className="text-sm text-muted-foreground">5 members · 1 buổi chưa thanh toán</div>
              </div>
            </div>
            <div className="space-y-3">
              <div className="flex items-center justify-between rounded-lg bg-muted p-3">
                <div className="flex items-center gap-3">
                  <div className="h-8 w-8 rounded-full bg-secondary" />
                  <span>Anh Tuấn</span>
                </div>
                <span className="font-semibold text-primary">150K</span>
              </div>
              <div className="flex items-center justify-between rounded-lg bg-muted p-3">
                <div className="flex items-center gap-3">
                  <div className="h-8 w-8 rounded-full bg-secondary" />
                  <span>Chị Linh</span>
                </div>
                <span className="font-semibold text-primary">150K</span>
              </div>
              <div className="flex items-center justify-between rounded-lg bg-muted p-3">
                <div className="flex items-center gap-3">
                  <div className="h-8 w-8 rounded-full bg-secondary" />
                  <span>Bạn Minh</span>
                </div>
                <span className="font-semibold text-green-500">✓ Đã thanh toán</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="py-20">
        <div className="mx-auto max-w-5xl px-4">
          <h2 className="mb-12 text-center text-3xl font-bold">Tại sao chọn {vi.app.name}?</h2>
          <div className="grid gap-8 md:grid-cols-3">
            <FeatureCard
              icon="⚡"
              title="Nhanh chóng"
              description="Tạo buổi chơi, chia tiền chỉ trong 30 giây. Không cần Excel, không cần tính tay."
            />
            <FeatureCard
              icon="🔗"
              title="Chia sẻ dễ dàng"
              description="Gửi 1 link cho cả nhóm. Mỗi người tự xem bill và chuyển tiền riêng."
            />
            <FeatureCard
              icon="✅"
              title="Tự động đối soát"
              description="Khi có chuyển khoản, hệ thống tự nhận diện và đánh dấu đã trả."
            />
            <FeatureCard
              icon="📱"
              title="Mọi thiết bị"
              description="Hoạt động tốt trên điện thoại, tablet, và máy tính."
            />
            <FeatureCard
              icon="🔒"
              title="Bảo mật"
              description="Thông tin thanh toán được mã hóa. Không lưu số dư tài khoản."
            />
            <FeatureCard
              icon="💰"
              title="Miễn phí"
              description="Sử dụng hoàn toàn miễn phí. Không phí ẩn, không quảng cáo."
            />
          </div>
        </div>
      </section>

      {/* How it works */}
      <section className="bg-muted/50 py-20">
        <div className="mx-auto max-w-5xl px-4">
          <h2 className="mb-12 text-center text-3xl font-bold">Cách hoạt động</h2>
          <div className="grid gap-8 md:grid-cols-4">
            <StepCard number={1} title="Host tạo nhóm" description="Tạo group cho buổi chơi, thêm tài khoản nhận tiền" />
            <StepCard number={2} title="Thêm người chơi" description="Nhập tên và số điện thoại của các thành viên" />
            <StepCard number={3} title="Chia tiền" description="Nhập chi phí, hệ thống tự chia đều hoặc tùy chỉnh" />
            <StepCard number={4} title="Chia sẻ link" description="Gửi link cho nhóm, mỗi người tự chuyển tiền qua QR" />
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="py-20">
        <div className="mx-auto max-w-2xl px-4 text-center">
          <h2 className="mb-4 text-3xl font-bold">Sẵn sàng chia bill không lo thiếu?</h2>
          <p className="mb-8 text-muted-foreground">
            Hàng trăm nhóm cầu lông đã sử dụng. Đăng ký ngay và bắt đầu tổ chức buổi chơi tiếp theo.
          </p>
          <Button variant="primary" size="lg" className="text-lg px-10" onClick={() => window.location.href = '/register'}>
            Bắt đầu miễn phí
          </Button>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t py-8">
        <div className="mx-auto max-w-5xl px-4 text-center text-sm text-muted-foreground">
          <div className="mb-2 flex items-center justify-center gap-2">
            <span className="text-xl">🏸</span>
            <span className="font-semibold text-foreground">{vi.app.name}</span>
          </div>
          <p>Miễn phí cho nhóm cầu lông của bạn</p>
        </div>
      </footer>
    </div>
  );
}

function FeatureCard({ icon, title, description }: { icon: string; title: string; description: string }) {
  return (
    <div className="rounded-xl border bg-card p-6">
      <div className="mb-4 text-4xl">{icon}</div>
      <h3 className="mb-2 text-lg font-semibold">{title}</h3>
      <p className="text-sm text-muted-foreground">{description}</p>
    </div>
  );
}

function StepCard({ number, title, description }: { number: number; title: string; description: string }) {
  return (
    <div className="text-center">
      <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-primary text-lg font-bold text-primary-foreground">
        {number}
      </div>
      <h3 className="mb-2 font-semibold">{title}</h3>
      <p className="text-sm text-muted-foreground">{description}</p>
    </div>
  );
}
