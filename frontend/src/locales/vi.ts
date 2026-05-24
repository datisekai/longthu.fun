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
  // Auth surface — Story 1.5.
  auth: {
    register: {
      title: 'Đăng ký',
      subtitle: 'Tạo tài khoản host cho nhóm cầu lông của bạn.',
      emailLabel: 'Email',
      emailPlaceholder: 'ban@example.com',
      passwordLabel: 'Mật khẩu',
      passwordHint: 'Ít nhất 8 ký tự.',
      displayNameLabel: 'Tên hiển thị',
      displayNamePlaceholder: 'VD: Anh Hùng',
      submit: 'Đăng ký',
      submitting: 'Đang đăng ký…',
      goLogin: 'Đã có tài khoản? Đăng nhập',
    },
    login: {
      title: 'Đăng nhập',
      subtitle: 'Quản lý nhóm cầu lông của bạn.',
      emailLabel: 'Email',
      passwordLabel: 'Mật khẩu',
      submit: 'Đăng nhập',
      submitting: 'Đang đăng nhập…',
      goRegister: 'Chưa có tài khoản? Đăng ký',
      goReset: 'Quên mật khẩu?',
    },
    reset: {
      title: 'Quên mật khẩu?',
      body: 'MVP chưa có reset mật khẩu qua email. Nhắn admin (founder) trên Telegram, admin sẽ reset thủ công.',
      telegramLabel: 'Nhắn Telegram cho admin',
      telegramHref: 'https://t.me/datisekai',
      goLogin: 'Quay lại đăng nhập',
    },
    errors: {
      emailInvalid: 'Email không hợp lệ',
      passwordShort: 'Mật khẩu cần ít nhất 8 ký tự',
      displayNameRequired: 'Cần điền tên hiển thị',
      generic: 'Có gì đó sai — thử lại?',
    },
    logout: 'Đăng xuất',
  },
  onboarding: {
    title: 'Thiết lập nhóm đầu tiên',
    stepLabel: 'Bước 1/4',
    bank: {
      title: 'Tài khoản nhận tiền',
      subtitle: 'Thêm tài khoản ngân hàng để người chơi biết chuyển tiền về đâu.',
      bankLabel: 'Ngân hàng',
      accountNumberLabel: 'Số tài khoản',
      accountNumberPlaceholder: 'VD: 123456789',
      accountHolderLabel: 'Tên chủ tài khoản',
      accountHolderPlaceholder: 'VD: NGUYEN VAN A',
      submit: 'Lưu tài khoản',
      submitting: 'Đang lưu…',
      helper: 'Tài khoản đầu tiên sẽ tự động đặt làm mặc định.',
      bankOptions: {
        mbbank: 'MBBank',
        vcb: 'Vietcombank',
        tpb: 'TPBank',
      },
      errors: {
        bankRequired: 'Chọn ngân hàng',
        accountNumberInvalid: 'Số tài khoản chỉ gồm 8-16 chữ số',
        holderRequired: 'Nhập tên chủ tài khoản',
      },
    },
    step2: {
      title: 'Tiếp theo: tạo group',
      body: 'Tài khoản ngân hàng đã lưu. Bước sau mình sẽ tạo group cầu lông đầu tiên.',
    },
    finalizeBlocked: 'Bạn cần thêm tài khoản nhận tiền trước khi chốt buổi.',
  },
} as const;
