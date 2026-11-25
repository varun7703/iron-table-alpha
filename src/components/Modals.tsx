import React from 'react';
export { ManualModal } from './FieldManual';

export function ExitModal({
  connected,
  exiting,
  onClose,
  onConfirm,
}: {
  connected: boolean;
  exiting: boolean;
  onClose: () => void;
  onConfirm: () => void;
}) {
  return (
    <div className="modal-backdrop">
      <section
        className="manual exit-dialog"
        onKeyDown={e => {
          if (e.key === 'Escape' && !exiting) onClose();
        }}
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="exit-title"
        aria-describedby="exit-description"
      >
        <h2 id="exit-title">Leave this game?</h2>
        <p id="exit-description">
          Your armies will be dissolved and all your territories will become neutral. Your seat and capital will be
          removed. You cannot rejoin an ongoing match after exiting.
        </p>
        <div className="exit-actions">
          <button autoFocus className="secondary" disabled={exiting} onClick={onClose}>
            Stay in game
          </button>
          <button className="primary exit-confirm" disabled={!connected || exiting} onClick={onConfirm}>
            {exiting ? 'Leaving…' : 'Dissolve armies & exit'}
          </button>
        </div>
      </section>
    </div>
  );
}
