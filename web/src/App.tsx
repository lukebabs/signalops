import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { RouterProvider } from '@tanstack/react-router';
import { router } from './router';
import { DashboardStreamBridge } from './components/DashboardStreamBridge';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 10_000, refetchOnWindowFocus: false, retry: false },
  },
});

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <DashboardStreamBridge />
      <RouterProvider router={router} />
    </QueryClientProvider>
  );
}
