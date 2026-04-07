import React, { createContext, useRef, useEffect, useState, useCallback } from 'react';

export const WebSocketContext = createContext(null);

export const WebSocketProvider = ({ children }) => {
    const ws = useRef(null);
    // tickerMap: { AAPL: { symbol, price, prevPrice, volume, side, timestamp, server }, ... }
    const [tickerMap, setTickerMap] = useState({});
    const [token, setToken] = useState(localStorage.getItem('sc2-token'));

    useEffect(() => {
        if (!token) return;

        const env = process.env.REACT_APP_NGINX_ENV || 'local';
        const host = process.env.REACT_APP_NGINX_HOST || 'localhost';
        const port = process.env.REACT_APP_NGINX_PORT || '7700';

        const serverUrl = env === 'local'
            ? `ws://${host}:${port}/ws/stream?token=${token}`
            : `wss://${host}/ws/stream?token=${token}`;

        ws.current = new WebSocket(serverUrl);

        ws.current.onmessage = (event) => {
            try {
                const tick = JSON.parse(event.data);
                if (tick.symbol && tick.price !== undefined) {
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
                }
            } catch (e) {
                console.error('Failed to parse market tick:', e);
            }
        };

        return () => {
            if (ws.current) ws.current.close();
        };
    }, [token]);

    const sendMessage = useCallback((msg) => {
        if (ws.current && ws.current.readyState === WebSocket.OPEN) {
            ws.current.send(JSON.stringify(msg));
        }
    }, []);

    const removeTicker = useCallback((symbol) => {
        setTickerMap((prev) => {
            const next = { ...prev };
            delete next[symbol];
            return next;
        });
    }, []);

    return (
        <WebSocketContext.Provider value={{ ws, tickerMap, sendMessage, setToken, removeTicker }}>
            {children}
        </WebSocketContext.Provider>
    );
};
