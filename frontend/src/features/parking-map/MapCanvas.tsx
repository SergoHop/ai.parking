import type { ParkingLot } from '../../types/api';
import { MockMap } from './MockMap';
import { YandexMap } from './YandexMap';

interface MapCanvasProps {
  lots: ParkingLot[];
}

const apiKey = (import.meta.env.VITE_YANDEX_MAPS_API_KEY ?? '').trim();
const hasYandexKey = apiKey.length > 20 && !apiKey.toLowerCase().includes('your');

/** Picks the real Yandex map when a key is configured, otherwise a schematic. */
export function MapCanvas({ lots }: MapCanvasProps) {
  if (!hasYandexKey) {
    return <MockMap lots={lots} />;
  }
  return <YandexMap lots={lots} apiKey={apiKey} />;
}
