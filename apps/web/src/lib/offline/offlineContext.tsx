import { useQueryClient } from '@tanstack/react-query';
import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import { flushOfflineQueue, pendingMutationCount } from './mutations';
import { isOnline, subscribeNetworkStatus } from './network';

type OfflineContextValue = {
  online: boolean;
  pendingCount: number;
  syncing: boolean;
  syncNow: () => Promise<void>;
  refreshPendingCount: () => Promise<void>;
  lastSyncError: string;
};

const OfflineContext = createContext<OfflineContextValue | null>(null);

export function OfflineProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [online, setOnline] = useState(isOnline());
  const [pendingCount, setPendingCount] = useState(0);
  const [syncing, setSyncing] = useState(false);
  const [lastSyncError, setLastSyncError] = useState('');

  const refreshPendingCount = useCallback(async () => {
    setPendingCount(await pendingMutationCount());
  }, []);

  const syncNow = useCallback(async () => {
    if (!isOnline() || syncing) {
      return;
    }
    setSyncing(true);
    setLastSyncError('');
    try {
      const result = await flushOfflineQueue();
      await refreshPendingCount();
      if (result.failed > 0 && result.lastError) {
        setLastSyncError(result.lastError);
      }
      if (result.synced > 0) {
        await queryClient.invalidateQueries();
      }
    } finally {
      setSyncing(false);
    }
  }, [queryClient, refreshPendingCount, syncing]);

  useEffect(() => {
    void refreshPendingCount();
  }, [refreshPendingCount]);

  useEffect(() => {
    return subscribeNetworkStatus((nextOnline) => {
      setOnline(nextOnline);
      if (nextOnline) {
        void syncNow();
      }
    });
  }, [syncNow]);

  const value = useMemo(
    () => ({
      online,
      pendingCount,
      syncing,
      syncNow,
      refreshPendingCount,
      lastSyncError,
    }),
    [lastSyncError, online, pendingCount, refreshPendingCount, syncNow, syncing],
  );

  return <OfflineContext.Provider value={value}>{children}</OfflineContext.Provider>;
}

export function useOfflineStatus() {
  const context = useContext(OfflineContext);
  if (!context) {
    throw new Error('useOfflineStatus must be used within OfflineProvider');
  }
  return context;
}

export function useOfflineQueueRefresh() {
  const { syncNow } = useOfflineStatus();
  const queryClient = useQueryClient();

  return useCallback(async () => {
    await syncNow();
    await queryClient.invalidateQueries();
  }, [queryClient, syncNow]);
}
