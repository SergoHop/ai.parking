import { useEffect, useRef, useState } from 'react';
import { useMapStore } from '../../store/mapStore';
import type { ParkingLot } from '../../types/api';
import { pluralSpots, statusLabels } from './status';

interface MockMapProps {
  lots: ParkingLot[];
}

interface Point {
  lat: number;
  lon: number;
}

interface Viewport {
  width: number;
  height: number;
}

interface Insets {
  top: number;
  right: number;
  bottom: number;
  left: number;
}

const MARGIN = 0.08;

/**
 * Projects geographic points into pixel positions inside the viewport.
 * Longitude is scaled by cos(latitude), so the schematic keeps the real
 * proportions of the parking row instead of stretching it to the container.
 */
function projector(points: Point[], viewport: Viewport, insets: Insets) {
  const meanLat = points.reduce((sum, point) => sum + point.lat, 0) / points.length;
  const kx = Math.cos((meanLat * Math.PI) / 180);

  const xs = points.map((point) => point.lon * kx);
  const ys = points.map((point) => point.lat);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minY = Math.min(...ys);
  const maxY = Math.max(...ys);

  const spanX = maxX - minX;
  const spanY = maxY - minY;
  const usableW = Math.max(0, viewport.width - insets.left - insets.right);
  const usableH = Math.max(0, viewport.height - insets.top - insets.bottom);

  const scale = spanX > 0 || spanY > 0
    ? Math.min(
        spanX > 0 ? (usableW * (1 - 2 * MARGIN)) / spanX : Number.POSITIVE_INFINITY,
        spanY > 0 ? (usableH * (1 - 2 * MARGIN)) / spanY : Number.POSITIVE_INFINITY,
      )
    : 0;

  const offsetX = insets.left + (usableW - spanX * scale) / 2;
  const offsetY = insets.top + (usableH - spanY * scale) / 2;

  return (point: Point) => ({
    left: offsetX + (point.lon * kx - minX) * scale,
    top: offsetY + (maxY - point.lat) * scale,
  });
}

function useViewport() {
  const ref = useRef<HTMLDivElement>(null);
  const [viewport, setViewport] = useState<Viewport>({ width: 0, height: 0 });

  useEffect(() => {
    const element = ref.current;
    if (!element) {
      return;
    }
    const observer = new ResizeObserver((entries) => {
      const rect = entries[0]?.contentRect;
      if (rect) {
        setViewport({ width: rect.width, height: rect.height });
      }
    });
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  return { ref, viewport };
}

/**
 * Schematic fallback used until a Yandex Maps API key is configured.
 * It uses the same coordinates and live data, so latency and colours can be
 * verified without spending the daily JS API quota.
 */
export function MockMap({ lots }: MockMapProps) {
  const { ref, viewport } = useViewport();

  const selectedLotId = useMapStore((state) => state.selectedLotId);
  const selectedSpotId = useMapStore((state) => state.selectedSpotId);
  const showOccupied = useMapStore((state) => state.showOccupied);
  const selectLot = useMapStore((state) => state.selectLot);
  const selectSpot = useMapStore((state) => state.selectSpot);
  const clearSelection = useMapStore((state) => state.clearSelection);

  const focusedLot = lots.find((lot) => lot.id === selectedLotId);
  const focusedSpots = (focusedLot?.spots ?? []).filter(
    (spot) => spot.map_lat !== undefined && spot.map_lon !== undefined,
  );

  const points: Point[] = focusedLot
    ? focusedSpots.map((spot) => ({ lat: spot.map_lat as number, lon: spot.map_lon as number }))
    : lots.map((lot) => ({ lat: lot.center_lat, lon: lot.center_lon }));

  const ready = viewport.width > 0 && viewport.height > 0 && points.length > 0;
  const noteReserve = viewport.width < 600 ? 84 : 56;
  const project = ready
    ? projector(points, viewport, {
        top: focusedLot ? 56 : 12,
        right: 12,
        bottom: noteReserve,
        left: 12,
      })
    : null;
  // Keep labels inside the visible area: half of the widest marker plus a gap.
  const clampLeft = (value: number, halfWidth: number) =>
    Math.min(Math.max(value, halfWidth), Math.max(halfWidth, viewport.width - halfWidth));

  return (
    <div className="mock-map" ref={ref}>
      <div className="mock-map__grid" aria-hidden="true" />
      <div className="mock-map__road mock-map__road--h" aria-hidden="true" />
      <div className="mock-map__road mock-map__road--v" aria-hidden="true" />

      {focusedLot ? (
        <>
          <button type="button" className="mock-map__back" onClick={clearSelection}>
            ← Все парковки
          </button>
          <div
            className={[
              'mock-map__badge',
              focusedLot.free_spots > 0 ? '' : 'mock-map__badge--empty',
            ].join(' ')}
          >
            {focusedLot.free_spots} {pluralSpots(focusedLot.free_spots)} свободно
          </div>
          {project &&
            focusedSpots.map((spot) => {
              if (spot.status === 'occupied' && !showOccupied) {
                return null;
              }
              const position = project({ lat: spot.map_lat as number, lon: spot.map_lon as number });
              return (
                <button
                  key={spot.id}
                  type="button"
                  className={[
                    'mock-spot',
                    `status-${spot.status}`,
                    selectedSpotId === spot.id ? 'is-selected' : '',
                  ].join(' ')}
                  style={{
                    left: `${clampLeft(position.left, 30)}px`,
                    top: `${position.top}px`,
                  }}
                  title={`${spot.code}: ${statusLabels[spot.status]}`}
                  onClick={() => selectSpot(focusedLot.id, spot.id)}
                >
                  {spot.code}
                </button>
              );
            })}
          {focusedSpots.length === 0 && (
            <div className="mock-map__empty">
              У мест этой парковки нет координат на карте
            </div>
          )}
        </>
      ) : (
        project &&
        lots.map((lot) => {
          const position = project({ lat: lot.center_lat, lon: lot.center_lon });
          return (
            <button
              key={lot.id}
              type="button"
              className={['mock-lot', lot.fresh ? 'is-fresh' : 'is-stale'].join(' ')}
              style={{
                left: `${clampLeft(position.left, 82)}px`,
                top: `${position.top}px`,
              }}
              onClick={() => selectLot(lot.id)}
            >
              <strong>{lot.name}</strong>
              <span>
                {lot.free_spots} {pluralSpots(lot.free_spots)} · {lot.total_spots} всего
              </span>
            </button>
          );
        })
      )}

      {lots.length === 0 && <div className="mock-map__empty">Нет парковок для отображения</div>}

      <div className="mock-map__note">
        Схема без Яндекс Карт: ключ API не подключён. Данные и WebSocket — настоящие.
      </div>
    </div>
  );
}
