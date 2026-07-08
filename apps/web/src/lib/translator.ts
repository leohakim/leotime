import type { MessageKey } from './i18n';

export type Translator = (key: MessageKey) => string;
