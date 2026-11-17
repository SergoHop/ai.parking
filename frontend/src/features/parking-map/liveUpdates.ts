import type { QueryClient } from '@tanstack/react-query';
import type { LiveMessage, ParkingLot, SpotStatus } from '../../types/api';

function recount(lot: ParkingLot): ParkingLot {
  const spots = lot.spots ?? [];
  return {
    ...lot,
    free_spots: spots.filter((spot) => spot.status === 'free').length,
    occupied_spots: spots.filter((spot) => spot.status === 'occupied').length,
    unknown_spots: spots.filter((spot) => spot.status === 'unknown').length,
  };
}

/** Applies a live occupancy event to the cached parking lot list. */
export function patchSpotStatus(queryClient: QueryClient, message: LiveMessage): void {
  if (message.spot_id === undefined) {
    return;
  }

  queryClient.setQueryData<ParkingLot[]>(['lots'], (lots) => {
    if (!lots) {
      return lots;
    }
    return lots.map((lot) => {
      if (lot.id !== message.lot_id) {
        return lot;
      }
      const spots = (lot.spots ?? []).map((spot) =>
        spot.id === message.spot_id
          ? {
              ...spot,
              status: message.status as SpotStatus,
              confidence: message.confidence ?? spot.confidence,
              status_updated_at: message.at,
              fresh: message.status !== 'unknown',
            }
          : spot,
      );
      return recount({ ...lot, spots, last_frame_at: message.at, fresh: message.status !== 'unknown' });
    });
  });
}
