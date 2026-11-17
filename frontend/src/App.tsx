import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MapPage } from './pages/MapPage';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: true,
    },
  },
});

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <MapPage />
    </QueryClientProvider>
  );
}
