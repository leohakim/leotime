import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { ApiError } from './api';
import { QueryErrorBanner } from './crudFormUi';

const t = (key: string) =>
  ({
    directoryLoadFailed: 'Could not load directory.',
    maintenanceModeMessage: 'Server is in maintenance mode.',
    reloadApp: 'Reload application',
    retry: 'Retry',
  })[key] ?? key;

describe('QueryErrorBanner', () => {
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  test('shows maintenance copy and reload for maintenance_mode errors', () => {
    const reload = vi.fn();
    vi.stubGlobal('location', { ...window.location, reload });
    const error = new ApiError(503, {
      code: 'maintenance_mode',
      message: 'server is in maintenance mode; reload the application',
    });

    render(<QueryErrorBanner error={error} onRetry={() => undefined} t={t} />);

    expect(screen.getByRole('alert')).toHaveTextContent('Server is in maintenance mode.');
    fireEvent.click(screen.getByRole('button', { name: 'Reload application' }));
    expect(reload).toHaveBeenCalled();
  });

  test('shows retry for regular API errors', () => {
    const onRetry = vi.fn();
    const error = new ApiError(500, { code: 'internal_error', message: 'Something broke' });

    render(<QueryErrorBanner error={error} onRetry={onRetry} t={t} />);

    expect(screen.getByRole('alert')).toHaveTextContent('Something broke');
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(onRetry).toHaveBeenCalled();
  });
});
