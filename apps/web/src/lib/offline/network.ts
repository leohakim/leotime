import { isApiError } from '../api';

type NetworkListener = (online: boolean) => void;

const listeners = new Set<NetworkListener>();

export function isOnline(): boolean {
  return typeof navigator === 'undefined' ? true : navigator.onLine;
}

export function subscribeNetworkStatus(listener: NetworkListener): () => void {
  if (typeof window === 'undefined') {
    return () => undefined;
  }

  listeners.add(listener);
  const handleOnline = () => listeners.forEach((item) => item(true));
  const handleOffline = () => listeners.forEach((item) => item(false));
  window.addEventListener('online', handleOnline);
  window.addEventListener('offline', handleOffline);

  return () => {
    listeners.delete(listener);
    window.removeEventListener('online', handleOnline);
    window.removeEventListener('offline', handleOffline);
  };
}

export function isNetworkFailure(error: unknown): boolean {
  if (error instanceof TypeError) {
    return true;
  }
  return isApiError(error) && (error.status === 502 || error.status === 503);
}
