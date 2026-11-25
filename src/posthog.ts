import posthog from 'posthog-js';

const posthogKey = import.meta.env.VITE_POSTHOG_KEY;
const posthogHost = import.meta.env.VITE_POSTHOG_HOST;
export const isPosthogEnabled = Boolean(posthogKey && posthogHost);

if (posthogKey && posthogHost) {
  posthog.init(posthogKey, {
    api_host: posthogHost,
    defaults: '2026-08-30',
    capture_exceptions: {
      capture_unhandled_errors: true,
      capture_unhandled_rejections: true,
      capture_console_errors: false,
    },
    logs: {
      serviceName: 'iron-table-web',
      environment: import.meta.env.MODE,
    },
  });
} else if (import.meta.env.DEV) {
  console.warn('PostHog disabled: set VITE_POSTHOG_KEY and VITE_POSTHOG_HOST to enable analytics.');
}

export const posthogGameLogger = {
  info(message: string, attributes?: Record<string, string | number | boolean>) {
    if (isPosthogEnabled) posthog.logger.info(message, attributes);
  },
  warn(message: string, attributes?: Record<string, string | number | boolean>) {
    if (isPosthogEnabled) posthog.logger.warn(message, attributes);
  },
};

export default posthog;
