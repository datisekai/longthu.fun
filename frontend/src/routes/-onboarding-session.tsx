import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useMemo, useState } from 'react';
import { Button } from '@/components/ui/button';
import { ApiError, apiRequest } from '@/lib/api';
import { formatMoney } from '@/lib/money';
import { vi } from '@/locales/vi';
import type { CostItem, CostItemType, FinalizeResponse, Player, Session } from '@/types/api';

const COST_TYPES: CostItemType[] = ['court', 'shuttle', 'water', 'other', 'discount'];

/**
 * Returns YYYY-MM-DD for today's date in Asia/Ho_Chi_Minh local time.
 * Using a fixed offset (+07:00 has no DST) is safe — Vietnam has had no
 * daylight-saving transitions since 1976.
 */
export function todayInVN(now: Date = new Date()): string {
  const vn = new Date(now.getTime() + 7 * 60 * 60_000);
  const y = vn.getUTCFullYear();
  const m = String(vn.getUTCMonth() + 1).padStart(2, '0');
  const d = String(vn.getUTCDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

/**
 * Split preview: returns (splittableTotal, perHead) rounded to nearest VND.
 * Items with isIncludedInSplit=false are excluded from the splittable total.
 */
export interface SplitPreview {
  total: number; // sum of ALL items (display)
  splittable: number; // sum of items with isIncludedInSplit=true
  participantCount: number;
  perHead: number; // rounded to nearest VND
}

export function computeSplit(items: CostItem[], participantCount: number): SplitPreview {
  let total = 0;
  let splittable = 0;
  for (const it of items) {
    total += it.amount;
    if (it.isIncludedInSplit) splittable += it.amount;
  }
  const perHead =
    participantCount > 0 ? Math.round(splittable / participantCount) : 0;
  return { total, splittable, participantCount, perHead };
}

interface SessionDraftStepProps {
  groupId: number;
  onSaved?: (session: Session) => void;
  onComplete?: (result: FinalizeResponse) => void;
}

export function SessionDraftStep({ groupId, onSaved, onComplete }: SessionDraftStepProps) {
  const qc = useQueryClient();
  const [date, setDate] = useState(todayInVN());
  const [title, setTitle] = useState('');
  const [location, setLocation] = useState('');
  const [costItems, setCostItems] = useState<CostItem[]>([]);
  const [participantIDs, setParticipantIDs] = useState<number[]>([]);
  const [sessionId, setSessionId] = useState<number | null>(null);
  const [finalizeResult, setFinalizeResult] = useState<FinalizeResponse | null>(null);
  const [copiedType, setCopiedType] = useState<'link' | 'message' | null>(null);
  const [pendingItem, setPendingItem] = useState<{ type: CostItemType; label: string; amount: string }>({
    type: 'court',
    label: '',
    amount: '',
  });
  const [pendingError, setPendingError] = useState<string | null>(null);

  // Load active roster for the participant picker (AC5).
  // Pre-populate participantIDs so the Save button is immediately enabled.
  const playersQuery = useQuery({
    queryKey: ['groups', groupId, 'players'],
    queryFn: () =>
      apiRequest<{ players: Player[] }>(`/api/v1/groups/${groupId}/players`).then((r) => r.players),
  });

  // Once players load, pre-select everyone so the user doesn't have to tick manually.
  useEffect(() => {
    if (playersQuery.data && participantIDs.length === 0) {
      setParticipantIDs(playersQuery.data.map((p) => p.id));
    }
  }, [playersQuery.data]); // eslint-disable-line react-hooks/exhaustive-deps

  // Mutations.
  const createDraft = useMutation({
    mutationFn: (input: { date: string; title: string; location: string }) =>
      apiRequest<Session>(`/api/v1/groups/${groupId}/sessions`, { method: 'POST', body: input }),
  });

  const addItem = useMutation({
    mutationFn: (vars: { sessionId: number; item: { type: CostItemType; label: string; amount: number } }) =>
      apiRequest<CostItem>(`/api/v1/sessions/${vars.sessionId}/cost-items`, {
        method: 'POST',
        body: { ...vars.item, isIncludedInSplit: true },
      }),
  });

  const removeItem = useMutation({
    mutationFn: (vars: { sessionId: number; itemId: number }) =>
      apiRequest<void>(`/api/v1/sessions/${vars.sessionId}/cost-items/${vars.itemId}`, {
        method: 'DELETE',
      }),
  });

  const setParticipantsMut = useMutation({
    mutationFn: (vars: { sessionId: number; playerIds: number[] }) =>
      apiRequest<{ participants: Array<{ playerId: number }> }>(
        `/api/v1/sessions/${vars.sessionId}/participants`,
        { method: 'PUT', body: { playerIds: vars.playerIds } },
      ),
  });

  const finalize = useMutation({
    mutationFn: (vars: { sessionId: number }) =>
      apiRequest<FinalizeResponse>(`/api/v1/sessions/${vars.sessionId}/finalize`, { method: 'POST' }),
  });

  const preview = useMemo(() => computeSplit(costItems, participantIDs.length), [costItems, participantIDs]);
  const canSave =
    date.length === 10 && costItems.length > 0 && participantIDs.length > 0;

  async function ensureDraft(): Promise<number> {
    if (sessionId) return sessionId;
    const created = await createDraft.mutateAsync({ date, title, location });
    setSessionId(created.id);
    return created.id;
  }

  async function handleAddItem() {
    setPendingError(null);
    const amount = Number.parseInt(pendingItem.amount.replace(/[^\d-]/g, ''), 10);
    if (!Number.isFinite(amount) || amount === 0) {
      setPendingError(vi.onboarding.session.errors.amountInvalid);
      return;
    }
    const label = pendingItem.label.trim();
    if (!label) {
      setPendingError(vi.onboarding.session.errors.labelRequired);
      return;
    }
    const sid = await ensureDraft();
    const item = await addItem.mutateAsync({
      sessionId: sid,
      item: { type: pendingItem.type, label, amount },
    });
    setCostItems((prev) => [...prev, item]);
    setPendingItem({ type: 'court', label: '', amount: '' });
  }

  async function handleRemoveItem(itemId: number) {
    if (!sessionId) return;
    await removeItem.mutateAsync({ sessionId, itemId });
    setCostItems((prev) => prev.filter((it) => it.id !== itemId));
  }

  async function handleToggleParticipant(playerId: number, checked: boolean) {
    const next = checked
      ? [...participantIDs, playerId]
      : participantIDs.filter((id) => id !== playerId);
    setParticipantIDs(next);
    if (sessionId && next.length > 0) {
      await setParticipantsMut.mutateAsync({ sessionId, playerIds: next });
    }
  }

  async function handleSave() {
    const sid = await ensureDraft();
    if (participantIDs.length > 0) {
      await setParticipantsMut.mutateAsync({ sessionId: sid, playerIds: participantIDs });
    }
    qc.invalidateQueries({ queryKey: ['sessions', sid] });
    onSaved?.({
      id: sid,
      groupId,
      date,
      title: title || undefined,
      location: location || undefined,
      status: 'draft',
      totalCost: preview.total,
    });
  }

  async function handleFinalize() {
    const sid = await ensureDraft();
    if (participantIDs.length > 0) {
      await setParticipantsMut.mutateAsync({ sessionId: sid, playerIds: participantIDs });
    }
    try {
      const result = await finalize.mutateAsync({ sessionId: sid });
      setFinalizeResult(result);
      qc.invalidateQueries({ queryKey: ['sessions', sid] });
      onComplete?.(result);
    } catch {
      // Error is handled by finalize.isError
    }
  }

  async function handleCopyLink() {
    if (!finalizeResult) return;
    await navigator.clipboard.writeText(finalizeResult.shareUrl);
    setCopiedType('link');
    setTimeout(() => setCopiedType(null), 2000);
  }

  function getShareMessage(): string {
    if (!finalizeResult) return '';
    const dateStr = finalizeResult.session.date
      ? new Date(finalizeResult.session.date + 'T00:00:00+07:00').toLocaleDateString('vi-VN', {
          day: '2-digit',
          month: '2-digit',
        })
      : '';
    return `🏸 Bill cầu lông ${dateStr} có rồi nha mấy con vợ
Tổng: ${formatMoney(finalizeResult.session.totalCost)} · Đã trả: 0đ · Còn nợ: ${formatMoney(finalizeResult.session.totalCost)}
Vào link tự xem tên + QR, đỡ phải ping admin:
${finalizeResult.shareUrl}`;
  }

  async function handleCopyMessage() {
    await navigator.clipboard.writeText(getShareMessage());
    setCopiedType('message');
    setTimeout(() => setCopiedType(null), 2000);
  }

  const mutationError = (() => {
    for (const m of [createDraft, addItem, removeItem, setParticipantsMut, finalize]) {
      if (m.error instanceof ApiError) {
        return m.error.fieldError('amount') ?? m.error.fieldError('playerIds') ?? m.error.problem.title;
      }
      if (m.isError) return vi.onboarding.session.finalizeError;
    }
    return undefined;
  })();

  return (
    <section className="space-y-5">
      <div className="space-y-2">
        <h2 className="text-xl font-semibold text-foreground">{vi.onboarding.step4.title}</h2>
        <p className="text-sm leading-6 text-muted-foreground">{vi.onboarding.step4.subtitle}</p>
      </div>

      <div className="space-y-4">
        {/* Date / title / location */}
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="flex flex-col gap-1.5">
            <label className="text-sm font-medium text-foreground" htmlFor="session-date">
              {vi.onboarding.session.dateLabel}
            </label>
            <input
              id="session-date"
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              className="flex h-11 w-full rounded-md border border-input bg-muted px-3 py-2 text-sm text-foreground"
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <label className="text-sm font-medium text-foreground" htmlFor="session-title">
              {vi.onboarding.session.titleLabel}
            </label>
            <input
              id="session-title"
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={vi.onboarding.session.titlePlaceholder}
              className="flex h-11 w-full rounded-md border border-input bg-muted px-3 py-2 text-sm text-foreground"
            />
          </div>
        </div>
        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium text-foreground" htmlFor="session-location">
            {vi.onboarding.session.locationLabel}
          </label>
          <input
            id="session-location"
            type="text"
            value={location}
            onChange={(e) => setLocation(e.target.value)}
            placeholder={vi.onboarding.session.locationPlaceholder}
            className="flex h-11 w-full rounded-md border border-input bg-muted px-3 py-2 text-sm text-foreground"
          />
        </div>

        {/* Cost items */}
        <div className="space-y-2">
          <h3 className="text-sm font-semibold text-foreground">{vi.onboarding.session.costItemsTitle}</h3>
          <ul className="space-y-1.5">
            {costItems.map((it) => (
              <li
                key={it.id}
                className="flex items-center justify-between rounded-md border border-input bg-muted px-3 py-2 text-sm text-foreground"
              >
                <span>
                  <span className="font-medium">{vi.onboarding.session.costTypes[it.type]}</span> · {it.label} ·{' '}
                  {formatMoney(it.amount)}
                </span>
                <button
                  type="button"
                  onClick={() => handleRemoveItem(it.id)}
                  className="text-xs text-destructive hover:underline"
                >
                  {vi.onboarding.session.removeCostItem}
                </button>
              </li>
            ))}
          </ul>
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-[100px_1fr_140px]">
            <select
              value={pendingItem.type}
              onChange={(e) => setPendingItem((p) => ({ ...p, type: e.target.value as CostItemType }))}
              className="flex h-11 rounded-md border border-input bg-muted px-3 py-2 text-sm text-foreground"
              aria-label={vi.onboarding.session.costTypeLabel}
            >
              {COST_TYPES.map((t) => (
                <option key={t} value={t}>
                  {vi.onboarding.session.costTypes[t]}
                </option>
              ))}
            </select>
            <input
              type="text"
              value={pendingItem.label}
              onChange={(e) => setPendingItem((p) => ({ ...p, label: e.target.value }))}
              placeholder={vi.onboarding.session.costLabelPlaceholder}
              aria-label={vi.onboarding.session.costLabelLabel}
              className="flex h-11 rounded-md border border-input bg-muted px-3 py-2 text-sm text-foreground"
            />
            <input
              type="text"
              inputMode="numeric"
              value={pendingItem.amount}
              onChange={(e) => setPendingItem((p) => ({ ...p, amount: e.target.value }))}
              placeholder={vi.onboarding.session.costAmountPlaceholder}
              aria-label={vi.onboarding.session.costAmountLabel}
              className="flex h-11 rounded-md border border-input bg-muted px-3 py-2 text-sm text-foreground"
            />
          </div>
          {pendingError ? (
            <p className="text-xs text-destructive" role="alert">
              {pendingError}
            </p>
          ) : null}
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={addItem.isPending}
            onClick={handleAddItem}
          >
            {vi.onboarding.session.addCostItem}
          </Button>
        </div>

        {/* Participants */}
        <div className="space-y-2">
          <h3 className="text-sm font-semibold text-foreground">{vi.onboarding.session.participantsTitle}</h3>
          <p className="text-xs text-muted-foreground">{vi.onboarding.session.participantsHint}</p>
          <ul className="grid grid-cols-2 gap-2 sm:grid-cols-3">
            {(playersQuery.data ?? []).map((p) => {
              const checked = participantIDs.includes(p.id);
              return (
                <li key={p.id}>
                  <label className="flex cursor-pointer items-center gap-2 rounded-md border border-input bg-muted px-3 py-2 text-sm text-foreground">
                    <input
                      type="checkbox"
                      checked={checked}
                      onChange={(e) => handleToggleParticipant(p.id, e.target.checked)}
                      className="size-4 accent-primary"
                    />
                    {p.displayName}
                  </label>
                </li>
              );
            })}
          </ul>
        </div>

        {/* Split preview */}
        <div className="rounded-md border border-primary/40 bg-primary/10 px-3 py-2 text-sm text-foreground">
          {preview.participantCount > 0 && costItems.length > 0
            ? vi.onboarding.session.preview.line(
                formatMoney(preview.splittable),
                preview.participantCount,
                formatMoney(preview.perHead),
              )
            : vi.onboarding.session.preview.empty}
        </div>

        {mutationError ? (
          <p
            className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive"
            role="alert"
          >
            {mutationError}
          </p>
        ) : null}

        {finalizeResult ? (
          // Story 1.11/1.12 confirmation panel with toast feedback
          <section className="space-y-4 rounded-md border border-primary/40 bg-primary/10 p-4">
            <div className="space-y-1 text-center">
              <h2 className="text-xl font-bold text-primary">{vi.onboarding.session.confirmationTitle}</h2>
              <p className="text-sm text-muted-foreground">{vi.onboarding.session.confirmationSubtitle}</p>
            </div>
            <div className="rounded-md bg-background p-3 text-center">
              <p className="text-xs text-muted-foreground">{vi.onboarding.session.shareUrlLabel}</p>
              <p className="mt-1 text-2xl font-bold font-display text-primary">{finalizeResult.shareCode}</p>
              <p className="text-xs text-muted-foreground">{finalizeResult.shareUrl}</p>
            </div>
            <div className="flex gap-2">
              <Button variant="secondary" size="lg" className="flex-1" onClick={handleCopyLink}>
                {copiedType === 'link' ? vi.onboarding.session.copied : vi.onboarding.session.copyLink}
              </Button>
              <Button variant="primary" size="lg" className="flex-1" onClick={handleCopyMessage}>
                {copiedType === 'message' ? vi.onboarding.session.copied : vi.onboarding.session.copyMessage}
              </Button>
            </div>
            {copiedType ? (
              <p className="text-center text-xs text-primary">
                {copiedType === 'link' ? vi.onboarding.session.copiedLink : vi.onboarding.session.copiedMessage}
              </p>
            ) : null}
            <Button
              type="button"
              variant="outline"
              size="lg"
              className="w-full"
              onClick={() => window.location.href = '/'}
            >
              {vi.onboarding.session.complete}
            </Button>
          </section>
        ) : (
          <>
            <Button
              type="button"
              size="lg"
              className="w-full"
              disabled={!canSave || setParticipantsMut.isPending || createDraft.isPending}
              onClick={handleSave}
            >
              {setParticipantsMut.isPending || createDraft.isPending
                ? vi.onboarding.session.saving
                : vi.onboarding.session.saveDraft}
            </Button>

            {canSave ? (
              <Button
                type="button"
                variant="primary"
                size="lg"
                className="w-full"
                disabled={!canSave || finalize.isPending || setParticipantsMut.isPending || createDraft.isPending}
                onClick={handleFinalize}
              >
                {finalize.isPending
                  ? vi.onboarding.session.finalizing
                  : vi.onboarding.session.finalize}
              </Button>
            ) : null}

            {canSave ? (
              <p className="text-center text-xs text-muted-foreground">{vi.onboarding.session.readyForFinalize}</p>
            ) : null}
          </>
        )}
      </div>
    </section>
  );
}
