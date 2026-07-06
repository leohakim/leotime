import type { MessageKey } from './i18n';
import type { Translator } from './timeEntryUi';

export function PlaceholderPage({ titleKey, t }: { titleKey: MessageKey; t: Translator }) {
  return (
    <section className="page-placeholder" aria-labelledby="placeholder-title">
      <h2 id="placeholder-title">{t(titleKey)}</h2>
      <p>{t('pageComingSoon')}</p>
    </section>
  );
}
