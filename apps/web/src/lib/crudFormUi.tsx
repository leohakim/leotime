import type { Translator } from './translator';

export function fieldClass(error?: string) {
  return error ? 'form-field has-error' : 'form-field';
}

export function FieldError({ id, message }: { id: string; message?: string }) {
  if (!message) {
    return null;
  }
  return (
    <span className="field-message" id={id}>
      {message}
    </span>
  );
}

export function DirectoryInactiveHeading({ count, t }: { count: number; t: Translator }) {
  if (count === 0) {
    return null;
  }

  return (
    <div className="directory-inactive-heading">
      <span>{t('inactiveDirectory')}</span>
      <strong>{count}</strong>
    </div>
  );
}

export function hasErrors(errors: Record<string, string | undefined>) {
  return Object.values(errors).some(Boolean);
}

export function rateToMinor(value: string) {
  const normalized = value.trim().replace(',', '.');
  if (!normalized) {
    return 0;
  }
  return Math.round(Number(normalized) * 100);
}

export function formatRateInput(value: number) {
  if (value === 0) {
    return '';
  }
  return (value / 100).toFixed(2);
}

export function formatMinor(value: number) {
  return (value / 100).toFixed(2);
}

export function initials(value: string) {
  const parts = value.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return 'LT';
  }
  return parts
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join('');
}
