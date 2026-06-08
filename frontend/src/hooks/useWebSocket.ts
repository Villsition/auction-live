import { useEffect, useRef, useCallback, useState } from 'react';
import type { WSMessage } from '../types';

export function useWebSocket(roomId: number, token: string | null) {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectRef = useRef(0);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const mountedRef = useRef(true);
  const [connected, setConnected] = useState(false);
  const listenersRef = useRef<Map<string, Set<(data: WSMessage) => void>>>(new Map());

  const connect = useCallback(() => {
    if (!token || !mountedRef.current) return;
    const protocol = location.protocol === 'https:' ? 'wss' : 'ws';
    const ws = new WebSocket(`${protocol}://${location.host}/api/ws?token=${token}&room_id=${roomId}`);

    ws.onopen = () => {
      if (!mountedRef.current) { ws.close(); return; }
      setConnected(true);
      reconnectRef.current = 0;
    };

    ws.onmessage = (e) => {
      try {
        const msg: WSMessage = JSON.parse(e.data);
        listenersRef.current.get(msg.type)?.forEach(fn => fn(msg));
        listenersRef.current.get('*')?.forEach(fn => fn(msg));
      } catch { /* ignore */ }
    };

    ws.onclose = () => {
      if (!mountedRef.current) return;
      setConnected(false);
      const delay = Math.min(1000 * Math.pow(2, reconnectRef.current), 30000);
      reconnectRef.current++;
      reconnectTimerRef.current = setTimeout(connect, delay);
    };

    ws.onerror = () => ws.close();
    wsRef.current = ws;
  }, [roomId, token]);

  useEffect(() => {
    mountedRef.current = true;
    reconnectRef.current = 0;
    connect();
    return () => {
      mountedRef.current = false;
      if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current);
      wsRef.current?.close();
    };
  }, [connect]);

  const subscribe = useCallback((type: string, fn: (data: WSMessage) => void) => {
    if (!listenersRef.current.has(type)) {
      listenersRef.current.set(type, new Set());
    }
    listenersRef.current.get(type)!.add(fn);
    return () => {
      listenersRef.current.get(type)?.delete(fn);
    };
  }, []);

  return { connected, subscribe };
}
