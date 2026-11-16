/** Vite replaces this public value at build time; it is not a secret. */
export function websocketURL(): string {
  const configured = (import.meta.env.VITE_GAME_SERVER_URL || '').trim();
  const url = new URL(configured || location.origin);
  if (!['http:', 'https:', 'ws:', 'wss:'].includes(url.protocol)) {
    throw new Error('VITE_GAME_SERVER_URL must use HTTPS or WebSocket.');
  }
  url.protocol = ['https:', 'wss:'].includes(url.protocol) ? 'wss:' : 'ws:';
  if (location.protocol === 'https:' && url.protocol !== 'wss:') {
    throw new Error('The game server must use HTTPS when the website uses HTTPS.');
  }
  url.pathname = '/ws';
  url.search = '';
  url.hash = '';
  return url.toString();
}
