import React, { useEffect, useRef, useState } from 'react';
import { sampleBattle } from './battleMotion';
import { calcEconomy } from './combat';
import { BattlefieldMap } from './components/BattlefieldMap';
import { CampaignBar } from './components/CampaignBar';
import { DiceStage } from './components/DiceStage';
import { LandingView } from './components/LandingView';
import { LeftPanel } from './components/LeftPanel';
import { ExitModal, ManualModal } from './components/Modals';
import { RightPanel } from './components/RightPanel';
import { TopBar } from './components/TopBar';
import { websocketURL } from './connection';
import type { Game, Territory, Unit } from './types';

function readStoredSession(): { code: string; player: string; token: string } | null {
  try {
    const raw = sessionStorage.getItem('iron-session');
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed.code === 'string' && typeof parsed.player === 'string' && typeof parsed.token === 'string') {
      return parsed;
    }
  } catch {
    try {
      sessionStorage.removeItem('iron-session');
    } catch {
      // ignore storage failure
    }
  }
  return null;
}

export function App() {
  const [game, setGame] = useState<Game | null>(null);
  const [player, setPlayer] = useState('');
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState('');
  const [name, setName] = useState('');
  const [code, setCode] = useState('');
  const [selected, setSelected] = useState<number | null>(null);
  const [army, setArmy] = useState('');
  const [mode, setMode] = useState('move');
  const [amount, setAmount] = useState(1);
  const [now, setNow] = useState(Date.now());
  const [help, setHelp] = useState(false);
  const [copied, setCopied] = useState(false);
  const [reducedMotion, setReducedMotion] = useState(() => matchMedia('(prefers-reduced-motion: reduce)').matches);
  const [exitOpen, setExitOpen] = useState(false);
  const [exiting, setExiting] = useState(false);
  const [disconnectStarted, setDisconnectStarted] = useState(0);
  const [phaseNotice, setPhaseNotice] = useState('');
  const [scrubPaused, setScrubPaused] = useState(false);
  const [scrubFrame, setScrubFrame] = useState<number | null>(null);
  const [scrubT, setScrubT] = useState(1);
  const [rightTab, setRightTab] = useState<'sector' | 'report'>('sector');

  const phaseStarted = useRef(Date.now());
  const exitCommand = useRef('');
  const lastPhase = useRef('');
  const copyTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const playbackOrigin = useRef<{ progress: number; time: number } | null>(null);
  const socket = useRef<WebSocket | null>(null);
  const session = useRef<{ code: string; player: string; token: string } | null>(readStoredSession());

  function resetClientSession(clearError: boolean) {
    session.current = null;
    try {
      sessionStorage.removeItem('iron-session');
    } catch {
      // ignore storage failure
    }
    setGame(null);
    setPlayer('');
    setSelected(null);
    setArmy('');
    setExitOpen(false);
    setExiting(false);
    setRightTab('sector');
    setScrubPaused(false);
    setScrubFrame(null);
    setScrubT(1);
    playbackOrigin.current = null;
    lastPhase.current = '';
    setPhaseNotice('');
    exitCommand.current = '';
    if (clearError) setError('');
  }

  function send(action: string, data: Record<string, unknown> = {}) {
    if (socket.current?.readyState !== WebSocket.OPEN) {
      setError('Connecting to the command server…');
      return;
    }
    setError('');
    const id = crypto.randomUUID();
    if (action === 'leave') {
      exitCommand.current = id;
      setExiting(true);
    }
    socket.current.send(JSON.stringify({ id, action, ...data }));
  }

  useEffect(() => {
    let stopped = false;
    let attempt = 0;
    let timer: ReturnType<typeof setTimeout>;
    function connect() {
      if (stopped) return;
      let ws: WebSocket;
      try {
        ws = new WebSocket(websocketURL());
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Invalid game server address');
        return;
      }
      socket.current = ws;
      ws.onopen = () => {
        attempt = 0;
        setDisconnectStarted(0);
        setConnected(true);
        if (session.current) {
          ws.send(JSON.stringify({ id: crypto.randomUUID(), action: 'resume', ...session.current }));
        }
      };
      ws.onmessage = e => {
        let m: Record<string, any>;
        try {
          m = JSON.parse(e.data);
        } catch {
          return;
        }
        if (m.type === 'state') setGame(m.game);
        if (m.type === 'left') {
          resetClientSession(true);
        }
        if (m.type === 'session') {
          session.current = { code: m.code, player: m.player, token: m.token };
          try {
            sessionStorage.setItem('iron-session', JSON.stringify(session.current));
          } catch {
            // ignore storage quota errors
          }
          setPlayer(m.player);
        }
        if (m.type === 'error') {
          if (m.id === exitCommand.current) {
            setExiting(false);
            exitCommand.current = '';
          }
          setError(m.message);
          if (m.message === 'Session expired' || m.message === 'Lobby not found') {
            resetClientSession(false);
          }
        }
      };
      ws.onclose = () => {
        if (session.current) setDisconnectStarted(start => start || Date.now());
        setConnected(false);
        setExiting(false);
        if (!stopped) {
          timer = setTimeout(connect, Math.min(30000, 1000 * 2 ** Math.min(attempt++, 5)) + Math.random() * 500);
        }
      };
    }
    connect();
    return () => {
      stopped = true;
      clearTimeout(timer);
      if (copyTimer.current) clearTimeout(copyTimer.current);
      socket.current?.close();
    };
  }, []);

  useEffect(() => {
    const ts = Date.now();
    phaseStarted.current = ts;
    setNow(ts);
    setArmy('');
    setMode('move');
    setScrubPaused(false);
    setScrubFrame(null);
    setScrubT(1);
    playbackOrigin.current = null;
    if (game?.phase) {
      if (lastPhase.current && lastPhase.current !== game.phase) {
        setPhaseNotice(lastPhase.current.toUpperCase() + ' COMPLETE → ' + game.phase.toUpperCase());
      }
      if (game.phase !== 'gather' && game.phase !== 'plan') {
        setRightTab('sector');
      }
      lastPhase.current = game.phase;
    }
    const timeout = setTimeout(() => setPhaseNotice(''), 3500);
    return () => clearTimeout(timeout);
  }, [game?.phase, game?.round]);

  const replayDuration = game?.simulationDuration || 12000;
  const totalFrames = Math.max(1, game?.timeline?.length || 1);
  const liveProgress =
    game?.phase === 'simulate'
      ? playbackOrigin.current
        ? Math.max(0, Math.min(1, playbackOrigin.current.progress + (now - playbackOrigin.current.time) / replayDuration))
        : Math.max(0, Math.min(1, (now - phaseStarted.current) / replayDuration))
      : 0;
  const simAnimating = game?.phase === 'simulate' && !scrubPaused && liveProgress < 1;

  const disconnectTimers = game?.players.map(p => p.disconnectedUntil || 0).join(',') || '';
  useEffect(() => {
    if (!game) return;
    let clock: ReturnType<typeof setTimeout>;
    function step() {
      const t = Date.now();
      setNow(t);
      const gathering = game!.phase === 'gather' && t - phaseStarted.current <= 5200;
      const grace = Boolean(disconnectStarted && !connected) || game!.players.some(p => p.disconnectedUntil > t);
      const simStillPlaying =
        game!.phase === 'simulate' &&
        !scrubPaused &&
        (playbackOrigin.current
          ? playbackOrigin.current.progress + (t - playbackOrigin.current.time) / replayDuration < 1
          : (t - phaseStarted.current) / replayDuration < 1);
      const fast = simStillPlaying || (gathering && !reducedMotion);
      const rolling = game!.phase === 'roll';
      if (fast || rolling || gathering || game!.deadline || grace) {
        clock = setTimeout(step, fast ? 33 : rolling ? 150 : 500);
      }
    }
    step();
    return () => clearTimeout(clock);
  }, [
    game?.phase,
    game?.round,
    game?.deadline,
    replayDuration,
    reducedMotion,
    disconnectTimers,
    connected,
    disconnectStarted,
    scrubPaused,
    simAnimating,
  ]);

  const isScrubbed = game?.phase === 'simulate' && scrubPaused && scrubFrame !== null;
  const frameIndex = isScrubbed
    ? Math.max(0, Math.min(totalFrames - 1, scrubFrame))
    : Math.min(totalFrames - 1, Math.floor(liveProgress * totalFrames));
  const frameT = isScrubbed
    ? scrubT
    : liveProgress >= 1
      ? 1
      : Math.max(0, Math.min(1, liveProgress * totalFrames - frameIndex));
  const simulationProgress = isScrubbed
    ? Math.max(0, Math.min(1, (frameIndex + frameT) / totalFrames))
    : liveProgress;

  const frame = game?.phase === 'simulate' ? game.timeline?.[frameIndex] : undefined;
  const beforeFrame = game?.phase === 'simulate' ? game.timeline?.[Math.max(0, frameIndex - 1)] : undefined;
  const motion =
    frame && beforeFrame && game ? sampleBattle(beforeFrame, frame, frameT, game.territories, reducedMotion) : undefined;
  const displayUnits = motion?.units || game?.units || [];
  const displayTerritories =
    game?.territories.map(t => (frame ? { ...t, owner: motion ? motion.owners[t.id] : frame.owners[t.id] } : t)) || [];
  const previousFrame = beforeFrame;
  const changedUnits =
    frame?.units.filter(u => {
      const old = previousFrame?.units.find(v => v.id === u.id);
      return frameT >= 0.62 && old && (old.health !== u.health || old.count !== u.count);
    }) || [];
  const destroyedSet = new Set(frame?.destroyed || []);
  const destroyedUnits =
    frameT >= 0.62 ? previousFrame?.units.filter(u => destroyedSet.has(u.id)) || [] : [];

  const me = game?.players.find(p => p.id === player);
  const territory = displayTerritories.find(t => t.id === selected);
  const unit = displayUnits.find(u => u.id === army);
  const phase = game?.phase || 'lobby';
  const seconds = game?.deadline ? Math.max(0, Math.ceil((game.deadline - now) / 1000)) : 0;
  const full = ['draft', 'lobby', 'roll', 'capital', 'finished'].includes(phase) || Boolean(me?.eliminated);
  const locked = Boolean(me?.ready || me?.eliminated || !connected);
  const activePlayers = game?.players.filter(p => !p.eliminated) || [];
  const readyCount = activePlayers.filter(p => p.ready).length;
  const draftTurnPlayer = phase === 'draft' ? game?.players.find(p => p.id === game.turn) : undefined;
  const isMyDraftTurn = Boolean(draftTurnPlayer && draftTurnPlayer.id === player);
  const draftPickNumber = draftTurnPlayer
    ? Math.min(4, (game?.territories.filter(t => t.owner === draftTurnPlayer.id).length || 0) + 1)
    : 0;

  const visibleEvents = phase === 'simulate' ? frame?.events || [] : [];
  const frameEvents =
    phase === 'simulate' ? (frameIndex === 0 ? visibleEvents : visibleEvents.slice(previousFrame?.events.length || 0)) : [];
  const currentEvent = frameEvents[frameEvents.length - 1];

  const econ = game
    ? calcEconomy(
        game,
        player,
        phase === 'simulate' ? displayTerritories : undefined,
        phase === 'simulate' ? displayUnits : undefined,
      )
    : {
        income: { humans: 0, food: 0, iron: 0, currency: 0 },
        foodUpkeep: 0,
        foodNet: 0,
        hasOwnCapital: false,
        ownCapitalLost: false,
        territoriesHeld: 0,
        capitalsControlled: 0,
        vp: 0,
        hasBorderArmies: false,
      };

  function chooseArmy(u: Unit) {
    setArmy(u.id);
    setMode(u.type === 'artillery' ? 'bombard' : 'move');
    setRightTab('sector');
  }

  function clickTerritory(t: Territory) {
    setRightTab('sector');
    if (phase === 'capital') {
      setSelected(t.id);
      return;
    }
    if (phase === 'draft' && game?.turn === player && !t.owner) {
      send('claim', { target: t.id });
      setSelected(t.id);
      return;
    }
    if (phase === 'plan' && !locked && unit && unit.owner === player && unit.territory !== t.id) {
      send('order', { unit: unit.id, kind: mode, target: t.id });
      setArmy('');
      return;
    }
    setSelected(t.id);
    setArmy('');
  }

  function handleTogglePause() {
    if (scrubPaused) {
      const ts = Date.now();
      const atEnd = frameIndex >= totalFrames - 1 && frameT >= 1;
      const startProgress = atEnd ? 0 : Math.max(0, Math.min(0.999, (frameIndex + frameT) / totalFrames));
      playbackOrigin.current = { progress: startProgress, time: ts };
      setNow(ts);
      setScrubFrame(null);
      setScrubPaused(false);
    } else {
      setScrubFrame(frameIndex);
      setScrubT(frameT);
      setScrubPaused(true);
    }
  }

  function handleStepFrame(delta: number) {
    const next = Math.max(0, Math.min(totalFrames - 1, frameIndex + delta));
    setScrubFrame(next);
    setScrubT(1);
    setScrubPaused(true);
  }

  function handleSelectFrame(idx: number) {
    setScrubFrame(Math.max(0, Math.min(totalFrames - 1, idx)));
    setScrubT(1);
    setScrubPaused(true);
  }

  function handleCopyCode() {
    if (!game) return;
    const promise = navigator.clipboard?.writeText(game.code);
    if (!promise) {
      setError('Clipboard unavailable in this browser context');
      return;
    }
    promise
      .then(() => {
        setCopied(true);
        if (copyTimer.current) clearTimeout(copyTimer.current);
        copyTimer.current = setTimeout(() => setCopied(false), 1800);
      })
      .catch(() => setError('Could not copy room code to clipboard'));
  }

  return (
    <div className="app">
      <TopBar
        connected={connected}
        game={game}
        copied={copied}
        onToggleHelp={() => setHelp(h => !h)}
        onOpenExit={() => setExitOpen(true)}
        onCopyCode={handleCopyCode}
      />

      {exitOpen && game && (
        <ExitModal
          connected={connected}
          exiting={exiting}
          onClose={() => setExitOpen(false)}
          onConfirm={() => send('leave')}
        />
      )}

      {phaseNotice && (
        <div className="phase-notice" role="alert">
          {phaseNotice}
        </div>
      )}

      {game?.phase === 'roll' && <DiceStage game={game} now={now} />}

      {!connected && game && disconnectStarted > 0 && (
        <div className="disconnect-banner" role="status">
          Connection lost — reconnecting. Seat grace remaining:{' '}
          {Math.max(0, Math.ceil((disconnectStarted + 300000 - now) / 1000 / 60))} min. Reconnect within five minutes to
          keep your armies and territory.
        </div>
      )}

      {error && (
        <div className="toast" role="alert">
          {error}
          <button onClick={() => setError('')}>×</button>
        </div>
      )}

      {help && <ManualModal onClose={() => setHelp(false)} />}

      {!game ? (
        <LandingView
          name={name}
          code={code}
          connected={connected}
          onNameChange={setName}
          onCodeChange={setCode}
          onCreate={() => send('create', { name })}
          onJoin={() => send('join', { name, code })}
        />
      ) : (
        <>
          <CampaignBar
            game={game}
            me={me}
            phase={phase}
            seconds={seconds}
            connected={connected}
            readyCount={readyCount}
            activeCount={activePlayers.length}
            draftTurnPlayer={draftTurnPlayer}
            isMyDraftTurn={isMyDraftTurn}
            draftPickNumber={draftPickNumber}
            onToggleReady={() => send(me?.ready ? 'unready' : 'ready')}
          />
          <div className="game-layout">
            <LeftPanel
              game={game}
              me={me}
              player={player}
              phase={phase}
              now={now}
              full={full}
              econ={econ}
              onAddBot={() => send('addBot')}
              onStart={() => send('start')}
            />
            <BattlefieldMap
              game={game}
              me={me}
              player={player}
              phase={phase}
              locked={locked}
              selected={selected}
              army={army}
              unit={unit}
              mode={mode}
              now={now}
              phaseStarted={phaseStarted.current}
              reducedMotion={reducedMotion}
              simulationProgress={simulationProgress}
              frameIndex={frameIndex}
              frame={frame}
              frameT={frameT}
              beforeFrame={beforeFrame}
              displayUnits={displayUnits}
              displayTerritories={displayTerritories}
              visibleEvents={visibleEvents}
              frameEvents={frameEvents}
              currentEvent={currentEvent}
              scrubPaused={scrubPaused}
              draftTurnPlayer={draftTurnPlayer}
              isMyDraftTurn={isMyDraftTurn}
              draftPickNumber={draftPickNumber}
              onTogglePause={handleTogglePause}
              onStepFrame={handleStepFrame}
              onSelectFrame={handleSelectFrame}
              onToggleReducedMotion={setReducedMotion}
              onClickTerritory={clickTerritory}
              onSelectArmy={(tId, u) => {
                setSelected(tId);
                chooseArmy(u);
              }}
            />
            <RightPanel
              game={game}
              me={me}
              player={player}
              phase={phase}
              connected={connected}
              exiting={exiting}
              locked={locked}
              econ={econ}
              territory={territory}
              unit={unit}
              army={army}
              mode={mode}
              amount={amount}
              rightTab={rightTab}
              frame={frame}
              previousFrame={previousFrame}
              displayUnits={displayUnits}
              displayTerritories={displayTerritories}
              visibleEvents={visibleEvents}
              currentEvent={currentEvent}
              changedUnits={changedUnits}
              destroyedUnits={destroyedUnits}
              onSetRightTab={setRightTab}
              onChooseArmy={chooseArmy}
              onClearArmy={() => setArmy('')}
              onSetMode={setMode}
              onSetAmount={setAmount}
              onSend={send}
            />
          </div>
        </>
      )}
    </div>
  );
}
