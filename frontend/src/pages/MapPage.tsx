import { useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { connectLive, type LiveStatus } from '../api/live';
import { LotPanel } from '../features/parking-map/LotPanel';
import { MapCanvas } from '../features/parking-map/MapCanvas';
import { patchSpotStatus } from '../features/parking-map/liveUpdates';
import { pluralSpots } from '../features/parking-map/status';
import { useParkingLots } from '../features/parking-map/useParkingLots';

const liveLabels: Record<LiveStatus, string> = {
  online: 'live',
  connecting: 'подключение',
  offline: 'нет связи',
};

export function MapPage() {
  const queryClient = useQueryClient();
  const { data: lots = [], isLoading, error, refetch, isFetching } = useParkingLots();
  const [live, setLive] = useState<LiveStatus>('connecting');

  useEffect(() => {
    return connectLive({
      onMessage: (message) => {
        if (message.type === 'spot_status') {
          patchSpotStatus(queryClient, message);
          return;
        }
        // A camera went online/offline: effective statuses may have changed.
        void queryClient.invalidateQueries({ queryKey: ['lots'] });
      },
      onStatus: setLive,
    });
  }, [queryClient]);

  const totalFree = lots.reduce((sum, lot) => sum + lot.free_spots, 0);

  return (
    <div className="app">
      <header className="topbar">
        <div className="topbar__titles">
          <h1>Парковка рядом</h1>
          <p>
            {totalFree > 0
              ? `${totalFree} ${pluralSpots(totalFree)} свободно по данным камер`
              : 'Свободных мест по данным камер пока нет'}
          </p>
        </div>
        <div className="topbar__actions">
          <span className={`live live--${live}`} title="Состояние WebSocket">
            <i />
            {liveLabels[live]}
          </span>
          <button
            type="button"
            className="refresh"
            onClick={() => void refetch()}
            disabled={isFetching}
          >
            {isFetching ? 'Обновляю…' : 'Обновить'}
          </button>
        </div>
      </header>

      {error && (
        <div className="error-banner">
          <span>Не удалось получить данные: {error.message}</span>
          <button type="button" className="link-button" onClick={() => void refetch()}>
            Повторить
          </button>
        </div>
      )}

      <main className="layout">
        <section className="map-pane">
          {isLoading ? (
            <div className="map-empty">Загружаем парковки…</div>
          ) : (
            <MapCanvas lots={lots} />
          )}
        </section>
        <LotPanel lots={lots} />
      </main>
    </div>
  );
}
