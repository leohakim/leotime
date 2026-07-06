import { render } from '@testing-library/react';
import { describe, expect, test } from 'vitest';
import { LeotimeLogo, LeotimeMark } from './leotimeLogo';

describe('leotimeLogo', () => {
  test('renders the mark with accessible title', () => {
    const { container } = render(<LeotimeMark title="leotime" />);
    expect(container.querySelector('title')?.textContent).toBe('leotime');
    expect(container.querySelector('svg')).toBeTruthy();
  });

  test('renders the wordmark logo', () => {
    const { container } = render(<LeotimeLogo />);
    expect(container.querySelector('.leotime-wordmark')?.textContent).toBe('leotime');
  });
});
