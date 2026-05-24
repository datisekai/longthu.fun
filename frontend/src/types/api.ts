// Backend response types. Keep in sync with backend/internal/auth/service.go
// and any other hand-shipped Go response shape. There's no codegen pipeline
// in MVP (architecture §API documentation) so this is the manual contract.

export interface PublicUser {
  id: number;
  email: string;
  displayName: string;
  tier: 'free' | 'pro' | 'pro_plus';
}

export interface BankAccount {
  id: number;
  bankName: string;
  bankCode: 'MBBANK' | 'VCB' | 'TPB';
  accountNumber: string;
  accountHolderName: string;
  isDefault: boolean;
}

// Added in Story 1.8.
export interface Group {
  id: number;
  name: string;
  privacyMode: 'public' | 'private_leaning';
  autoDetectEnabled: boolean;
  defaultBankAccountId?: number;
}

// Added in Story 1.9.
export interface Player {
  id: number;
  groupId: number;
  displayName: string;
  publicCode: string;
  isActive: boolean;
}

// Added in Story 1.10.
export type CostItemType = 'court' | 'shuttle' | 'water' | 'other' | 'discount';

export interface Session {
  id: number;
  groupId: number;
  date: string; // YYYY-MM-DD
  title?: string;
  location?: string;
  status: 'draft' | 'finalized' | 'archived';
  totalCost: number;
  shareCode?: string;
  finalizedAt?: string;
}

export interface CostItem {
  id: number;
  sessionId: number;
  type: CostItemType;
  label: string;
  amount: number;
  isIncludedInSplit: boolean;
}

export interface SessionDraftSnapshot {
  session: Session;
  costItems: CostItem[];
  participants: Array<{ playerId: number }>;
}

export interface RFC7807Problem {
  type: string;
  title: string;
  status: number;
  detail?: string;
  instance?: string;
  errors?: Array<{ field: string; message: string }>;
}
