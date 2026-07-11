import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { SurfaceEmpty, SurfaceError, SurfaceLoading } from './feedbackUi';

describe('feedbackUi', () => {
  it('renders loading feedback with a sync pill', () => {
    render(<SurfaceLoading label="Cargando…" />);

    expect(screen.getByText('Cargando…')).toHaveClass('sync-pill');
    expect(screen.getByText('Cargando…').closest('.surface-feedback-loading')).toBeTruthy();
  });

  it('renders error feedback with optional retry', () => {
    const onRetry = vi.fn();
    render(<SurfaceError message="No se pudo cargar" onRetry={onRetry} retryLabel="Reintentar" />);

    expect(screen.getByRole('alert')).toHaveTextContent('No se pudo cargar');
    fireEvent.click(screen.getByRole('button', { name: 'Reintentar' }));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it('renders empty feedback with panel styling', () => {
    render(
      <SurfaceEmpty>
        <p>Sin resultados</p>
      </SurfaceEmpty>,
    );

    expect(screen.getByText('Sin resultados').closest('.panel-empty-state')).toBeTruthy();
  });
});
