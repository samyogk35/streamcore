import React, { useContext, useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { WebSocketContext } from '../WebsocketContext';
import '../App.css';

function TickerCard({ data }) {
  const [flash, setFlash] = useState('');
  const change = data.price - data.prevPrice;
  const direction = change > 0 ? 'up' : change < 0 ? 'down' : 'flat';

  useEffect(() => {
    if (direction !== 'flat') {
      setFlash(direction);
      const timer = setTimeout(() => setFlash(''), 600);
      return () => clearTimeout(timer);
    }
  }, [data.updatedAt, direction]);

  return (
    <div className={`ticker-card ${flash ? `ticker-flash-${flash}` : ''}`}>
      <div className="ticker-card-header">
        <span className="ticker-symbol">{data.symbol}</span>
        <span className="ticker-server">{data.server}</span>
      </div>
      <div className={`ticker-price ticker-${direction}`}>
        ${data.price?.toFixed(4)}
      </div>
      <div className="ticker-details">
        <span className={`ticker-change ticker-${direction}`}>
          {direction === 'up' && '+'}{change !== 0 ? change.toFixed(4) : '0.0000'}
        </span>
        <span className="ticker-volume">Vol: {data.volume}</span>
      </div>
      <div className="ticker-meta">
        <span>{data.timestamp ? new Date(data.timestamp).toLocaleTimeString() : '—'}</span>
        {data.side && <span className={`tick-side ${data.side}`}>{data.side}</span>}
      </div>
    </div>
  );
}

function Dashboard() {
  const { tickerMap, sendMessage, setToken, removeTicker } = useContext(WebSocketContext);
  const [symbol, setSymbol] = useState('');
  const navigate = useNavigate();

  const tickers = Object.values(tickerMap).sort((a, b) => a.symbol.localeCompare(b.symbol));

  const handleSubscribe = () => {
    const sym = symbol.trim().toUpperCase();
    if (!sym) return;
    sendMessage({ type: 'subscribe_ticker', symbol: sym });
  };

  const handleUnsubscribe = () => {
    const sym = symbol.trim().toUpperCase();
    if (!sym) return;
    sendMessage({ type: 'unsubscribe_ticker', symbol: sym });
    removeTicker(sym);
  };

  const handleLogout = () => {
    localStorage.removeItem('sc2-token');
    setToken(null);
    navigate('/login');
  };

  return (
    <div className="dashboard-container">
      <div className="dashboard-header">
        <span className="headings">StreamCore</span>
        <button className="logout-button" onClick={handleLogout}>Logout</button>
      </div>

      <div className="dashboard-controls">
        <input
          type="text"
          value={symbol}
          onChange={e => setSymbol(e.target.value.toUpperCase())}
          onKeyDown={e => e.key === 'Enter' && handleSubscribe()}
          placeholder="Symbol (e.g. AAPL)"
          className="symbol-input"
        />
        <button onClick={handleSubscribe}>Subscribe</button>
        <button onClick={handleUnsubscribe} className="unsub-button">Unsubscribe</button>
      </div>

      <div className="ticker-grid">
        {tickers.length === 0 ? (
          <div className="no-ticks">No tickers yet — subscribe to a symbol above.</div>
        ) : (
          tickers.map((t) => <TickerCard key={t.symbol} data={t} />)
        )}
      </div>
    </div>
  );
}

export default Dashboard;
