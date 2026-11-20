/** デザイントークン。実際の色は style.css の CSS 変数（ライト/ダークで切り替わる）。 */
const token = (name) => `rgb(var(--c-${name}) / <alpha-value>)`

/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  // ダークは OS の設定（prefers-color-scheme）に追従する。設定は不要。
  theme: {
    extend: {
      colors: {
        canvas: token('canvas'), // ページ背景
        surface: token('surface'), // カード・ダイアログ
        subtle: token('subtle'), // 薄い面（見出し行・コード・ホバー）
        line: token('line'), // 罫線
        fg: token('fg'), // 本文
        muted: token('muted'), // 補助文字（AA を満たす濃さ）
        accent: {DEFAULT: token('accent'), hover: token('accent-hover'), soft: token('accent-soft'), fg: token('accent-fg')},
        danger: {DEFAULT: token('danger'), hover: token('danger-hover'), soft: token('danger-soft'), fg: token('danger-fg'), line: token('danger-line')},
        success: {DEFAULT: token('success'), soft: token('success-soft'), fg: token('success-fg'), line: token('success-line')},
        warning: {DEFAULT: token('warning'), soft: token('warning-soft'), fg: token('warning-fg'), line: token('warning-line')},
      },
      borderColor: {DEFAULT: token('line')},
      borderRadius: {card: '0.75rem', ctl: '0.5rem'},
      boxShadow: {
        card: '0 1px 2px rgb(var(--shadow) / 0.06), 0 1px 3px rgb(var(--shadow) / 0.08)',
        pop: '0 10px 30px rgb(var(--shadow) / 0.25)',
      },
      spacing: {gutter: '1.5rem'},
      fontFamily: {
        // システムフォントのみ（外部フォント・CDN なし）。日本語対応のフォントを含める。
        sans: ['system-ui', '-apple-system', 'BlinkMacSystemFont', '"Segoe UI"', '"Hiragino Sans"', '"Hiragino Kaku Gothic ProN"', '"Yu Gothic UI"', '"Noto Sans JP"', 'Meiryo', 'sans-serif'],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Consolas', '"Liberation Mono"', '"Noto Sans Mono CJK JP"', 'monospace'],
      },
    },
  },
  plugins: [],
}
