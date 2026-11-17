import type { LiveMessage } from '../types/api';

export type LiveStatus = 'connecting' | 'online' | 'offline';

interface LiveHandlers {
  onMessage: (message: LiveMessage) => void;
  onStatus: (status: LiveStatus) => void;
}

function liveURL(): string {
  const explicit = import.meta.env.VITE_WS_URL?.trim();
  if (explicit) {
    return explicit;
  }
  const api = (import.meta.env.VITE_API_URL ?? 'http://localhost:8080').replace(/\/+$/, '');
  return `${api.replace(/^http/, 'ws')}/ws/live`;
}

/**
 * Opens the live status stream and reconnects with exponential backoff.
 * Returns a cleanup function.
 */
export function connectLive(handlers: LiveHandlers): () => void {
  let socket: WebSocket | null = null;
  let timer: number | undefined;
  let attempt = 0;
  let closed = false;

  const scheduleReconnect = () => {
    if (closed) {
      return;
    }
    const delay = Math.min(1000 * 2 ** attempt, 10_000);
    attempt += 1;
    timer = window.setTimeout(open, delay);
  };

  const open = () => {
    if (closed) {
      return;
    }
    handlers.onStatus('connecting');
    socket = new WebSocket(liveURL());

    socket.onopen = () => {
      attempt = 0;
      handlers.onStatus('online');
    };

    socket.onmessage = (event) => {
      try {
        handlers.onMessage(JSON.parse(event.data as string) as LiveMessage);
      } catch {
        // Ignore malformed frames; the next REST refresh will reconcile state.
      }
    };

    socket.onerror = () => {
      socket?.close();
    };

    socket.onclose = () => {
      handlers.onStatus('offline');
      scheduleReconnect();
    };
  };

  open();

  return () => {
    closed = true;
    if (timer !== undefined) {
      window.clearTimeout(timer);
    }
    socket?.close();
  };
}
