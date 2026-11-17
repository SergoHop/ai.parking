import type { SpotStatus } from '../../types/api';

export const statusLabels: Record<SpotStatus, string> = {
  free: 'свободно',
  occupied: 'занято',
  unknown: 'нет данных',
};

export function formatAge(iso: string | undefined, now: number): string {
  if (!iso) {
    return 'нет данных';
  }
  const seconds = Math.max(0, Math.round((now - new Date(iso).getTime()) / 1000));
  if (seconds < 5) {
    return 'только что';
  }
  if (seconds < 60) {
    return `${seconds} с назад`;
  }
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) {
    return `${minutes} мин назад`;
  }
  const hours = Math.round(minutes / 60);
  return `${hours} ч назад`;
}

export function pluralSpots(count: number): string {
  const mod10 = count % 10;
  const mod100 = count % 100;
  if (mod10 === 1 && mod100 !== 11) {
    return 'место';
  }
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) {
    return 'места';
  }
  return 'мест';
}
