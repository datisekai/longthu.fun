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
    // Story 1.8 — Group create step.
    group: {
      title: 'Tạo group cầu lông',
      subtitle: 'Đặt tên group bạn hay đánh — VD: "Tối thứ 3", "Sân K34 cuối tuần".',
      nameLabel: 'Tên group',
      namePlaceholder: 'VD: Tối thứ 3',
      submit: 'Tạo group',
      submitting: 'Đang tạo group…',
      helper: 'Mỗi group là một nhóm bạn chơi riêng; sau này bạn có thể tạo thêm group khác.',
      errors: {
        nameRequired: 'Tên group không được để trống',
        nameTooLong: 'Tên group tối đa 120 ký tự',
      },
    },
    step3: {
      title: 'Tiếp theo: thêm người chơi',
      body: 'Group đã tạo xong. Bước sau mình sẽ thêm các con vợ vào group.',
    },
    // Story 1.9 — bulk-add Players.
    players: {
      title: 'Thêm các con vợ vào group',
      subtitle: 'Mỗi dòng là 1 tên người chơi. Tối đa theo gói: Free 6, PRO 8, PRO Plus 15.',
      namesLabel: 'Danh sách người chơi',
      namesPlaceholder: 'Đạt\nLý\nTâm\nHùng',
      helper: 'Tên giữ nguyên dấu tiếng Việt. Trùng tên trong group sẽ bị từ chối.',
      submit: 'Thêm vào group',
      submitting: 'Đang thêm…',
      summary: (count: number) => `${count} người sẵn sàng thêm vào group.`,
      tierHint: (cap: number) => `Gói hiện tại cho phép tối đa ${cap} người/group.`,
      errors: {
        namesRequired: 'Cần ít nhất 1 tên người chơi',
        tooMany: (cap: number) => `Quá nhiều: tối đa ${cap} người/group theo gói hiện tại`,
        duplicateInSubmit: 'Có tên trùng trong danh sách, sửa lại rồi gửi nhé',
        nameTooLong: 'Mỗi tên tối đa 60 ký tự',
      },
    },
    step4: {
      title: 'Tạo buổi đánh đầu tiên',
      subtitle: 'Nhập ngày, các khoản chi (sân / cầu / nước), tick những con vợ đã đánh — app sẽ chia tiền giúp.',
    },
    // Story 1.10 — Session draft.
    session: {
      dateLabel: 'Ngày đánh',
      titleLabel: 'Tiêu đề (không bắt buộc)',
      titlePlaceholder: 'VD: Tối thứ 3',
      locationLabel: 'Địa điểm (không bắt buộc)',
      locationPlaceholder: 'VD: Sân K34',
      costItemsTitle: 'Khoản chi',
      addCostItem: 'Thêm khoản chi',
      removeCostItem: 'Xóa',
      costTypeLabel: 'Loại',
      costLabelLabel: 'Mô tả',
      costAmountLabel: 'Số tiền (VND)',
      costLabelPlaceholder: 'VD: Sân 360k',
      costAmountPlaceholder: 'VD: 360000',
      includedInSplit: 'Chia đều',
      costTypes: {
        court: 'Sân',
        shuttle: 'Cầu',
        water: 'Nước',
        other: 'Khác',
        discount: 'Giảm giá',
      },
      participantsTitle: 'Người chơi tham gia',
      participantsHint: 'Tick những người đã đánh buổi này. Phải tick ít nhất 1 người.',
      preview: {
        empty: 'Thêm khoản chi và tick người chơi để xem chia tiền.',
        line: (total: string, count: number, perHead: string) =>
          `Tổng: ${total} · ${count} suất · ${perHead}/người`,
      },
      errors: {
        dateRequired: 'Chọn ngày đánh',
        amountInvalid: 'Số tiền không hợp lệ',
        labelRequired: 'Cần điền mô tả',
        atLeastOneItem: 'Cần ít nhất 1 khoản chi',
        atLeastOneParticipant: 'Cần ít nhất 1 người chơi',
        generic: 'Có gì đó sai — thử lại?',
      },
      saving: 'Đang lưu…',
      saveDraft: 'Lưu buổi đánh',
      readyForFinalize: 'Sẵn sàng chốt bill 🚀',
      // Story 1.11 finalize
      finalize: 'Chốt bill 🚀',
      finalizing: 'Đang chốt…',
      confirmationTitle: 'Bill đã chốt! 🎉',
      confirmationSubtitle: 'Link chia cho mấy con vợ đây:',
      copyLink: 'Copy link',
      copyMessage: 'Copy tin nhắn',
      copiedLink: 'Đã copy link',
      copiedMessage: 'Đã copy tin nhắn',
      shareUrlLabel: 'Link bill',
      finalizeError: 'Không chốt được — thử lại?',
    },
    finalizeBlocked: 'Bạn cần thêm tài khoản nhận tiền trước khi chốt buổi.',
  },
} as const;
