CREATE VIRTUAL TABLE entry_search USING fts5(
    entry_id UNINDEXED,
    title,
    content,
    tags,
    attachment_captions,
    tokenize = 'unicode61 remove_diacritics 2',
    prefix = '2 3 4'
);

INSERT INTO entry_search(entry_id, title, content, tags, attachment_captions)
SELECT e.id,
       COALESCE(e.title, ''),
       e.content_text,
       COALESCE((SELECT GROUP_CONCAT(t.name, ' ') FROM tags t JOIN entry_tags et ON et.tag_id=t.id WHERE et.entry_id=e.id), ''),
       COALESCE((SELECT GROUP_CONCAT(a.caption, ' ') FROM attachments a WHERE a.entry_id=e.id AND a.caption IS NOT NULL), '')
FROM entries e;

CREATE TRIGGER entries_search_insert AFTER INSERT ON entries BEGIN
    INSERT INTO entry_search(entry_id, title, content, tags, attachment_captions)
    VALUES (new.id, COALESCE(new.title, ''), new.content_text, '', '');
END;

CREATE TRIGGER entries_search_update AFTER UPDATE OF title, content_text ON entries BEGIN
    UPDATE entry_search SET title=COALESCE(new.title, ''), content=new.content_text WHERE entry_id=new.id;
END;

CREATE TRIGGER entries_search_delete AFTER DELETE ON entries BEGIN
    DELETE FROM entry_search WHERE entry_id=old.id;
END;

CREATE TRIGGER entry_tags_search_insert AFTER INSERT ON entry_tags BEGIN
    UPDATE entry_search SET tags=COALESCE((SELECT GROUP_CONCAT(t.name, ' ') FROM tags t JOIN entry_tags et ON et.tag_id=t.id WHERE et.entry_id=new.entry_id), '') WHERE entry_id=new.entry_id;
END;

CREATE TRIGGER entry_tags_search_delete AFTER DELETE ON entry_tags BEGIN
    UPDATE entry_search SET tags=COALESCE((SELECT GROUP_CONCAT(t.name, ' ') FROM tags t JOIN entry_tags et ON et.tag_id=t.id WHERE et.entry_id=old.entry_id), '') WHERE entry_id=old.entry_id;
END;

CREATE TRIGGER tags_search_rename AFTER UPDATE OF name ON tags BEGIN
    UPDATE entry_search SET tags=COALESCE((SELECT GROUP_CONCAT(t.name, ' ') FROM tags t JOIN entry_tags et ON et.tag_id=t.id WHERE et.entry_id=entry_search.entry_id), '')
    WHERE entry_id IN (SELECT entry_id FROM entry_tags WHERE tag_id=new.id);
END;

CREATE TRIGGER attachments_search_insert AFTER INSERT ON attachments BEGIN
    UPDATE entry_search SET attachment_captions=COALESCE((SELECT GROUP_CONCAT(caption, ' ') FROM attachments WHERE entry_id=new.entry_id AND caption IS NOT NULL), '') WHERE entry_id=new.entry_id;
END;

CREATE TRIGGER attachments_search_update AFTER UPDATE OF caption ON attachments BEGIN
    UPDATE entry_search SET attachment_captions=COALESCE((SELECT GROUP_CONCAT(caption, ' ') FROM attachments WHERE entry_id=new.entry_id AND caption IS NOT NULL), '') WHERE entry_id=new.entry_id;
END;

CREATE TRIGGER attachments_search_delete AFTER DELETE ON attachments BEGIN
    UPDATE entry_search SET attachment_captions=COALESCE((SELECT GROUP_CONCAT(caption, ' ') FROM attachments WHERE entry_id=old.entry_id AND caption IS NOT NULL), '') WHERE entry_id=old.entry_id;
END;
