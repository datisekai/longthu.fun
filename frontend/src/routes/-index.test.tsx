import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { vi } from '@/locales/vi';

// Smoke test: ensures Vitest + Testing Library + the Vietnamese locales module
// are wired correctly. Routing-aware tests for the actual `/` route arrive once
// per-route loaders + auth gating land in Stories 1.5+.
describe('Home greeting', () => {
  it('renders the Vietnamese greeting from the locales module', () => {
    render(<h1>{vi.home.greeting}</h1>);
    expect(screen.getByText('Longthu.fun đang chạy 🏸')).toBeInTheDocument();
  });

  it('exposes the home subtitle in Vietnamese', () => {
    expect(vi.home.subtitle).toMatch(/cầu lông|con vợ/);
  });
});
