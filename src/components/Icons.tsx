import React from 'react';

export function Icon({ type, size = 24 }: { type: string; size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {type === 'infantry' ? (
        <>
          <circle cx="15" cy="7" r="3" />
          <path d="M10 15l5-3 5 3-2 7H12zM13 22l-3 7m7-7 4 7M8 20l17-9M22 9l3 5" />
        </>
      ) : type === 'tank' ? (
        <>
          <rect x="3" y="19" width="26" height="9" rx="4" />
          <path d="M8 19l3-8h10l4 8M19 12l11-5M8 24h16" />
          <circle cx="9" cy="24" r="1" />
          <circle cx="23" cy="24" r="1" />
        </>
      ) : type === 'artillery' ? (
        <>
          <circle cx="11" cy="24" r="5" />
          <path d="M11 24l9-12 8-8-3-3-9 9-7 11M15 23l13 5M20 12l4 4" />
        </>
      ) : type === 'food' ? (
        <path d="M16 29V6M16 13C5 15 5 5 7 4c5 0 9 4 9 9zm0 8C5 23 5 13 7 12c5 0 9 4 9 9zm0-4c11 2 11-8 9-9-5 0-9 4-9 9z" />
      ) : type === 'iron' ? (
        <path d="M4 14l7-9 13 3 5 13-9 7-13-3zM11 5l4 12 14 4M15 17l5 11M4 14l11 3" />
      ) : type === 'humans' ? (
        <>
          <circle cx="16" cy="9" r="5" />
          <path d="M6 28v-4a10 10 0 0120 0v4z" />
        </>
      ) : (
        <>
          <circle cx="16" cy="16" r="12" />
          <path d="M20 10h-6a3 3 0 000 6h4a3 3 0 010 6h-7m5-15v18" />
        </>
      )}
    </svg>
  );
}
