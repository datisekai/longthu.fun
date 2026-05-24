// Thin fetch wrapper around the backend REST API.
//
// All requests:
//   * include `credentials: 'include'` so the lt_session cookie is sent
//   * set `Content-Type: application/json` on POST/PUT/PATCH
//   * parse RFC 7807 problem details into a typed ApiError on non-2xx
import type { RFC7807Problem } from '@/types/api';

const apiBaseURL = (import.meta.env?.RSBUILD_PUBLIC_API_BASE_URL as string | undefined) ?? 'http://localhost:8080';

export class ApiError extends Error {
  status: number;
  problem: RFC7807Problem;

  constructor(problem: RFC7807Problem) {
    super(problem.title || `HTTP ${problem.status}`);
    this.name = 'ApiError';
    this.status = problem.status;
    this.problem = problem;
  }

  /** Convenience: lookup a per-field validation error (RFC 7807 errors[] array). */
  fieldError(field: string): string | undefined {
    return this.problem.errors?.find((e) => e.field === field)?.message;
  }
}

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE';
  body?: unknown;
  signal?: AbortSignal;
}

/** Fire an API request and return the parsed JSON body. Throws ApiError on non-2xx. */
export async function apiRequest<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, signal } = opts;
  const url = path.startsWith('http') ? path : `${apiBaseURL}${path}`;

  const init: RequestInit = {
    method,
    credentials: 'include',
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    signal,
  };

  const res = await fetch(url, init);

  if (res.status === 204) {
    return undefined as T;
  }

  const text = await res.text();
  const parsed: unknown = text ? JSON.parse(text) : undefined;

  if (!res.ok) {
    const problem: RFC7807Problem =
      isProblem(parsed)
        ? parsed
        : { type: 'about:blank', title: res.statusText || `HTTP ${res.status}`, status: res.status };
    throw new ApiError(problem);
  }

  return parsed as T;
}

function isProblem(v: unknown): v is RFC7807Problem {
  return (
    typeof v === 'object' &&
    v !== null &&
    typeof (v as RFC7807Problem).title === 'string' &&
    typeof (v as RFC7807Problem).status === 'number'
  );
}
