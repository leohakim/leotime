import { CircleAlert, CircleCheck, X } from 'lucide-react';
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';
import type { MessageKey } from './i18n';
import { isLocalId } from './offline/sync';

type ToastVariant = 'success' | 'error';

type ToastItem = {
  id: string;
  message: string;
  variant: ToastVariant;
};

type ToastInput = {
  message: string;
  variant?: ToastVariant;
  durationMs?: number;
};

type ToastContextValue = {
  show: (input: ToastInput) => void;
  success: (message: string) => void;
  error: (message: string) => void;
  dismiss: (id: string) => void;
};

const ToastContext = createContext<ToastContextValue | null>(null);

const DEFAULT_DURATION_MS = 4_000;
const MAX_VISIBLE_TOASTS = 3;

export type ToastTranslator = (key: MessageKey) => string;

export function toastMutationSuccess(
  toast: Pick<ToastContextValue, 'success'>,
  t: ToastTranslator,
  key: MessageKey,
  entityId?: string,
) {
  if (entityId && isLocalId(entityId)) {
    toast.success(t('toastSavedOffline'));
    return;
  }
  toast.success(t(key));
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const dismiss = useCallback((id: string) => {
    setToasts((current) => current.filter((item) => item.id !== id));
  }, []);

  const show = useCallback(
    ({ message, variant = 'success', durationMs = DEFAULT_DURATION_MS }: ToastInput) => {
      const id = crypto.randomUUID();
      setToasts((current) => [...current.slice(-(MAX_VISIBLE_TOASTS - 1)), { id, message, variant }]);
      window.setTimeout(() => dismiss(id), durationMs);
    },
    [dismiss],
  );

  const success = useCallback((message: string) => show({ message, variant: 'success' }), [show]);
  const error = useCallback((message: string) => show({ message, variant: 'error', durationMs: 5_000 }), [show]);

  const value = useMemo(() => ({ show, success, error, dismiss }), [dismiss, error, show, success]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast-stack" aria-live="polite" aria-relevant="additions">
        {toasts.map((toast) => (
          <div key={toast.id} className={`toast toast-${toast.variant}`} role="status">
            {toast.variant === 'success' ? <CircleCheck aria-hidden="true" /> : <CircleAlert aria-hidden="true" />}
            <span className="toast-message">{toast.message}</span>
            <button type="button" className="toast-dismiss" aria-label="Close" onClick={() => dismiss(toast.id)}>
              <X aria-hidden="true" size={16} />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within ToastProvider');
  }
  return context;
}
