import { create } from 'zustand';

export const debugEnabled =
  typeof window !== 'undefined' &&
  new URLSearchParams(window.location.search).get('debug') === '1';

interface MapState {
  selectedLotId: number | null;
  selectedSpotId: number | null;
  showOccupied: boolean;
  selectLot: (lotId: number) => void;
  selectSpot: (lotId: number, spotId: number) => void;
  clearSelection: () => void;
  toggleOccupied: () => void;
}

export const useMapStore = create<MapState>((set) => ({
  selectedLotId: null,
  selectedSpotId: null,
  showOccupied: debugEnabled,
  selectLot: (lotId) => set({ selectedLotId: lotId, selectedSpotId: null }),
  selectSpot: (lotId, spotId) => set({ selectedLotId: lotId, selectedSpotId: spotId }),
  clearSelection: () => set({ selectedLotId: null, selectedSpotId: null }),
  toggleOccupied: () => set((state) => ({ showOccupied: !state.showOccupied })),
}));
