import { describe, expect, test, vi } from 'vitest';
import { ApiError, isMaintenanceModeError } from './api';
import { confirmDestructiveAction } from './destructiveUi';

describe('confirmDestructiveAction', () => {
  test('returns true when the user confirms', () => {
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(true);
    expect(confirmDestructiveAction('Delete this item?')).toBe(true);
    expect(confirm).toHaveBeenCalledWith('Delete this item?');
    confirm.mockRestore();
  });

  test('returns false when the user cancels', () => {
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false);
    expect(confirmDestructiveAction('Delete this item?')).toBe(false);
    confirm.mockRestore();
  });
});

describe('isMaintenanceModeError', () => {
  test('detects maintenance_mode API errors', () => {
    const error = new ApiError(503, {
      code: 'maintenance_mode',
      message: 'server is in maintenance mode; reload the application',
    });
    expect(isMaintenanceModeError(error)).toBe(true);
  });

  test('ignores other API errors', () => {
    const error = new ApiError(500, { code: 'internal_error', message: 'boom' });
    expect(isMaintenanceModeError(error)).toBe(false);
    expect(isMaintenanceModeError(new Error('nope'))).toBe(false);
  });
});
