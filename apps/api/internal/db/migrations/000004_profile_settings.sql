ALTER TABLE app_settings ADD COLUMN timezone TEXT NOT NULL DEFAULT 'Europe/Madrid';
ALTER TABLE app_settings ADD COLUMN default_currency TEXT NOT NULL DEFAULT 'EUR';
ALTER TABLE app_settings ADD COLUMN theme_mode TEXT NOT NULL DEFAULT 'solid';
