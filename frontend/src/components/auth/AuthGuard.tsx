import * as React from 'react';
import { Navigate } from '@tanstack/react-router';
import { useAuthSession } from '@/hooks/useAuthSession';
import { vi } from '@/locales/vi';

interface AuthGuardProps {
  children: React.ReactNode;
}

export function AuthGuard({ children }: AuthGuardProps) {
  const { user, isLoading } = useAuthSession();

  if (isLoading) {
    return (
      <main className="flex min-h-screen items-center justify-center px-6 text-sm text-muted-foreground">
        {vi.actions.loading}
      </main>
    );
  }

  if (!user) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}
