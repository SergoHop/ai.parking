import { useEffect, useState } from 'react';

/** Re-renders occasionally so relative timestamps and TTL markers stay honest. */
export function useNow(intervalMs = 5000): number {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), intervalMs);
    return () => window.clearInterval(timer);
  }, [intervalMs]);

  return now;
}
