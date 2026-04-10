import React, { useContext, useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { WebSocketContext } from '../WebsocketContext';
import '../App.css';

const POPULAR_SYMBOLS = [
  'BTC/USD', 'ETH/USD', 'SOL/USD',
  'AAPL', 'MSFT', 'GOOGL', 'AMZN', 'TSLA',
  'META', 'NVDA', 'AMD', 'NFLX', 'SPY',
];

/* -------------------------------------------------------
   Utility helpers
   ------------------------------------------------------- */
function formatVolume(vol) {
  if (!vol) return '—';
  if (vol >= 1e6) return (vol / 1e6).toFixed(2) + 'M';
  if (vol >= 1e3) return (vol / 1e3).toFixed(1) + 'K';
  return String(vol);
}

function formatPrice(price) {
  if (price === undefined || price === null) return '—';
  return '$' + price.toFixed(4);
}

const CHART_TF = [
  { label: '1S', ms: 0 },
  { label: '1M', ms: 60_000 },
  { label: '5M', ms: 300_000 },
  { label: '15M', ms: 900_000 },
];

function buildCandles(ticks, ms) {
  if (!ticks || ticks.length === 0) return [];
  if (ms === 0) {
    return ticks.map(t => ({
      time: new Date(t.timestamp).getTime(),
      open: t.price, high: t.price, low: t.price, close: t.price,
      volume: t.volume || 0, side: t.side,
    }));
  }
  const buckets = {};
  ticks.forEach(t => {
    const bucket = Math.floor(new Date(t.timestamp).getTime() / ms) * ms;
    if (!buckets[bucket]) {
      buckets[bucket] = { time: bucket, open: t.price, high: t.price, low: t.price, close: t.price, volume: t.volume || 0 };
    } else {
      buckets[bucket].high = Math.max(buckets[bucket].high, t.price);
      buckets[bucket].low = Math.min(buckets[bucket].low, t.price);
      buckets[bucket].close = t.price;
      buckets[bucket].volume += (t.volume || 0);
    }
  });
  return Object.values(buckets).sort((a, b) => a.time - b.time);
}

/* -------------------------------------------------------
   Sparkline — lightweight SVG line
   ------------------------------------------------------- */
function Sparkline({ data, width = 100, height = 28, color = '#38BDF8' }) {
  if (!data || data.length < 2) {
    return <svg width={width} height={height} style={{ display: 'block', opacity: 0.15 }} />;
  }
  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  const points = data.map((v, i) => {
    const x = (i / (data.length - 1)) * width;
    const y = height - ((v - min) / range) * (height - 4) - 2;
    return `${x},${y}`;
  }).join(' ');

  return (
    <svg width={width} height={height} style={{ display: 'block' }}>
      <polyline
        points={points}
        fill="none"
        stroke={color}
        strokeWidth="1.5"
        strokeLinejoin="round"
        strokeLinecap="round"
      />
    </svg>
  );
}

/* -------------------------------------------------------
   Ticker Row
   ------------------------------------------------------- */
function TickerRow({ data, isSelected, onClick, onUnsubscribe, priceHistory }) {
  const [flash, setFlash] = useState('');
  const change = data.price - data.prevPrice;
  const direction = change > 0 ? 'up' : change < 0 ? 'down' : 'flat';
  const pctChange = data.prevPrice && data.prevPrice !== 0
    ? ((change / data.prevPrice) * 100).toFixed(2)
    : '0.00';

  useEffect(() => {
    if (direction !== 'flat') {
      setFlash(direction);
      const timer = setTimeout(() => setFlash(''), 400);
      return () => clearTimeout(timer);
    }
  }, [data.updatedAt, direction]);

  return (
    <tr
      className={`ticker-row ${flash ? `row-flash-${flash}` : ''} ${isSelected ? 'row-selected' : ''}`}
      onClick={onClick}
    >
      <td className="cell-symbol">{data.symbol}</td>
      <td className={`cell-price price-${direction}`}>{formatPrice(data.price)}</td>
      <td className="cell-spark">
        <Sparkline
          data={priceHistory}
          color={direction === 'up' ? '#4ADE80' : direction === 'down' ? '#F87171' : '#38BDF8'}
        />
      </td>
      <td className="cell-vol">{formatVolume(data.volume)}</td>
      <td className={`cell-change change-${direction}`}>
        {direction === 'up' ? '+' : ''}{pctChange}%
      </td>
      <td className="cell-side">
        <span className={`side-tag ${data.side || ''}`}>{data.side || '—'}</span>
      </td>
      <td className="cell-time">
        {data.timestamp ? new Date(data.timestamp).toLocaleTimeString() : '—'}
      </td>
      <td className="cell-unsub">
        <button className="unsub-btn" onClick={e => { e.stopPropagation(); onUnsubscribe(data.symbol); }}>×</button>
      </td>
    </tr>
  );
}

/* -------------------------------------------------------
   Chart Panel (slide-out, right side)
   ------------------------------------------------------- */
function ChartPanel({ symbol, tickerData, tickHistory, onClose }) {
  const [tfIdx, setTfIdx] = useState(0);
  const [hoverCandle, setHoverCandle] = useState(null);
  const [crosshairX, setCrosshairX] = useState(null);
  const svgRef = useRef(null);

  const tf = CHART_TF[tfIdx];
  const isLine = tf.ms === 0;
  const candles = useMemo(() => buildCandles(tickHistory || [], tf.ms), [tickHistory, tf.ms]);

  // SVG dimensions
  const W = 560, priceH = 210, volH = 60, GAP = 14, totalH = priceH + GAP + volH;
  const padL = 62, padR = 12, padT = 10, padB = 24;

  // Price scale
  const rawMin = candles.length > 0 ? Math.min(...candles.map(c => c.low)) : 0;
  const rawMax = candles.length > 0 ? Math.max(...candles.map(c => c.high)) : 1;
  const pPad = (rawMax - rawMin) * 0.06 || 0.01;
  const scaleMin = rawMin - pPad;
  const scaleMax = rawMax + pPad;
  const pRange = scaleMax - scaleMin;
  const py = (p) => padT + (1 - (p - scaleMin) / pRange) * (priceH - padT - padB);

  // Volume scale
  const volMax = candles.length > 0 ? (Math.max(...candles.map(c => c.volume)) || 1) : 1;

  // Candle geometry
  const n = candles.length || 1;
  const slotW = (W - padL - padR) / n;
  const candleW = Math.max(1, Math.min(slotW * 0.65, 12));
  const cx = (i) => padL + (i + 0.5) * slotW;

  const handleMouseMove = useCallback((e) => {
    if (!svgRef.current || candles.length === 0) return;
    const rect = svgRef.current.getBoundingClientRect();
    const svgX = ((e.clientX - rect.left) / rect.width) * W;
    const idx = Math.max(0, Math.min(n - 1, Math.round((svgX - padL) / slotW - 0.5)));
    setCrosshairX(cx(idx));
    setHoverCandle({ ...candles[idx], idx });
  }, [candles, slotW, n]);

  const handleMouseLeave = useCallback(() => {
    setCrosshairX(null);
    setHoverCandle(null);
  }, []);

  const gridPrices = useMemo(() => {
    if (candles.length === 0) return [];
    return [1, 2, 3].map(i => scaleMin + i * (pRange / 4));
  }, [candles.length, scaleMin, pRange]);

  const change = tickerData ? (tickerData.price - tickerData.prevPrice) : 0;
  const direction = change > 0 ? 'up' : change < 0 ? 'down' : 'flat';
  const lineColor = direction === 'down' ? '#F87171' : '#4ADE80';

  // Line chart path (1S mode)
  let linePath = '', areaPath = '';
  if (isLine && candles.length > 1) {
    const pts = candles.map((c, i) => ({ x: cx(i), y: py(c.close) }));
    linePath = pts.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ');
    areaPath = linePath + ` L${pts[pts.length - 1].x.toFixed(1)},${priceH - padB} L${pts[0].x.toFixed(1)},${priceH - padB} Z`;
  }

  const fmtAxisPrice = (p) => p >= 1000 ? p.toFixed(1) : p.toFixed(4);

  return (
    <div className="chart-panel">
      <div className="chart-header">
        <div className="chart-title">
          <span className="chart-symbol">{symbol}</span>
          <span className={`chart-price price-${direction}`}>{formatPrice(tickerData?.price)}</span>
          <span className={`chart-change change-${direction}`}>
            {change >= 0 ? '+' : ''}{change.toFixed(4)}
          </span>
        </div>
        <button className="chart-close" onClick={onClose}>ESC</button>
      </div>

      <div className="chart-timeframes">
        {CHART_TF.map((t, i) => (
          <button
            key={t.label}
            className={`tf-btn ${tfIdx === i ? 'tf-active' : ''}`}
            onClick={() => setTfIdx(i)}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="chart-canvas">
        {candles.length < 2 ? (
          <div className="chart-empty">ACCUMULATING DATA...</div>
        ) : (
          <>
            <svg
              ref={svgRef}
              width="100%"
              height="100%"
              viewBox={`0 0 ${W} ${totalH}`}
              preserveAspectRatio="none"
              onMouseMove={handleMouseMove}
              onMouseLeave={handleMouseLeave}
              style={{ display: 'block', cursor: 'crosshair' }}
            >
              <defs>
                <linearGradient id={`ag-${symbol}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={lineColor} stopOpacity="0.2" />
                  <stop offset="100%" stopColor={lineColor} stopOpacity="0" />
                </linearGradient>
              </defs>

              {/* Price grid */}
              {gridPrices.map((p, i) => (
                <g key={i}>
                  <line x1={padL} y1={py(p)} x2={W - padR} y2={py(p)} stroke="#1E2545" strokeWidth="1" />
                  <text x={padL - 4} y={py(p) + 3.5} textAnchor="end" fill="#475569" fontSize="9" fontFamily="JetBrains Mono, monospace">
                    {fmtAxisPrice(p)}
                  </text>
                </g>
              ))}

              {/* Volume divider */}
              <line x1={padL} y1={priceH + GAP / 2} x2={W - padR} y2={priceH + GAP / 2} stroke="#1E2545" strokeWidth="1" strokeDasharray="3,5" />
              <text x={padL - 4} y={priceH + GAP / 2 + 3} textAnchor="end" fill="#334155" fontSize="8" fontFamily="JetBrains Mono, monospace">VOL</text>

              {/* Line chart (1S) */}
              {isLine && (
                <>
                  <path d={areaPath} fill={`url(#ag-${symbol})`} />
                  <path d={linePath} fill="none" stroke={lineColor} strokeWidth="1.5" strokeLinejoin="round" />
                  <circle cx={cx(candles.length - 1)} cy={py(candles[candles.length - 1].close)} r="3" fill={lineColor} />
                </>
              )}

              {/* Candlesticks (1M / 5M / 15M) */}
              {!isLine && candles.map((c, i) => {
                const isUp = c.close >= c.open;
                const color = isUp ? '#4ADE80' : '#F87171';
                const bodyTop = py(Math.max(c.open, c.close));
                const bodyH = Math.max(1, py(Math.min(c.open, c.close)) - bodyTop);
                const x = cx(i);
                return (
                  <g key={i} opacity={hoverCandle?.idx === i ? 1 : 0.85}>
                    <line x1={x} y1={py(c.high)} x2={x} y2={py(c.low)} stroke={color} strokeWidth="1" />
                    <rect x={x - candleW / 2} y={bodyTop} width={candleW} height={bodyH}
                      fill={isUp ? color : 'none'} stroke={color} strokeWidth="1" />
                  </g>
                );
              })}

              {/* Volume bars */}
              {candles.map((c, i) => {
                const barH = (c.volume / volMax) * (volH - 8);
                return (
                  <rect key={i}
                    x={cx(i) - candleW / 2} y={totalH - barH}
                    width={candleW} height={barH}
                    fill={c.close >= c.open ? 'rgba(74,222,128,0.4)' : 'rgba(248,113,113,0.4)'}
                  />
                );
              })}

              {/* Crosshair */}
              {crosshairX !== null && hoverCandle && (
                <>
                  <line x1={crosshairX} y1={padT} x2={crosshairX} y2={totalH} stroke="#38BDF8" strokeWidth="1" strokeDasharray="3,3" opacity="0.5" />
                  <line x1={padL} y1={py(hoverCandle.close)} x2={W - padR} y2={py(hoverCandle.close)} stroke="#38BDF8" strokeWidth="1" strokeDasharray="3,3" opacity="0.35" />
                  <rect x={0} y={py(hoverCandle.close) - 8} width={padL - 2} height={16} fill="#12162B" />
                  <text x={padL - 4} y={py(hoverCandle.close) + 3.5} textAnchor="end" fill="#38BDF8" fontSize="9" fontFamily="JetBrains Mono, monospace">
                    {fmtAxisPrice(hoverCandle.close)}
                  </text>
                </>
              )}
            </svg>

            {/* Tooltip */}
            {hoverCandle && crosshairX !== null && (
              <div
                className="chart-tooltip"
                style={{
                  left: `${(crosshairX / W) * 100}%`,
                  transform: crosshairX > W * 0.55 ? 'translateX(calc(-100% - 10px))' : 'translateX(10px)',
                }}
              >
                <div className="tt-time">{new Date(hoverCandle.time).toLocaleTimeString()}</div>
                {!isLine && (
                  <>
                    <div className="tt-row"><span>O</span><span>{formatPrice(hoverCandle.open)}</span></div>
                    <div className="tt-row"><span>H</span><span className="price-up">{formatPrice(hoverCandle.high)}</span></div>
                    <div className="tt-row"><span>L</span><span className="price-down">{formatPrice(hoverCandle.low)}</span></div>
                  </>
                )}
                <div className="tt-row"><span>C</span><span>{formatPrice(hoverCandle.close)}</span></div>
                <div className="tt-row tt-vol"><span>VOL</span><span>{formatVolume(hoverCandle.volume)}</span></div>
              </div>
            )}
          </>
        )}
      </div>

      <div className="chart-stats-grid">
        <div className="chart-stat">
          <span className="chart-stat-label">VOLUME</span>
          <span className="chart-stat-value">{formatVolume(tickerData?.volume)}</span>
        </div>
        <div className="chart-stat">
          <span className="chart-stat-label">SIDE</span>
          <span className={`chart-stat-value ${tickerData?.side || ''}`}>
            {tickerData?.side?.toUpperCase() || '—'}
          </span>
        </div>
        <div className="chart-stat">
          <span className="chart-stat-label">HIGH</span>
          <span className="chart-stat-value">
            {candles.length > 0 ? formatPrice(Math.max(...candles.map(c => c.high))) : '—'}
          </span>
        </div>
        <div className="chart-stat">
          <span className="chart-stat-label">LOW</span>
          <span className="chart-stat-value">
            {candles.length > 0 ? formatPrice(Math.min(...candles.map(c => c.low))) : '—'}
          </span>
        </div>
      </div>
    </div>
  );
}

/* -------------------------------------------------------
   System Telemetry Overlay
   ------------------------------------------------------- */
function useUptime(connectedAt) {
  const [uptime, setUptime] = useState(0);
  useEffect(() => {
    if (!connectedAt) return;
    const tick = () => setUptime(Math.floor((Date.now() - connectedAt) / 1000));
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [connectedAt]);
  return uptime;
}

function fmtUptime(s) {
  if (s < 60) return `${s}s`;
  if (s < 3600) return `${Math.floor(s / 60)}m ${s % 60}s`;
  return `${Math.floor(s / 3600)}h ${Math.floor((s % 3600) / 60)}m`;
}

function fmtSince(ts) {
  if (!ts) return '—';
  const s = Math.floor((Date.now() - ts) / 1000);
  if (s < 5) return 'just now';
  if (s < 60) return `${s}s ago`;
  return `${Math.floor(s / 60)}m ago`;
}

function TelemetryOverlay({ onClose }) {
  const {
    connectionStatus, throughput, throughputHistory,
    latency, connectedAt, reconnectCount, lastTickAt, symbolTickCounts,
  } = useContext(WebSocketContext);
  const [, setTick] = useState(0);
  const uptime = useUptime(connectedAt);

  // Rerender every second so "last tick" stays fresh
  useEffect(() => {
    const id = setInterval(() => setTick(t => t + 1), 1000);
    return () => clearInterval(id);
  }, []);

  const topSymbols = Object.entries(symbolTickCounts || {})
    .sort((a, b) => b[1] - a[1])
    .slice(0, 6);

  const latencyColor = latency === null ? 'var(--text-dim)'
    : latency < 50 ? 'var(--up)' : latency < 150 ? '#FCD34D' : 'var(--down)';

  return (
    <div className="telemetry-overlay" onClick={onClose}>
      <div className="telemetry-modal" onClick={e => e.stopPropagation()}>
        <div className="telemetry-header">
          <span className="telemetry-title">SYSTEM TELEMETRY</span>
          <button className="telemetry-close" onClick={onClose}>ESC</button>
        </div>

        {/* Connection stats row */}
        <div className="telemetry-stats-row">
          <div className="tstat">
            <span className="tstat-label">STATUS</span>
            <span className={`tstat-value ${connectionStatus === 'CONNECTED' ? 'val-up' : 'val-down'}`}>
              {connectionStatus}
            </span>
          </div>
          <div className="tstat">
            <span className="tstat-label">TOTAL TICKS</span>
            <span className="tstat-value">{Object.values(symbolTickCounts || {}).reduce((a, b) => a + b, 0)}</span>
          </div>
          <div className="tstat">
            <span className="tstat-label">RTT LATENCY</span>
            <span className="tstat-value" style={{ color: latencyColor }}>
              {latency !== null ? `${latency}ms` : '—'}
            </span>
          </div>
          <div className="tstat">
            <span className="tstat-label">SESSION UP</span>
            <span className="tstat-value">{connectedAt ? fmtUptime(uptime) : '—'}</span>
          </div>
          <div className="tstat">
            <span className="tstat-label">RECONNECTS</span>
            <span className="tstat-value">{reconnectCount > 0 ? reconnectCount - 1 : '—'}</span>
          </div>
          <div className="tstat">
            <span className="tstat-label">LAST TICK</span>
            <span className="tstat-value">{fmtSince(lastTickAt)}</span>
          </div>
        </div>

        {/* Ingestion rate chart */}
        <div className="telemetry-throughput">
          <div className="throughput-header">
            <span>INGESTION RATE</span>
            <span className="throughput-value">{throughput} msg/s</span>
          </div>
          <div className="throughput-sparkline">
            <Sparkline data={throughputHistory} width={660} height={56} color="#38BDF8" />
          </div>
        </div>

        {/* Per-symbol tick counts */}
        <div className="telemetry-symbols">
          <div className="tsym-header">TICKS BY SYMBOL (SESSION)</div>
          {topSymbols.length === 0 ? (
            <div className="active-empty">No ticks received yet</div>
          ) : (
            <div className="tsym-grid">
              {topSymbols.map(([sym, count]) => {
                const maxCount = topSymbols[0][1] || 1;
                return (
                  <div key={sym} className="tsym-row">
                    <span className="tsym-name">{sym}</span>
                    <div className="tsym-bar-wrap">
                      <div className="tsym-bar" style={{ width: `${(count / maxCount) * 100}%` }} />
                    </div>
                    <span className="tsym-count">{count}</span>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

/* -------------------------------------------------------
   Symbol Filter Matrix (left drawer)
   ------------------------------------------------------- */
function FilterDrawer({ onSubscribe, onUnsubscribe, subscribedSymbols, availableSymbols, onClose }) {
  const [search, setSearch] = useState('');

  const list = availableSymbols.length > 0 ? availableSymbols : POPULAR_SYMBOLS;
  const filteredSymbols = list.filter(s =>
    s.toUpperCase().includes(search.toUpperCase())
  );

  return (
    <div className="filter-drawer">
      <div className="filter-header">
        <span className="filter-title">SYMBOL MATRIX</span>
        <button className="filter-close" onClick={onClose}>ESC</button>
      </div>

      <div className="filter-search">
        <input
          type="text"
          value={search}
          onChange={e => setSearch(e.target.value)}
          placeholder="SEARCH SYMBOL..."
          className="filter-input"
          autoFocus
        />
      </div>

      <div className="filter-symbols">
        {filteredSymbols.length === 0 ? (
          <div className="active-empty">No symbols match</div>
        ) : filteredSymbols.map(sym => {
          const isActive = subscribedSymbols.includes(sym);
          return (
            <button
              key={sym}
              className={`sym-btn ${isActive ? 'sym-active' : ''}`}
              onClick={() => isActive ? onUnsubscribe(sym) : onSubscribe(sym)}
            >
              {sym}
              {isActive && <span className="sym-remove">&times;</span>}
            </button>
          );
        })}
      </div>

      <div className="filter-active">
        <div className="active-header">ACTIVE STREAMS</div>
        {subscribedSymbols.length === 0 ? (
          <div className="active-empty">No active subscriptions</div>
        ) : (
          subscribedSymbols.map(sym => (
            <div key={sym} className="active-item">
              <span className="active-dot" />
              <span>{sym}</span>
              <button className="active-remove" onClick={() => onUnsubscribe(sym)}>&times;</button>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

/* -------------------------------------------------------
   Main Dashboard
   ------------------------------------------------------- */
function Dashboard() {
  const {
    tickerMap, priceHistory, tickHistory, sendMessage, setToken, removeTicker,
    connectionStatus, activeNode, throughput, throughputHistory,
  } = useContext(WebSocketContext);

  const [selectedSymbol, setSelectedSymbol] = useState(null);
  const [showTelemetry, setShowTelemetry] = useState(false);
  const [showFilter, setShowFilter] = useState(false);
  const [quickSymbol, setQuickSymbol] = useState('');
  const [availableSymbols, setAvailableSymbols] = useState([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const navigate = useNavigate();

  // Fetch known symbols from DB on mount
  useEffect(() => {
    const tok = localStorage.getItem('sc2-token');
    if (!tok) return;
    fetch('/api/tickers/symbols', { headers: { 'Authorization': `Bearer ${tok}` } })
      .then(r => r.json())
      .then(data => { if (Array.isArray(data)) setAvailableSymbols(data); })
      .catch(() => {});
  }, []);

  const tickers = Object.values(tickerMap).sort((a, b) => a.symbol.localeCompare(b.symbol));
  const subscribedSymbols = tickers.map(t => t.symbol);

  // Summary stats
  const totalVolume = tickers.reduce((sum, t) => sum + (t.volume || 0), 0);
  const topMover = tickers.reduce((best, t) => {
    const c = Math.abs(t.price - t.prevPrice);
    return c > (best.change || 0)
      ? { symbol: t.symbol, change: c, price: t.price, prevPrice: t.prevPrice }
      : best;
  }, { change: 0 });

  const handleSubscribe = useCallback((sym) => {
    const s = (sym || '').trim().toUpperCase();
    if (!s) return;
    sendMessage({ type: 'subscribe_ticker', symbol: s });
  }, [sendMessage]);

  const handleUnsubscribe = useCallback((sym) => {
    const s = (sym || '').trim().toUpperCase();
    if (!s) return;
    sendMessage({ type: 'unsubscribe_ticker', symbol: s });
    removeTicker(s);
    if (selectedSymbol === s) setSelectedSymbol(null);
  }, [sendMessage, removeTicker, selectedSymbol]);

  const handleQuickSubscribe = () => {
    handleSubscribe(quickSymbol);
    setQuickSymbol('');
  };

  const handleLogout = () => {
    localStorage.removeItem('sc2-token');
    setToken(null);
    navigate('/login');
  };

  // Keyboard shortcuts
  useEffect(() => {
    const onKey = (e) => {
      if (e.key === 'Escape') {
        if (showTelemetry) setShowTelemetry(false);
        else if (showFilter) setShowFilter(false);
        else if (selectedSymbol) setSelectedSymbol(null);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [showTelemetry, showFilter, selectedSymbol]);

  return (
    <div className="sc-dashboard">
      {/* ── Top Navigation ── */}
      <nav className="sc-nav">
        <div className="nav-brand">
          <span className="brand-text">STREAMCORE</span>
          <span className="brand-version">v2</span>
        </div>

        <div className="nav-telemetry">
          <div className="nav-status">
            <span className={`status-dot status-${connectionStatus.toLowerCase()}`} />
            <span className="status-label">{connectionStatus}</span>
          </div>
          <div className="nav-node" onClick={() => setShowTelemetry(true)}>
            <span className="node-label">NODE</span>
            <span className="node-value">{activeNode || '—'}</span>
          </div>
          <div className="nav-throughput" onClick={() => setShowTelemetry(true)}>
            <span className="tp-label">MSG/S</span>
            <span className="tp-value">{throughput}</span>
          </div>
        </div>

        <div className="nav-actions">
          <button className="nav-btn" onClick={() => setShowFilter(true)}>FILTER</button>
          <button className="nav-btn" onClick={() => setShowTelemetry(true)}>TELEMETRY</button>
          <button className="nav-btn nav-btn-logout" onClick={handleLogout}>LOGOUT</button>
        </div>
      </nav>

      {/* ── Main Content ── */}
      <div className={`sc-main ${selectedSymbol ? 'main-split' : ''}`}>
        {/* Ticker Grid */}
        <div className="sc-grid">
          <div className="quick-sub" style={{ position: 'relative' }}>
            <input
              type="text"
              value={quickSymbol}
              onChange={e => { setQuickSymbol(e.target.value.toUpperCase()); setShowSuggestions(true); }}
              onKeyDown={e => { if (e.key === 'Enter') { handleQuickSubscribe(); setShowSuggestions(false); } if (e.key === 'Escape') setShowSuggestions(false); }}
              onFocus={() => setShowSuggestions(true)}
              onBlur={() => setTimeout(() => setShowSuggestions(false), 150)}
              placeholder="ENTER SYMBOL..."
              className="quick-input"
            />
            <button className="quick-btn" onClick={() => { handleQuickSubscribe(); setShowSuggestions(false); }}>SUBSCRIBE</button>
            {showSuggestions && quickSymbol.length > 0 && (() => {
              const matches = availableSymbols.filter(s =>
                s.includes(quickSymbol) && !subscribedSymbols.includes(s)
              ).slice(0, 8);
              return matches.length > 0 ? (
                <div className="quick-suggestions">
                  {matches.map(s => (
                    <div key={s} className="suggestion-item" onMouseDown={() => { handleSubscribe(s); setQuickSymbol(''); setShowSuggestions(false); }}>
                      {s}
                    </div>
                  ))}
                </div>
              ) : null;
            })()}
          </div>

          {tickers.length === 0 ? (
            <div className="grid-empty">
              <div className="empty-icon">&#9671;</div>
              <div className="empty-text">NO ACTIVE STREAMS</div>
              <div className="empty-hint">Subscribe to a symbol or open the Filter Matrix</div>
            </div>
          ) : (
            <div className="table-wrap">
              <table className="sc-table">
                <thead>
                  <tr>
                    <th>SYMBOL</th>
                    <th>PRICE</th>
                    <th>TREND</th>
                    <th>VOLUME</th>
                    <th>CHG%</th>
                    <th>SIDE</th>
                    <th>TIME</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {tickers.map(t => (
                    <TickerRow
                      key={t.symbol}
                      data={t}
                      isSelected={selectedSymbol === t.symbol}
                      onClick={() => setSelectedSymbol(
                        selectedSymbol === t.symbol ? null : t.symbol
                      )}
                      onUnsubscribe={handleUnsubscribe}
                      priceHistory={priceHistory[t.symbol] || []}
                    />
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Chart Panel */}
        {selectedSymbol && tickerMap[selectedSymbol] && (
          <ChartPanel
            symbol={selectedSymbol}
            tickerData={tickerMap[selectedSymbol]}
            tickHistory={tickHistory[selectedSymbol] || []}
            onClose={() => setSelectedSymbol(null)}
          />
        )}
      </div>

      {/* ── Bottom Summary Bar ── */}
      <div className="sc-summary">
        <div className="summary-widget">
          <span className="summary-label">TOTAL VOLUME</span>
          <span className="summary-value">{formatVolume(totalVolume)}</span>
        </div>
        <div className="summary-widget">
          <span className="summary-label">ACTIVE FEEDS</span>
          <span className="summary-value">{tickers.length}</span>
        </div>
        <div className="summary-widget">
          <span className="summary-label">TOP MOVER</span>
          <span className={`summary-value ${topMover.symbol
            ? (topMover.price > topMover.prevPrice ? 'val-up' : 'val-down')
            : ''}`}
          >
            {topMover.symbol || '—'}
          </span>
        </div>
        <div className="summary-widget">
          <span className="summary-label">THROUGHPUT</span>
          <Sparkline data={throughputHistory} width={100} height={20} color="#38BDF8" />
        </div>
      </div>

      {/* ── Filter Drawer ── */}
      {showFilter && (
        <>
          <div className="drawer-backdrop" onClick={() => setShowFilter(false)} />
          <FilterDrawer
            onSubscribe={handleSubscribe}
            onUnsubscribe={handleUnsubscribe}
            subscribedSymbols={subscribedSymbols}
            availableSymbols={availableSymbols}
            onClose={() => setShowFilter(false)}
          />
        </>
      )}

      {/* ── Telemetry Overlay ── */}
      {showTelemetry && (
        <TelemetryOverlay onClose={() => setShowTelemetry(false)} />
      )}
    </div>
  );
}

export default Dashboard;
