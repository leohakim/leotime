UPDATE time_entries
SET client_id = (
  SELECT p.client_id
  FROM projects p
  WHERE p.id = time_entries.project_id
    AND p.user_id = time_entries.user_id
    AND p.client_id IS NOT NULL
)
WHERE client_id IS NULL
  AND project_id IS NOT NULL
  AND EXISTS (
    SELECT 1
    FROM projects p
    WHERE p.id = time_entries.project_id
      AND p.user_id = time_entries.user_id
      AND p.client_id IS NOT NULL
  );
