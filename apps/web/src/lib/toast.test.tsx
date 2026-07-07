import { act, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { ToastProvider, useToast } from './toast';

function ToastProbe() {
  const toast = useToast();
  return (
    <button type="button" onClick={() => toast.success('Saved')}>
      Show toast
    </button>
  );
}

describe('toast', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  test('shows and auto-dismisses success toast', () => {
    vi.useFakeTimers();
    render(
      <ToastProvider>
        <ToastProbe />
      </ToastProvider>,
    );

    act(() => {
      screen.getByRole('button', { name: 'Show toast' }).click();
    });

    expect(screen.getByRole('status')).toHaveTextContent('Saved');

    act(() => {
      vi.advanceTimersByTime(4_000);
    });

    expect(screen.queryByRole('status')).toBeNull();
  });
});
