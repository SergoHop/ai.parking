import { useQuery } from '@tanstack/react-query';
import { api } from '../../api/client';

export function useParkingLots() {
  return useQuery({
    queryKey: ['lots'],
    queryFn: api.lots,
    refetchInterval: 30_000,
    staleTime: 5_000,
  });
}
