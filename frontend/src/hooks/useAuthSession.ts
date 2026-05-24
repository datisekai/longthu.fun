import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiRequest, ApiError } from '@/lib/api';
import type { PublicUser } from '@/types/api';

const SESSION_KEY = ['auth', 'me'] as const;

interface RegisterParams {
  email: string;
  password: string;
  displayName: string;
}

interface LoginParams {
  email: string;
  password: string;
}

/**
 * Reads the current session from /api/v1/auth/me.
 * Returns `user = null` when not authenticated (the API 401 is caught).
 */
export function useAuthSession() {
  const queryClient = useQueryClient();

  const { data, isLoading, isError } = useQuery<PublicUser | null>({
    queryKey: SESSION_KEY,
    queryFn: async () => {
      try {
        return await apiRequest<PublicUser>('/api/v1/auth/me');
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) return null;
        throw err;
      }
    },
    retry: false,
    staleTime: 60_000,
  });

  const registerMutation = useMutation({
    mutationFn: (params: RegisterParams) =>
      apiRequest<PublicUser>('/api/v1/auth/register', { method: 'POST', body: params }),
    onSuccess: (user) => {
      queryClient.setQueryData(SESSION_KEY, user);
    },
  });

  const loginMutation = useMutation({
    mutationFn: (params: LoginParams) =>
      apiRequest<PublicUser>('/api/v1/auth/login', { method: 'POST', body: params }),
    onSuccess: (user) => {
      queryClient.setQueryData(SESSION_KEY, user);
    },
  });

  const logoutMutation = useMutation({
    mutationFn: () => apiRequest<{ ok: boolean }>('/api/v1/auth/logout', { method: 'POST' }),
    onSuccess: () => {
      queryClient.setQueryData(SESSION_KEY, null);
    },
  });

  return {
    user: data ?? null,
    isLoading,
    isError,
    register: registerMutation,
    login: loginMutation,
    logout: logoutMutation,
  };
}
