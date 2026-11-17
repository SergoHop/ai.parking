import { useEffect, useRef, useState } from 'react';
import { useMapStore } from '../../store/mapStore';
import type { ParkingLot } from '../../types/api';
import type { YandexMapsApi } from '../../types/ymaps';
import { pluralSpots, statusLabels } from './status';

interface YandexMapProps {
  lots: ParkingLot[];
  apiKey: string;
}

const SCRIPT_ID = 'yandex-maps-v3';
const SCRIPT_TIMEOUT_MS = 15_000;

function loadApi(apiKey: string): Promise<YandexMapsApi> {
  if (window.ymaps3) {
    return Promise.resolve(window.ymaps3);
  }

  return new Promise((resolve, reject) => {
    const startedAt = Date.now();

    const waitForApi = () => {
      if (window.ymaps3) {
        resolve(window.ymaps3);
        return;
      }
      if (Date.now() - startedAt > SCRIPT_TIMEOUT_MS) {
        reject(new Error('Яндекс Карты не загрузились за 15 секунд'));
        return;
      }
      window.setTimeout(waitForApi, 100);
    };

    const existing = document.getElementById(SCRIPT_ID) as HTMLScriptElement | null;
    if (existing) {
      waitForApi();
      return;
    }

    const script = document.createElement('script');
    script.id = SCRIPT_ID;
    script.src = `https://api-maps.yandex.ru/v3/?apikey=${encodeURIComponent(apiKey)}&lang=ru_RU`;
    script.async = true;
    script.addEventListener('load', waitForApi, { once: true });
    script.addEventListener(
      'error',
      () => reject(new Error('Не удалось загрузить Яндекс Карты. Проверьте ключ и referer.')),
      { once: true },
    );
    document.head.append(script);
  });
}

type LngLat = [number, number];

function allPoints(lots: ParkingLot[]): LngLat[] {
  const points: LngLat[] = [];
  for (const lot of lots) {
    for (const spot of lot.spots ?? []) {
      if (spot.map_lon !== undefined && spot.map_lat !== undefined) {
        points.push([spot.map_lon, spot.map_lat]);
      }
    }
    points.push([lot.center_lon, lot.center_lat]);
  }
  return points;
}

/** Average of every known point: used for the «Открыть в Картах» link. */
function startCenter(lots: ParkingLot[]): LngLat {
  const points = allPoints(lots);
  if (points.length === 0) {
    return [37.6, 55.75];
  }
  const lon = points.reduce((sum, point) => sum + point[0], 0) / points.length;
  const lat = points.reduce((sum, point) => sum + point[1], 0) / points.length;
  return [lon, lat];
}

/** Initial viewport: fit every known lot, so both demo and CV lots are visible. */
function startLocation(lots: ParkingLot[]): Record<string, unknown> {
  const points = allPoints(lots);
  if (points.length === 0) {
    return { center: [37.6, 55.75], zoom: 15 };
  }
  if (points.length === 1) {
    return { center: points[0], zoom: 17 };
  }

  const lons = points.map((point) => point[0]);
  const lats = points.map((point) => point[1]);
  const minLon = Math.min(...lons);
  const maxLon = Math.max(...lons);
  const minLat = Math.min(...lats);
  const maxLat = Math.max(...lats);

  const padLon = Math.max((maxLon - minLon) * 0.25, 0.0008);
  const padLat = Math.max((maxLat - minLat) * 0.25, 0.0006);

  return {
    bounds: [
      [minLon - padLon, minLat - padLat],
      [maxLon + padLon, maxLat + padLat],
    ],
  };
}

export function YandexMap({ lots, apiKey }: YandexMapProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<unknown>(null);
  const markersRef = useRef<unknown[]>([]);
  const centerRef = useRef(startCenter(lots));
  const locationRef = useRef(startLocation(lots));
  const [ready, setReady] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const selectedLotId = useMapStore((state) => state.selectedLotId);
  const selectedSpotId = useMapStore((state) => state.selectedSpotId);
  const showOccupied = useMapStore((state) => state.showOccupied);
  const selectLot = useMapStore((state) => state.selectLot);
  const selectSpot = useMapStore((state) => state.selectSpot);

  useEffect(() => {
    let cancelled = false;

    (async () => {
      try {
        const ymaps3 = await loadApi(apiKey);
        await ymaps3.ready;
        if (cancelled || !containerRef.current) {
          return;
        }

        const map = new ymaps3.YMap(containerRef.current, {
          location: locationRef.current,
        });
        map.addChild(new ymaps3.YMapDefaultSchemeLayer({}));
        map.addChild(new ymaps3.YMapDefaultFeaturesLayer({}));

        mapRef.current = map;
        setReady(true);
      } catch (loadError) {
        if (!cancelled) {
          setError(loadError instanceof Error ? loadError.message : 'Карта недоступна');
        }
      }
    })();

    return () => {
      cancelled = true;
      for (const marker of markersRef.current) {
        try {
          (mapRef.current as { removeChild: (child: unknown) => void } | null)?.removeChild(marker);
        } catch {
          // The map may already be destroyed.
        }
      }
      markersRef.current = [];
      (mapRef.current as { destroy: () => void } | null)?.destroy();
      mapRef.current = null;
    };
  }, [apiKey]);

  useEffect(() => {
    const map = mapRef.current as {
      addChild: (child: unknown) => void;
      removeChild: (child: unknown) => void;
    } | null;
    const ymaps3 = window.ymaps3;
    if (!ready || !map || !ymaps3) {
      return;
    }

    for (const marker of markersRef.current) {
      try {
        map.removeChild(marker);
      } catch {
        // Ignore stale markers.
      }
    }
    markersRef.current = [];

    const addMarker = (element: HTMLElement, coordinates: [number, number]) => {
      const marker = new ymaps3.YMapMarker({ coordinates }, element);
      map.addChild(marker);
      markersRef.current.push(marker);
    };

    for (const lot of lots) {
      const element = document.createElement('button');
      element.type = 'button';
      element.className = ['ymap-lot', selectedLotId === lot.id ? 'is-selected' : ''].join(' ');
      const title = document.createElement('strong');
      title.textContent = lot.name;
      const summary = document.createElement('span');
      summary.textContent = `${lot.free_spots} ${pluralSpots(lot.free_spots)} свободно`;
      element.append(title, summary);
      element.addEventListener('click', () => selectLot(lot.id));
      addMarker(element, [lot.center_lon, lot.center_lat]);
    }

    for (const lot of lots) {
      for (const spot of lot.spots ?? []) {
        if (spot.map_lon === undefined || spot.map_lat === undefined) {
          continue;
        }
        if (spot.status === 'occupied' && !showOccupied) {
          continue;
        }
        const element = document.createElement('button');
        element.type = 'button';
        element.className = [
          'ymap-spot',
          `status-${spot.status}`,
          selectedSpotId === spot.id ? 'is-selected' : '',
        ].join(' ');
        element.textContent = spot.code;
        element.title = `${spot.code}: ${statusLabels[spot.status]}`;
        element.addEventListener('click', () => selectSpot(lot.id, spot.id));
        addMarker(element, [spot.map_lon, spot.map_lat]);
      }
    }
  }, [ready, lots, selectedLotId, selectedSpotId, showOccupied, selectLot, selectSpot]);

  useEffect(() => {
    if (!ready) {
      return;
    }
    const lot = lots.find((item) => item.id === selectedLotId);
    if (!lot) {
      return;
    }
    const map = mapRef.current as {
      setLocation?: (props: Record<string, unknown>) => void;
    } | null;
    map?.setLocation?.({ center: [lot.center_lon, lot.center_lat], zoom: 18, duration: 300 });
  }, [ready, selectedLotId, lots]);

  return (
    <div className="map-container">
      <div ref={containerRef} className="ymap" />
      {!ready && !error && <div className="map-overlay">Загружаем Яндекс Карты…</div>}
      {error && <div className="map-overlay map-overlay--error">{error}</div>}
      {ready && (
        <a
          className="map-attribution"
          href={`https://yandex.ru/maps/?ll=${centerRef.current[0]}%2C${centerRef.current[1]}&z=17`}
          target="_blank"
          rel="noreferrer"
        >
          Открыть в Картах
        </a>
      )}
    </div>
  );
}
