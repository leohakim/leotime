import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { describe, expect, test, vi } from 'vitest';
import { AuthScreen } from './authUi';
import { translate } from './i18n';
import { ToastProvider } from './toast';

const t = (key: Parameters<typeof translate>[1]) => translate('es', key);

function renderAuthScreen() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <AuthScreen locale="es" onAuthenticated={vi.fn()} setLocale={vi.fn()} t={t} />
      </ToastProvider>
    </QueryClientProvider>,
  );
}

describe('AuthScreen', () => {
  test('renders product context and the sign-in form together', () => {
    renderAuthScreen();

    expect(screen.getByRole('heading', { level: 1, name: /Tiempo, timesheet y facturas/ })).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 2, name: 'Tu mesa de trabajo diaria' })).toBeInTheDocument();
    expect(screen.getByText('Timer y entrada manual para captura rapida')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Entrar' })).toBeInTheDocument();
  });
});
