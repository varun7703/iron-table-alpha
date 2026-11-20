import React from 'react';
import { CAPITAL_CURRENCY_BONUS, CAPITAL_HUMANS_BONUS, CAPITAL_VP, MAX_STACKS_PER_SECTOR } from '../combat';

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

export function ManualModal({ onClose }: { onClose: () => void }) {
  return (
    <div
      className="modal-backdrop"
      onKeyDown={e => {
        if (e.key === 'Escape') onClose();
      }}
    >
      <section className="manual" role="dialog" aria-modal="true" aria-labelledby="manual-title">
        <button autoFocus className="close" onClick={onClose} aria-label="Close field manual">
          ×
        </button>
        <span className="eyebrow">COMMANDER’S HANDBOOK</span>
        <h2 id="manual-title">One map. Three phases.</h2>
        <p>
          Draft territories in snake dice order (A → B → C → D → D → C → B → A). Every territory yields manpower and
          currency, plus food or iron. Static terrain yields are always visible across the map.
        </p>
        <p>
          <b>Capitals (45s selection):</b> rival picks stay concealed until Round 1 begins. Holding your own capital
          grants <b>+{CAPITAL_CURRENCY_BONUS} currency</b> and <b>+{CAPITAL_HUMANS_BONUS} manpower</b> each Gather, plus{' '}
          <b>+15% direct combat defense</b> in your capital (stacking to +38% with Hold &amp; defend). Each controlled
          capital (yours or captured) awards <b>+{CAPITAL_VP} VP</b> at campaign end.
        </p>
        <p>
          <b>1-Hop Line of Sight:</b> you see ownership, capitals, garrisons, and battle dispatches on sectors you hold
          and all sectors sharing a border with your territory or armies — across Plan, Simulate, and Gather.
        </p>
        <p>
          <b>Plan (90s):</b> one order per army. Friendly-sector transfers are free; attacking or claiming neutral
          sectors costs 1 / 2 / 3 currency per infantry / artillery / tank. Artillery defaults to Bombard across one
          border after movement (×1.5 vs tanks, ×1.2 vs infantry), unless overrun in melee. Hold &amp; defend entrenches
          an army (+20% direct combat, −25% bombardment taken). You may also split an army (up to{' '}
          {MAX_STACKS_PER_SECTOR} stacks per sector).
        </p>
        <p>
          <b>Simulate:</b> border clashes resolve first, then movement, then artillery bombardment, then destination
          battles and retreats. Durability is 10 / 14 / 20 HP per infantry / artillery / tank. Use the playback
          scrubber to pause or step through frames, and check the Last Round tab during Gather or Plan.
        </p>
        <p>
          <b>Gather (120s):</b> income arrives automatically. Infantry costs 1 human + 4 currency; artillery 2 humans + 4
          currency + 4 iron; tanks 3 humans + 8 currency + 8 iron. Merge, reinforce, or split unfed armies (max{' '}
          {MAX_STACKS_PER_SECTOR} stacks per sector).
        </p>
        <p>
          Minimum rations (0.5 food per troop, rounded up) give 65% attack. Normal rations maintain strength. Extra
          rations (1.5, rounded up) heal 20 health. Unfed troops auto-feed when Gather advances; starvation costs 25
          health and reduces attack to 30%.
        </p>
        <p>
          After 20 rounds (or total conquest), highest <b>Victory Points (Territories + 2 × Capitals)</b> wins.
          Disconnected commanders keep their seat for 5 minutes. Matches are held in server memory and reset if the
          server restarts.
        </p>
      </section>
    </div>
  );
}
