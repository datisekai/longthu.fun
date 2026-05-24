// Backend response types. Keep in sync with backend/internal/auth/service.go
// and any other hand-shipped Go response shape. There's no codegen pipeline
// in MVP (architecture §API documentation) so this is the manual contract.

export interface PublicUser {
  id: number;
  email: string;
  displayName: string;
  tier: 'free' | 'pro' | 'pro_plus';
}

export interface RFC7807Problem {
  type: string;
  title: string;
  status: number;
  detail?: string;
  instance?: string;
  errors?: Array<{ field: string; message: string }>;
}
