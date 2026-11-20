import React from 'react';

export function LandingView({
  name,
  code,
  connected,
  onNameChange,
  onCodeChange,
  onCreate,
  onJoin,
}: {
  name: string;
  code: string;
  connected: boolean;
  onNameChange: (v: string) => void;
  onCodeChange: (v: string) => void;
  onCreate: () => void;
  onJoin: () => void;
}) {
  return (
    <main className="landing">
      <div className="intro">
        <span className="eyebrow">
          <span className="live-dot" /> MULTIPLAYER STRATEGY / 2–4 COMMANDERS
        </span>
        <h1>
          Every border
          <br />
          has a <i>breaking point.</i>
        </h1>
        <p>
          Gather your forces. Commit your orders.
          <br />
          Take your place at the war table.
        </p>
        <div className="landing-features">
          <span>
            01 <b>PLAN IN SECRET</b>
          </span>
          <span>
            02 <b>RESOLVE TOGETHER</b>
          </span>
          <span>
            03 <b>BUILD YOUR ARMY</b>
          </span>
        </div>
        <div className="mini-map">
          <svg viewBox="0 0 480 180">
            <defs>
              <pattern id="lg" width="20" height="20" patternUnits="userSpaceOnUse">
                <path d="M20 0H0V20" fill="none" stroke="#526153" strokeWidth=".5" />
              </pattern>
            </defs>
            <rect width="480" height="180" fill="url(#lg)" />
            {[0, 1, 2, 3, 4, 5, 6, 7].map(i => (
              <path
                key={i}
                d={`M${i * 60 + 10},40 l50,-${15 + (i % 3) * 6} 25,60 -25,55 -50,-15 -15,-50z`}
                fill={i < 4 ? '#64715b' : '#b27853'}
                fillOpacity=".65"
                stroke="#d3c6aa"
              />
            ))}
            <path d="M140 100Q240 20 340 100" stroke="#f4dfaa" strokeWidth="2" strokeDasharray="6 6" fill="none" />
            <circle cx="140" cy="100" r="6" fill="#f4dfaa" />
            <circle cx="340" cy="100" r="6" fill="#f4dfaa" />
          </svg>
          <span>THE CONTINENT AWAITS YOUR ORDERS</span>
        </div>
      </div>

      <section className="lobby-card">
        <span className="eyebrow">OPERATIONS ROOM</span>
        <h2>Take command.</h2>
        <p>Start a campaign or join your friends.</p>
        <label>
          COMMANDER NAME
          <input
            maxLength={24}
            placeholder="Enter your callsign"
            value={name}
            onChange={e => onNameChange(e.target.value)}
          />
        </label>
        <button className="primary" disabled={!name.trim() || !connected} onClick={onCreate}>
          Create a lobby <span>→</span>
        </button>
        <div className="divider">
          <span>OR JOIN AN OPERATION</span>
        </div>
        <label>
          ROOM CODE
          <input
            maxLength={6}
            placeholder="6-character code"
            className="code-input"
            value={code}
            onChange={e => onCodeChange(e.target.value.toUpperCase())}
          />
        </label>
        <button className="secondary" disabled={!name.trim() || code.length !== 6 || !connected} onClick={onJoin}>
          Join lobby <span>↗</span>
        </button>
        <div className="card-note">
          No account needed. Share your room code to play.
          <br />
          Or add a bot commander for a practice campaign.
        </div>
      </section>

      <div className="landing-footer">
        <span>A TABLETOP WAR GAME, BUILT FOR THE BROWSER.</span>
        <span>EST. 25–35 MIN / MATCH</span>
      </div>
    </main>
  );
}
