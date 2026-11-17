export type SpotStatus = 'free' | 'occupied' | 'unknown';
export type CameraStatus = 'online' | 'offline' | 'error';

export interface ParkingSpot {
  id: number;
  lot_id: number;
  camera_id?: number;
  code: string;
  polygon: number[][];
  map_lat?: number;
  map_lon?: number;
  status: SpotStatus;
  confidence?: number;
  status_updated_at?: string;
  fresh: boolean;
}

export interface Camera {
  id: number;
  lot_id: number;
  name: string;
  source_type: string;
  source: string;
  status: CameraStatus;
  last_frame_at?: string;
  fps?: number;
  resolution?: string;
  fresh: boolean;
}

export interface ParkingLot {
  id: number;
  name: string;
  address: string;
  kind: 'lot' | 'roadside';
  center_lat: number;
  center_lon: number;
  entrance_lat?: number;
  entrance_lon?: number;
  radius_m: number;
  geojson?: unknown;
  created_at: string;
  total_spots: number;
  free_spots: number;
  occupied_spots: number;
  unknown_spots: number;
  last_frame_at?: string;
  fresh: boolean;
  spots?: ParkingSpot[];
}

export interface LotDetails {
  lot: ParkingLot;
  spots: ParkingSpot[];
  cameras: Camera[];
}

export interface LiveMessage {
  type: 'spot_status' | 'camera_status';
  lot_id: number;
  spot_id?: number;
  code?: string;
  status: SpotStatus | CameraStatus;
  confidence?: number;
  camera_id?: number;
  last_frame_at?: string;
  at: string;
}
