/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    // Template files
    '../templates/**/*.templ',
    '../templates/**/*.html',
    // Go files (may contain CSS classes in literals)
    '../**/*.go',
    // Avoid node_modules
    '!../node_modules/**',
    '!../../node_modules/**'
  ],
  theme: {
    extend: {
      // Mobile-first responsive breakpoints
      screens: {
        'xs': '320px',   // Extra small phones
        'sm': '480px',   // Small phones
        'md': '768px',   // Tablets (main mobile/tablet breakpoint)
        'lg': '1024px',  // Small laptops
        'xl': '1280px',  // Large laptops
        '2xl': '1536px'  // Desktop
      },

      // Chat-specific color palette
      colors: {
        // Primary brand colors
        'duck-blue': {
          50: '#eff6ff',
          100: '#dbeafe',
          500: '#3b82f6',
          600: '#2563eb',
          700: '#1d4ed8',
          900: '#1e3a8a'
        },

        // Chat message colors
        'message': {
          'own': '#dcf8c6',      // User's own messages (light green)
          'other': '#ffffff',     // Other users' messages (white)
          'system': '#f3f4f6'     // System messages (light gray)
        },

        // Status colors
        'status': {
          'online': '#10b981',    // Green for online
          'offline': '#6b7280',   // Gray for offline
          'typing': '#f59e0b'     // Amber for typing
        }
      },

      // Typography optimized for chat
      fontFamily: {
        'sans': ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        'mono': ['JetBrains Mono', 'Menlo', 'Monaco', 'monospace']
      },

      // Spacing for touch targets
      spacing: {
        '18': '4.5rem',   // 72px - ideal touch target
        '88': '22rem',    // Large content areas
        '128': '32rem'    // Extra large content areas
      },

      // Chat-specific animations
      animation: {
        'fade-in': 'fadeIn 0.2s ease-in-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'pulse-slow': 'pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite'
      },

      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { transform: 'translateY(10px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' }
        }
      },

      // Layout utilities
      maxWidth: {
        'chat': '768px',      // Max width for chat container
        'sidebar': '320px'    // Sidebar width on larger screens
      },

      minHeight: {
        'touch': '44px',      // Minimum touch target height
        'message': '2.5rem'   // Minimum message height
      }
    }
  },

  plugins: [
    // Add forms plugin for better form styling
    require('@tailwindcss/forms')({
      strategy: 'class' // Use .form-input, .form-select, etc. classes
    }),

    // Add aspect-ratio plugin for media
    require('@tailwindcss/aspect-ratio'),

    // Add container queries plugin for responsive navigation
    require('@tailwindcss/container-queries'),

    // Custom plugin for chat-specific utilities
    function({ addUtilities, theme }) {
      const newUtilities = {
        // Touch-friendly button
        '.btn-touch': {
          minHeight: theme('minHeight.touch'),
          minWidth: theme('minHeight.touch'),
          padding: theme('spacing.3') + ' ' + theme('spacing.6'),
          borderRadius: theme('borderRadius.lg'),
          fontSize: theme('fontSize.base'),
          fontWeight: theme('fontWeight.medium'),
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          cursor: 'pointer',
          transition: 'all 0.2s ease-in-out'
        },

        // Message bubble base
        '.message-bubble': {
          padding: theme('spacing.3') + ' ' + theme('spacing.4'),
          borderRadius: theme('borderRadius.xl'),
          maxWidth: '85%',
          wordWrap: 'break-word',
          lineHeight: theme('lineHeight.relaxed')
        },

        // Safe area padding for mobile
        '.safe-area-padding': {
          paddingTop: 'env(safe-area-inset-top)',
          paddingBottom: 'env(safe-area-inset-bottom)',
          paddingLeft: 'env(safe-area-inset-left)',
          paddingRight: 'env(safe-area-inset-right)'
        },

        // Scrollable area with custom scrollbar
        '.scrollable': {
          '&::-webkit-scrollbar': {
            width: '6px'
          },
          '&::-webkit-scrollbar-track': {
            background: theme('colors.gray.100')
          },
          '&::-webkit-scrollbar-thumb': {
            background: theme('colors.gray.300'),
            borderRadius: theme('borderRadius.full')
          },
          '&::-webkit-scrollbar-thumb:hover': {
            background: theme('colors.gray.400')
          }
        }
      }

      addUtilities(newUtilities, ['responsive', 'hover', 'focus'])
    }
  ],

  // Optimize for production
  corePlugins: {
    // Disable unused features for smaller bundle
    float: false,
    clear: false,
    skew: false,
    caretColor: false,
    sepia: false
  }
}