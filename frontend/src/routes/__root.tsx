import { createRootRoute, Outlet } from '@tanstack/react-router';
import { ToastContainer } from '@/components/ui/toast';

export const Route = createRootRoute({
  component: RootLayout,
});

function RootLayout() {
  return (
    <ToastContainer>
      <div className="min-h-screen bg-background text-foreground font-sans">
        <Outlet />
      </div>
    </ToastContainer>
  );
}
