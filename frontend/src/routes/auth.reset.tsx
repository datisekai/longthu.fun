import { Link, createFileRoute } from '@tanstack/react-router';
import { Button } from '@/components/ui/button';
import { vi } from '@/locales/vi';

export const Route = createFileRoute('/auth/reset')({
  component: ResetPasswordPage,
});

export function ResetPasswordPage() {
  return (
    <main className="mx-auto flex min-h-screen w-full max-w-md flex-col justify-center gap-6 px-6 py-10">
      <header className="space-y-2">
        <h1 className="text-3xl font-bold text-foreground">{vi.auth.reset.title}</h1>
        <p className="text-sm leading-6 text-muted-foreground">{vi.auth.reset.body}</p>
      </header>

      <div className="flex flex-col gap-3">
        <Button asChild size="lg" className="w-full">
          <a href={vi.auth.reset.telegramHref} target="_blank" rel="noreferrer">
            {vi.auth.reset.telegramLabel}
          </a>
        </Button>
        <Button asChild variant="ghost" size="lg" className="w-full">
          <Link to="/login">{vi.auth.reset.goLogin}</Link>
        </Button>
      </div>
    </main>
  );
}
