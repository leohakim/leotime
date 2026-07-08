import { CloudOff, CloudUpload, RefreshCw } from 'lucide-react';
import type { MessageKey } from '../i18n';
import { useOfflineStatus } from './offlineContext';

export type Translator = (key: MessageKey) => string;

export function OfflineStatusPill({ t }: { t: Translator }) {
  const { lastSyncError, online, pendingCount, syncNow, syncing } = useOfflineStatus();

  if (online && pendingCount === 0 && !syncing && !lastSyncError) {
    return null;
  }

  const label = syncing
    ? t('offlineSyncing')
    : !online
      ? pendingCount > 0
        ? t('offlinePendingCount').replace('{count}', String(pendingCount))
        : t('offlineCreatesOnly')
      : pendingCount > 0
        ? t('offlinePendingCount').replace('{count}', String(pendingCount))
        : lastSyncError
          ? t('offlineSyncFailed')
          : t('offlineReady');

  return (
    <div className={`offline-status-pill${online ? '' : ' is-offline'}${syncing ? ' is-syncing' : ''}`}>
      {!online ? <CloudOff aria-hidden="true" /> : syncing ? <RefreshCw aria-hidden="true" className="spin-icon" /> : <CloudUpload aria-hidden="true" />}
      <span>{label}</span>
      {online && pendingCount > 0 && !syncing ? (
        <button className="offline-sync-button" onClick={() => void syncNow()} type="button">
          {t('offlineSyncNow')}
        </button>
      ) : null}
    </div>
  );
}
