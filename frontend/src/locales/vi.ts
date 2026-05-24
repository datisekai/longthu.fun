// Vietnamese-only UI strings. Per EXPERIENCE.md §Voice and Tone:
// every user-visible label lives here. No inline strings in components.
// English fallback in a user-facing page is a build-time concern (Story 7.6 lint).

export const vi = {
  app: {
    name: 'Longthu.fun',
    tagline: 'Chia bill cầu lông, app tự khớp.',
  },
  home: {
    greeting: 'Longthu.fun đang chạy 🏸',
    subtitle: 'Sẵn sàng chốt bill cầu lông cho mấy con vợ.',
  },
  money: {
    // Units used by the screen-reader expansion helper (lib/money.ts).
    unit: {
      thousand: 'nghìn',
      million: 'triệu',
      dong: 'đồng',
    },
  },
  // Charge status → Vietnamese label. Translations are FIXED — copy authors
  // do not invent synonyms (EXPERIENCE.md §Status verb translations).
  status: {
    paid: 'đã trả',
    unpaid: 'chưa trả',
    pending_confirmation: 'chờ xác nhận',
    suspected: 'nghi khớp',
    waived: 'miễn',
    matched: 'đã khớp',
    unmatched: 'chưa khớp',
    cancelled: 'đã hủy',
    expired: 'hết hạn',
  },
  // Common verbs (action affordances). Add sparingly; prefer feature-scoped
  // copy where the verb's context matters.
  actions: {
    save: 'Lưu',
    cancel: 'Hủy',
    confirm: 'Xác nhận',
    delete: 'Xóa',
    edit: 'Sửa',
    back: 'Quay lại',
    next: 'Tiếp',
    done: 'Xong',
    copy: 'Sao chép',
    send: 'Gửi',
    loading: 'Đang tải…',
  },
} as const;
