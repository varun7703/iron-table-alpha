import React from 'react';
import type { Game } from '../types';

export function TopBar({
  connected,
  game,
  copied,
  onToggleHelp,
  onOpenExit,
  onCopyCode,
}: {
  connected: boolean;
  game: Game | null;
  copied: boolean;
  onToggleHelp: () => void;
  onOpenExit: () => void;
  onCopyCode: () => void;
}) {
  return (
    <header className="topbar">
      <a className="brand" href="/" aria-label="Iron Table home">
        <span className="brand-mark">⌖</span>
        <span>
          IRON TABLE<small>FIELD COMMAND / ALPHA 0.2</small>
        </span>
      </a>
      <div className="top-right">
        <span className={'connection ' + (connected ? 'online' : '')}>
          {connected ? 'COMMAND LINK ONLINE' : 'RECONNECTING'}
        </span>
        <button className="subtle" onClick={onToggleHelp}>
          Field manual <span>↗</span>
        </button>
        {game && (
          <button className="exit-round" onClick={onOpenExit}>
            {game.phase === 'lobby' ? 'Leave lobby' : 'Exit round'}
          </button>
        )}
        {game && (
          <button className="room" onClick={onCopyCode}>
            {copied ? 'COPIED' : `ROOM ${game.code}`} ⧉
          </button>
        )}
      </div>
    </header>
  );
}
