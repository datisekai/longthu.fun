import { createFileRoute } from '@tanstack/react-router';
import { vi } from '@/locales/vi';

export const Route = createFileRoute('/')({
  component: HomePage,
});

function HomePage() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-3 px-6 text-center">
      <h1
        className="text-3xl font-bold text-primary"
        style={{ textShadow: '0 0 24px rgba(0, 255, 136, 0.35)' }}
      >
        {vi.home.greeting}
      </h1>
      <p className="text-sm text-muted-foreground">{vi.home.subtitle}</p>
    </main>
  );
}
