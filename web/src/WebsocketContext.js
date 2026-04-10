import React, { createContext, useRef, useEffect, useState, useCallback } from 'react';

export const WebSocketContext = createContext(null);

const MAX_PRICE_HISTORY = 60;
const MAX_TICK_HISTORY = 300;
const THROUGHPUT_INTERVAL = 1000;

const SUBSCRIPTIONS_KEY = 'sc2-subscriptions';

function loadSavedSubscriptions() {
    try {
        return JSON.parse(localStorage.getItem(SUBSCRIPTIONS_KEY)) || [];
    } catch {
        return [];
    }
}

export const WebSocketProvider = ({ children }) => {
    const ws = useRef(null);
    const [tickerMap, setTickerMap] = useState({});
    const [priceHistory, setPriceHistory] = useState({});
    const [token, setToken] = useState(localStorage.getItem('sc2-token'));
    const [connectionStatus, setConnectionStatus] = useState('DISCONNECTED');
    const [activeNode, setActiveNode] = useState(null);
    const [throughput, setThroughput] = useState(0);
    const [throughputHistory, setThroughputHistory] = useState([]);
    const [subscribedSymbols, setSubscribedSymbols] = useState(loadSavedSubscriptions);
    const [tickHistory, setTickHistory] = useState({});
    const [latency, setLatency] = useState(null);
    const [connectedAt, setConnectedAt] = useState(null);
    const [reconnectCount, setReconnectCount] = useState(0);
    const [lastTickAt, setLastTickAt] = useState(null);
    const [symbolTickCounts, setSymbolTickCounts] = useState({});
    const messageCount = useRef(0);

    useEffect(() => {
        const interval = setInterval(() => {
            const count = messageCount.current;
            messageCount.current = 0;
            setThroughput(count);
            setThroughputHistory(prev => {
                const next = [...prev, count];
                return next.length > 60 ? next.slice(-60) : next;
            });
        }, THROUGHPUT_INTERVAL);
        return () => clearInterval(interval);
    }, []);

    useEffect(() => {
        if (!token) return;
        setConnectionStatus('CONNECTING');

        const env = process.env.REACT_APP_NGINX_ENV || 'local';
        const host = process.env.REACT_APP_NGINX_HOST || 'localhost';
        const port = process.env.REACT_APP_NGINX_PORT || '7700';

        const serverUrl = env === 'local'
            ? `ws://${host}:${port}/ws/stream?token=${token}`
            : `wss://${host}/ws/stream?token=${token}`;

        ws.current = new WebSocket(serverUrl);

        ws.current.onopen = () => {
            setConnectionStatus('CONNECTED');
            setConnectedAt(Date.now());
            setReconnectCount(prev => prev + 1);
            const saved = loadSavedSubscriptions();
            saved.forEach(sym => {
                ws.current.send(JSON.stringify({ type: 'subscribe_ticker', symbol: sym }));
            });
        };
        ws.current.onclose = () => setConnectionStatus('DISCONNECTED');
        ws.current.onerror = () => setConnectionStatus('DISCONNECTED');

        ws.current.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);

                // Handle pong — measure RTT
                if (msg.type === 'pong') {
                    setLatency(Date.now() - msg.ts);
                    return;
                }

                const tick = msg;
                if (tick.symbol && tick.price !== undefined) {
                    messageCount.current++;
                    setLastTickAt(Date.now());
                    setSymbolTickCounts(prev => ({ ...prev, [tick.symbol]: (prev[tick.symbol] || 0) + 1 }));
                    if (tick.server) setActiveNode(tick.server);

                    setTickerMap((prev) => {
                        const existing = prev[tick.symbol];
                        return {
                            ...prev,
                            [tick.symbol]: {
                                ...tick,
                                prevPrice: existing ? existing.price : tick.price,
                                updatedAt: Date.now(),
                            },
                        };
                    });

                    setPriceHistory(prev => {
                        const history = prev[tick.symbol] || [];
                        const next = [...history, tick.price];
                        return {
                            ...prev,
                            [tick.symbol]: next.length > MAX_PRICE_HISTORY
                                ? next.slice(-MAX_PRICE_HISTORY)
                                : next,
                        };
                    });

                    setTickHistory(prev => {
                        const history = prev[tick.symbol] || [];
                        const entry = { price: tick.price, volume: tick.volume, timestamp: tick.timestamp, side: tick.side };
                        const next = [...history, entry];
                        return {
                            ...prev,
                            [tick.symbol]: next.length > MAX_TICK_HISTORY ? next.slice(-MAX_TICK_HISTORY) : next,
                        };
                    });
                }
            } catch (e) {
                console.error('Failed to parse market tick:', e);
            }
        };

        return () => {
            if (ws.current) ws.current.close();
        };
    }, [token]);

    // Ping every 5s to measure real RTT
    useEffect(() => {
        const interval = setInterval(() => {
            if (ws.current && ws.current.readyState === WebSocket.OPEN) {
                ws.current.send(JSON.stringify({ type: 'ping', ts: Date.now() }));
            }
        }, 5000);
        return () => clearInterval(interval);
    }, []);

    // On reconnect (including page reload), seed state for all persisted subscriptions
    useEffect(() => {
        if (connectionStatus !== 'CONNECTED') return;
        const saved = loadSavedSubscriptions();
        if (saved.length === 0) return;
        const tok = localStorage.getItem('sc2-token');
        saved.forEach(sym => {
            setTickerMap(prev => {
                if (prev[sym]) return prev;
                return {
                    ...prev,
                    [sym]: { symbol: sym, price: null, volume: null, side: null, timestamp: null, prevPrice: null, updatedAt: Date.now() },
                };
            });
            fetch(`/api/tickers/history?symbol=${encodeURIComponent(sym)}&limit=200`, {
                headers: { 'Authorization': `Bearer ${tok}` },
            })
                .then(r => r.json())
                .then(rows => {
                    if (!Array.isArray(rows) || rows.length === 0) return;
                    const sorted = [...rows].reverse();
                    setTickHistory(prev => ({
                        ...prev,
                        [sym]: sorted.map(r => ({ price: r.price, volume: r.volume, timestamp: r.timestamp, side: r.side })),
                    }));
                    setPriceHistory(prev => ({
                        ...prev,
                        [sym]: sorted.slice(-MAX_PRICE_HISTORY).map(r => r.price),
                    }));
                    const latest = rows[0];
                    setTickerMap(prev => ({
                        ...prev,
                        [sym]: { symbol: sym, price: latest.price, volume: latest.volume, side: latest.side, timestamp: latest.timestamp, prevPrice: rows.length > 1 ? rows[1].price : latest.price, updatedAt: Date.now() },
                    }));
                })
                .catch(() => {});
        });
    }, [connectionStatus]);

    const sendMessage = useCallback((msg) => {
        if (ws.current && ws.current.readyState === WebSocket.OPEN) {
            ws.current.send(JSON.stringify(msg));
        }
        if (msg.type === 'subscribe_ticker' && msg.symbol) {
            // Show placeholder row immediately while history loads
            setTickerMap(prev => {
                if (prev[msg.symbol]) return prev;
                return {
                    ...prev,
                    [msg.symbol]: {
                        symbol: msg.symbol,
                        price: null, volume: null, side: null,
                        timestamp: null, prevPrice: null, updatedAt: Date.now(),
                    },
                };
            });
            setSubscribedSymbols(prev => {
                if (prev.includes(msg.symbol)) return prev;
                const next = [...prev, msg.symbol];
                localStorage.setItem(SUBSCRIPTIONS_KEY, JSON.stringify(next));
                return next;
            });
            // Fetch historical trades and pre-seed charts
            const tok = localStorage.getItem('sc2-token');
            const sym = encodeURIComponent(msg.symbol);
            fetch(`/api/tickers/history?symbol=${sym}&limit=200`, {
                headers: { 'Authorization': `Bearer ${tok}` },
            })
                .then(r => r.json())
                .then(rows => {
                    if (!Array.isArray(rows) || rows.length === 0) return;
                    // rows are newest-first — reverse for chronological order
                    const sorted = [...rows].reverse();
                    setTickHistory(prev => ({
                        ...prev,
                        [msg.symbol]: sorted.map(r => ({
                            price: r.price, volume: r.volume,
                            timestamp: r.timestamp, side: r.side,
                        })),
                    }));
                    setPriceHistory(prev => ({
                        ...prev,
                        [msg.symbol]: sorted.slice(-MAX_PRICE_HISTORY).map(r => r.price),
                    }));
                    // Seed tickerMap with the latest historical tick
                    const latest = rows[0];
                    setTickerMap(prev => ({
                        ...prev,
                        [msg.symbol]: {
                            symbol: msg.symbol,
                            price: latest.price,
                            volume: latest.volume,
                            side: latest.side,
                            timestamp: latest.timestamp,
                            prevPrice: rows.length > 1 ? rows[1].price : latest.price,
                            updatedAt: Date.now(),
                        },
                    }));
                })
                .catch(() => {}); // no history yet — live ticks will fill it in
        } else if (msg.type === 'unsubscribe_ticker' && msg.symbol) {
            setSubscribedSymbols(prev => {
                const next = prev.filter(s => s !== msg.symbol);
                localStorage.setItem(SUBSCRIPTIONS_KEY, JSON.stringify(next));
                return next;
            });
        }
    }, []);

    const removeTicker = useCallback((symbol) => {
        setTickerMap((prev) => {
            const next = { ...prev };
            delete next[symbol];
            return next;
        });
        setPriceHistory(prev => {
            const next = { ...prev };
            delete next[symbol];
            return next;
        });
        setTickHistory(prev => {
            const next = { ...prev };
            delete next[symbol];
            return next;
        });
    }, []);

    return (
        <WebSocketContext.Provider value={{
            ws, tickerMap, priceHistory, tickHistory, sendMessage, setToken, removeTicker,
            connectionStatus, activeNode, throughput, throughputHistory,
            latency, connectedAt, reconnectCount, lastTickAt, symbolTickCounts,
        }}>
            {children}
        </WebSocketContext.Provider>
    );
};
