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
    // Get token from cookie or local storage if required by the gateway WS
    // Since gateway tests showed WS needs auth if configured, but let's assume it works or we need to pass a token
    // The problem statement didn't specify JWT required for leaderboard WS if we don't have it, but gateway might enforce it.
    // For now, let's connect directly.

    // Use relative URL if using next rewrites, but WS rewrites in Next.js app router are tricky.
    const wsBase = process.env.NEXT_PUBLIC_WS_BASE || 'ws://localhost:8080';
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
