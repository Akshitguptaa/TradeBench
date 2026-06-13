import { useEffect, useState, useRef, useCallback } from 'react';

export interface LeaderboardEntry {
  rank: number;
  contestant_id: string;
  score: number;
  p50_ms?: number;
  p99_ms?: number;
}

export function useLeaderboard() {
  const [entries, setEntries] = useState<LeaderboardEntry[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const connect = useCallback(() => {
    try {
      // Auto-detect protocol: use wss:// when page is served over HTTPS
      let wsBase = process.env.NEXT_PUBLIC_WS_BASE || 'ws://localhost:8080';
      if (typeof window !== 'undefined' && window.location.protocol === 'https:') {
        wsBase = wsBase.replace(/^ws:\/\//, 'wss://');
      }
      const wsUrl = `${wsBase}/ws/leaderboard`;
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        setIsConnected(true);
        console.log('Leaderboard WS Connected');
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data.type === 'snapshot') {
            setEntries(data.payload as LeaderboardEntry[]);
          } else if (data.type === 'update') {
            const update = data.payload as LeaderboardEntry;
            setEntries((prev) => {
              const index = prev.findIndex((e) => e.contestant_id === update.contestant_id);
              let next = [...prev];
              if (index >= 0) {
                next[index] = update;
              } else {
                next.push(update);
              }
              // Sort descending by score
              next.sort((a, b) => b.score - a.score);
              // Re-assign ranks
              next = next.map((e, i) => ({ ...e, rank: i + 1 }));
              return next;
            });
          }
        } catch (err) {
          console.error('Failed to parse WS message:', err);
        }
      };

      ws.onclose = () => {
        setIsConnected(false);
        console.log('Leaderboard WS Disconnected. Reconnecting in 3s...');
        reconnectTimeoutRef.current = setTimeout(connect, 3000);
      };

      ws.onerror = (err) => {
        console.error('Leaderboard WS Error:', err);
        ws.close();
      };
    } catch (err) {
      // SecurityError: browser blocks ws:// from https:// pages (mixed content)
      console.warn('WebSocket connection failed (likely mixed content):', err);
      setIsConnected(false);
    }
  }, []);

  const reconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.onclose = null;
      wsRef.current.close();
      setIsConnected(false);
    }
    if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current);
    connect();
  }, [connect]);

  useEffect(() => {
    connect();
    return () => {
      if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current);
      if (wsRef.current) {
        wsRef.current.onclose = null; // prevent reconnect on unmount
        wsRef.current.close();
      }
    };
  }, [connect]);

  return { entries, isConnected, reconnect };
}
