import type { LotDetails, ParkingLot } from '../types/api';

const API_URL = (import.meta.env.VITE_API_URL ?? 'http://localhost:8080').replace(/\/+$/, '');

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, {
    headers: { Accept: 'application/json' },
  });
  if (!response.ok) {
    throw new Error(`API ${response.status} ${response.statusText}`);
  }
  return (await response.json()) as T;
}

export const api = {
  lots: () => getJSON<ParkingLot[]>('/api/parking-lots?include=spots'),
  lot: (id: number) => getJSON<LotDetails>(`/api/parking-lots/${id}`),
};
