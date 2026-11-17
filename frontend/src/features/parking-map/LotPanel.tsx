import { debugEnabled, useMapStore } from '../../store/mapStore';
import type { ParkingLot } from '../../types/api';
import { formatAge, pluralSpots, statusLabels } from './status';
import { useNow } from './useNow';

interface LotPanelProps {
  lots: ParkingLot[];
}

export function LotPanel({ lots }: LotPanelProps) {
  const now = useNow();
  const selectedLotId = useMapStore((state) => state.selectedLotId);
  const selectedSpotId = useMapStore((state) => state.selectedSpotId);
  const showOccupied = useMapStore((state) => state.showOccupied);
  const selectLot = useMapStore((state) => state.selectLot);
  const selectSpot = useMapStore((state) => state.selectSpot);
  const clearSelection = useMapStore((state) => state.clearSelection);
  const toggleOccupied = useMapStore((state) => state.toggleOccupied);

  const selectedLot = lots.find((lot) => lot.id === selectedLotId);
  const selectedSpot = selectedLot?.spots?.find((spot) => spot.id === selectedSpotId);

  if (lots.length === 0) {
    return (
      <aside className="panel">
        <div className="empty">
          <h2>Парковок пока нет</h2>
          <p>Добавьте парковку и камеру через API или seed-скрипт.</p>
        </div>
      </aside>
    );
  }

  return (
    <aside className="panel">
      {selectedLot ? (
        <section className="panel__section">
          <button type="button" className="link-button" onClick={clearSelection}>
            ← Все парковки
          </button>
          <h2 className="panel__title">{selectedLot.name}</h2>
          <p className="panel__subtitle">{selectedLot.address || 'Адрес не указан'}</p>

          <div className="stats">
            <div className="stat stat--free">
              <strong>{selectedLot.free_spots}</strong>
              <span>свободно</span>
            </div>
            <div className="stat stat--occupied">
              <strong>{selectedLot.occupied_spots}</strong>
              <span>занято</span>
            </div>
            <div className="stat stat--unknown">
              <strong>{selectedLot.unknown_spots}</strong>
              <span>нет данных</span>
            </div>
          </div>

          <p className={['freshness', selectedLot.fresh ? 'is-fresh' : 'is-stale'].join(' ')}>
            {selectedLot.fresh ? 'Камера передаёт данные' : 'Данные устарели или камера offline'}
            {' · '}
            {formatAge(selectedLot.last_frame_at, now)}
          </p>

          {selectedSpot ? (
            <div className={`selected-spot status-${selectedSpot.status}`}>
              <strong>Место {selectedSpot.code}</strong>
              <span>{statusLabels[selectedSpot.status]}</span>
              {selectedSpot.confidence !== undefined && (
                <small>уверенность {Math.round(selectedSpot.confidence * 100)}%</small>
              )}
            </div>
          ) : (
            <p className="hint">Выберите место, чтобы подсветить его на карте.</p>
          )}

          <div className="spot-grid">
            {(selectedLot.spots ?? []).map((spot) => {
              if (spot.status === 'occupied' && !showOccupied) {
                return null;
              }
              return (
                <button
                  key={spot.id}
                  type="button"
                  className={[
                    'spot-chip',
                    `status-${spot.status}`,
                    selectedSpotId === spot.id ? 'is-selected' : '',
                  ].join(' ')}
                  onClick={() => selectSpot(selectedLot.id, spot.id)}
                >
                  <span className="spot-chip__code">{spot.code}</span>
                  <span className="spot-chip__status">{statusLabels[spot.status]}</span>
                </button>
              );
            })}
          </div>
        </section>
      ) : (
        <section className="panel__section">
          <h2 className="panel__title">Парковки рядом</h2>
          <p className="panel__subtitle">
            Свободных мест: {lots.reduce((sum, lot) => sum + lot.free_spots, 0)} из{' '}
            {lots.reduce((sum, lot) => sum + lot.total_spots, 0)}
          </p>
          <div className="lot-list">
            {lots.map((lot) => (
              <button key={lot.id} type="button" className="lot-card" onClick={() => selectLot(lot.id)}>
                <span className="lot-card__head">
                  <strong>{lot.name}</strong>
                  <span className={['dot', lot.fresh ? 'dot--online' : 'dot--offline'].join(' ')} />
                </span>
                <span className="lot-card__address">{lot.address || 'Адрес не указан'}</span>
                <span className="lot-card__counts">
                  <span className="count count--free">
                    {lot.free_spots} {pluralSpots(lot.free_spots)}
                  </span>
                  <span className="count count--unknown">{lot.unknown_spots} без данных</span>
                </span>
                <span className="lot-card__meta">{formatAge(lot.last_frame_at, now)}</span>
              </button>
            ))}
          </div>
        </section>
      )}

      <section className="panel__section panel__legend">
        <div className="legend">
          <span className="legend__item">
            <i className="dot dot--free" /> свободно
          </span>
          <span className="legend__item">
            <i className="dot dot--unknown" /> нет данных
          </span>
          {debugEnabled && (
            <span className="legend__item">
              <i className="dot dot--occupied" /> занято
            </span>
          )}
        </div>
        {debugEnabled && (
          <button type="button" className="link-button" onClick={toggleOccupied}>
            {showOccupied ? 'Скрыть занятые места' : 'Показать занятые места'}
          </button>
        )}
      </section>
    </aside>
  );
}
