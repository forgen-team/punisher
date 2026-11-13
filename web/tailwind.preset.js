// Tailwind preset (ported from @witems/config/tailwind/preset.js).
// v2 "Linear/Vercel 풍" 토큰 — 값은 src/styles/design-tokens.css 의 CSS 변수가 원천.
//
// 새 코드에서 쓰는 클래스 (전부 다크모드 자동 대응)
//   배경  bg-canvas(본문 패널) · bg-canvas-subtle(바깥·사이드바) · bg-canvas-elevated(카드) · hover:bg-canvas-hover
//         bg-canvas-active · bg-canvas-translucent(블러 바) · bg-overlay(딤)
//         bg-canvas-selected(탭·segmented 선택 칸) · bg-canvas-muted(본문 위 옅은 면 — 말풍선)
//   글자  text-fg · text-fg-secondary · text-fg-muted · text-fg-inverse
//   선    border-line · divide-line · border-line-subtle · border-line-strong
//   강조  text-accent(=accent-solid) · text-accent-text · bg-accent-subtle · bg-accent-solid · text-accent-on · ring-focus
//   상태  text-success / bg-success-subtle · text-warning / bg-warning-subtle · text-danger / bg-danger-subtle
//   글꼴  text-xs(12) · text-sm(14) · text-base(16) · text-xl(20) · text-2xl(24) — text-lg(18) 등은 스케일 밖
//   간격  Tailwind 기본 4px 스케일(1–12) 그대로 · 안전영역 pt-safe-top / pb-safe-bottom
//   모양  rounded-sm(6) · rounded-md(8) · rounded-lg(10) · rounded-xl(14) · rounded-full · shadow-xs · shadow-sm · shadow-overlay
//   자간  tracking-tight(제목) · tracking-ui(본문 UI)
//   폭    fold:(600px, 폴드 펼침 2단) · sm · md · lg(1024, 사이드바) — 2단 판정은 JS 에서 useIsSplitLayout()
//   z     z-nav(30) · z-panel(40) · z-sheet(45) · z-modal(50) · z-toast(9999)
// 주의: CSS 변수 색은 `/10` 같은 opacity modifier가 동작하지 않는다 → *-subtle 토큰을 쓸 것.
// shadcn HSL·raw gray·witems-*·chart·cta/surface/ink 등 옛 별칭은 D5에서 제거됐다 — 다시 추가하지 말 것.

const v = (name) => `var(${name})`;

/** @type {import("tailwindcss").Config} */
const preset = {
  content: [],
  theme: {
    // fold(600px): 갤럭시 폴드 펼친 화면(약 680~900px)부터 2단(목록+상세). 커버 화면(360~410px)은 모바일 그대로.
    // 순서가 CSS 우선순위이므로 기본값을 다시 적고 그 앞에 fold 를 둔다(extend 로 넣으면 맨 뒤에 붙어 md·lg 를 덮는다).
    screens: {
      fold: '600px',
      sm: '640px',
      md: '768px',
      lg: '1024px',
      xl: '1280px',
      '2xl': '1536px',
    },
    extend: {
      colors: {
        /* ===== 새 시맨틱 토큰 ===== */
        canvas: {
          DEFAULT: v('--color-bg'),
          subtle: v('--color-bg-subtle'),
          elevated: v('--color-bg-elevated'),
          hover: v('--color-surface-hover'),
          active: v('--color-surface-active'),
          translucent: v('--color-bg-translucent'),
          selected: v('--color-bg-selected'),
          muted: v('--color-bg-muted'),
        },
        fg: {
          DEFAULT: v('--color-text'),
          secondary: v('--color-text-secondary'),
          muted: v('--color-text-muted'),
          inverse: v('--color-inverse-text'),
        },
        line: {
          DEFAULT: v('--color-border'),
          strong: v('--color-border-strong'),
          subtle: v('--color-border-subtle'),
        },
        inverse: v('--color-inverse-bg'),
        focus: v('--color-focus-ring'),
        overlay: v('--color-overlay'),
        accent: {
          DEFAULT: v('--color-accent'),
          solid: v('--color-accent'),
          hover: v('--color-accent-hover'),
          subtle: v('--color-accent-subtle'),
          text: v('--color-accent-text'),
          on: v('--color-on-accent'),
        },
        success: {
          DEFAULT: v('--color-success'),
          subtle: v('--color-success-subtle'),
        },
        warning: {
          DEFAULT: v('--color-warning'),
          subtle: v('--color-warning-subtle'),
        },
        danger: {
          DEFAULT: v('--color-danger'),
          subtle: v('--color-danger-subtle'),
          on: v('--color-on-danger'),
        },

      },
      fontFamily: {
        sans: ['var(--font-sans)'],
        mono: ['var(--font-mono)'],
      },
      fontSize: {
        xs: ['var(--text-xs)', { lineHeight: 'var(--leading-xs)' }],
        sm: ['var(--text-sm)', { lineHeight: 'var(--leading-sm)' }],
        base: ['var(--text-md)', { lineHeight: 'var(--leading-md)' }],
        xl: ['var(--text-lg)', { lineHeight: 'var(--leading-lg)' }],
        '2xl': ['var(--text-xl)', { lineHeight: 'var(--leading-xl)' }],
      },
      letterSpacing: {
        tight: 'var(--tracking-tight)',
        ui: 'var(--tracking-ui)',
      },
      fontWeight: {
        normal: 'var(--font-weight-regular)',
        medium: 'var(--font-weight-medium)',
        semibold: 'var(--font-weight-semibold)',
      },
      borderRadius: {
        sm: 'var(--radius-sm)',
        md: 'var(--radius-md)',
        lg: 'var(--radius-lg)',
        xl: 'var(--radius-xl)',
        full: 'var(--radius-full)',
      },
      boxShadow: {
        xs: 'var(--shadow-xs)',
        sm: 'var(--shadow-sm)',
        overlay: 'var(--shadow-overlay)',
      },
      zIndex: {
        nav: 'var(--z-nav)',
        panel: 'var(--z-panel)',
        sheet: 'var(--z-sheet)',
        modal: 'var(--z-modal)',
        toast: 'var(--z-toast)',
      },
      spacing: {
        'safe-top': 'var(--safe-top)',
        'safe-right': 'var(--safe-right)',
        'safe-bottom': 'var(--safe-bottom)',
        'safe-left': 'var(--safe-left)',
        touch: 'var(--touch-target)',
      },
      minHeight: { touch: 'var(--touch-target)' },
      minWidth: { touch: 'var(--touch-target)' },
      transitionDuration: {
        fast: 'var(--duration-fast)',
        base: 'var(--duration-base)',
        slow: 'var(--duration-slow)',
      },
      transitionTimingFunction: { standard: 'var(--ease-standard)' },
      animation: {
        'fade-in': 'fadeIn 0.2s ease-out',
        'slide-up': 'fadeSlideUp 0.4s ease-out',
        'slide-down': 'fadeSlideDown 0.4s ease-out',
        'scale-in': 'scaleIn 0.3s ease-out',
        'sheet-up': 'sheetUp var(--duration-slow) var(--ease-standard)',
        'sheet-right': 'sheetRight var(--duration-slow) var(--ease-standard)',
      },
      keyframes: {
        fadeIn: { from: { opacity: '0' }, to: { opacity: '1' } },
        fadeSlideUp: {
          from: { opacity: '0', transform: 'translateY(16px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
        fadeSlideDown: {
          from: { opacity: '0', transform: 'translateY(-16px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
        scaleIn: {
          from: { opacity: '0', transform: 'scale(0.97)' },
          to: { opacity: '1', transform: 'scale(1)' },
        },
        sheetUp: { from: { transform: 'translateY(100%)' }, to: { transform: 'translateY(0)' } },
        sheetRight: { from: { transform: 'translateX(100%)' }, to: { transform: 'translateX(0)' } },
      },
    },
  },
};

export default preset;
