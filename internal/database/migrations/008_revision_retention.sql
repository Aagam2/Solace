DELETE FROM entry_revisions
WHERE id NOT IN (
    SELECT id FROM (
        SELECT id,
               ROW_NUMBER() OVER (PARTITION BY entry_id ORDER BY saved_at DESC, id DESC) AS position
        FROM entry_revisions
    )
    WHERE position <= 5
);

CREATE TRIGGER entry_revisions_retention
AFTER INSERT ON entry_revisions
BEGIN
    DELETE FROM entry_revisions
    WHERE entry_id = NEW.entry_id
      AND id NOT IN (
          SELECT id FROM entry_revisions
          WHERE entry_id = NEW.entry_id
          ORDER BY saved_at DESC, id DESC
          LIMIT 5
      );
END;
