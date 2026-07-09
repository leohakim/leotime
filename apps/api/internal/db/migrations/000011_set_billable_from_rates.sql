UPDATE time_entries
SET billable = 1
WHERE billable = 0
  AND (
    client_id IN (
      SELECT id FROM clients WHERE default_hourly_rate_minor > 0
    )
    OR project_id IN (
      SELECT id FROM projects WHERE default_hourly_rate_minor IS NOT NULL AND default_hourly_rate_minor > 0
    )
    OR project_id IN (
      SELECT p.id
      FROM projects p
      INNER JOIN clients c ON c.id = p.client_id
      WHERE c.default_hourly_rate_minor > 0
    )
  );
