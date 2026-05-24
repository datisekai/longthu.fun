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
} as const;
