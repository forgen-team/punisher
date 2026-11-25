-- 학습 자료 전문 검색(W5a): SQLite FTS5, trigram 토크나이저(한국어처럼 띄어쓰기로 낱말이 갈리지 않는 글도
-- 부분 문자열로 찾는다. 3글자 미만 검색어는 앱이 LIKE 로 찾는다). modernc SQLite 3.54 에서 FTS5·trigram 실측 확인.
-- 원본 표의 트리거가 색인을 맞춘다. 강의는 공개(published)된 것만, 섹션 하나가 한 행.
-- user_id: 본 사람만 찾는 행의 주인('' = 라이브러리 강의, 누구나).

CREATE VIRTUAL TABLE IF NOT EXISTS search_index USING fts5(
    kind UNINDEXED, ref_id UNINDEXED, user_id UNINDEXED, lecture_id UNINDEXED, anchor UNINDEXED,
    title, body, tokenize = 'trigram'
);

-- 섹션 글: 블록 JSON 의 글 값(표지·종류·언어 같은 키는 뺀다).
CREATE VIEW IF NOT EXISTS search_lecture_sections AS
SELECT l.id AS lecture_id, CASE WHEN l.kind = 'library' THEN '' ELSE COALESCE(l.requested_by, '') END AS user_id,
       COALESCE(json_extract(s.value, '$.anchor'), '') AS anchor, l.title AS title,
       COALESCE(json_extract(s.value, '$.title'), '') || ' ' || COALESCE((
           SELECT group_concat(t.value, ' ') FROM json_tree(s.value, '$.blocks') t
           WHERE t.type = 'text' AND t.key NOT IN ('kind', 'type', 'lang', 'language', 'anchor', 'id', 'ref', 'term', 'href', 'url')
       ), '') AS body
FROM lectures l, json_each(l.content, '$.sections') s
WHERE l.status = 'published' AND json_valid(l.content);

CREATE TRIGGER IF NOT EXISTS search_lectures_ai AFTER INSERT ON lectures BEGIN
    INSERT INTO search_index (kind, ref_id, user_id, lecture_id, anchor, title, body)
    SELECT 'lecture', lecture_id, user_id, lecture_id, anchor, title, body FROM search_lecture_sections WHERE lecture_id = new.id;
END;
CREATE TRIGGER IF NOT EXISTS search_lectures_au AFTER UPDATE OF content, title, status ON lectures BEGIN
    DELETE FROM search_index WHERE kind = 'lecture' AND ref_id = old.id;
    INSERT INTO search_index (kind, ref_id, user_id, lecture_id, anchor, title, body)
    SELECT 'lecture', lecture_id, user_id, lecture_id, anchor, title, body FROM search_lecture_sections WHERE lecture_id = new.id;
END;
CREATE TRIGGER IF NOT EXISTS search_lectures_ad AFTER DELETE ON lectures BEGIN
    DELETE FROM search_index WHERE kind = 'lecture' AND ref_id = old.id;
END;

CREATE TRIGGER IF NOT EXISTS search_notes_ai AFTER INSERT ON notes BEGIN
    INSERT INTO search_index (kind, ref_id, user_id, lecture_id, anchor, title, body)
    VALUES ('note', new.id, new.user_id, COALESCE(new.lecture_id, ''), new.anchor, new.quote, new.body);
END;
CREATE TRIGGER IF NOT EXISTS search_notes_au AFTER UPDATE ON notes BEGIN
    DELETE FROM search_index WHERE kind = 'note' AND ref_id = old.id;
    INSERT INTO search_index (kind, ref_id, user_id, lecture_id, anchor, title, body)
    VALUES ('note', new.id, new.user_id, COALESCE(new.lecture_id, ''), new.anchor, new.quote, new.body);
END;
CREATE TRIGGER IF NOT EXISTS search_notes_ad AFTER DELETE ON notes BEGIN
    DELETE FROM search_index WHERE kind = 'note' AND ref_id = old.id;
END;

-- 강의 중 질문·구절 대화(답이 붙으면 다시 색인).
CREATE TRIGGER IF NOT EXISTS search_questions_ai AFTER INSERT ON lecture_questions BEGIN
    INSERT INTO search_index (kind, ref_id, user_id, lecture_id, anchor, title, body)
    VALUES ('question', new.id, new.user_id, COALESCE(new.lecture_id, ''), new.anchor, new.quote, new.question || ' ' || new.answer);
END;
CREATE TRIGGER IF NOT EXISTS search_questions_au AFTER UPDATE OF question, answer, quote ON lecture_questions BEGIN
    DELETE FROM search_index WHERE kind = 'question' AND ref_id = old.id;
    INSERT INTO search_index (kind, ref_id, user_id, lecture_id, anchor, title, body)
    VALUES ('question', new.id, new.user_id, COALESCE(new.lecture_id, ''), new.anchor, new.quote, new.question || ' ' || new.answer);
END;
CREATE TRIGGER IF NOT EXISTS search_questions_ad AFTER DELETE ON lecture_questions BEGIN
    DELETE FROM search_index WHERE kind = 'question' AND ref_id = old.id;
END;

-- 질문 해설: 질문 + 해설 본문 글.
CREATE VIEW IF NOT EXISTS search_inquiry_text AS
SELECT i.id, i.user_id, i.question AS title,
       COALESCE((SELECT group_concat(t.value, ' ') FROM json_tree(i.content) t
                 WHERE t.type = 'text' AND t.key NOT IN ('kind', 'type', 'lang', 'language', 'anchor', 'id', 'ref', 'term', 'href', 'url')), '') AS body
FROM inquiries i WHERE json_valid(i.content);

CREATE TRIGGER IF NOT EXISTS search_inquiries_ai AFTER INSERT ON inquiries BEGIN
    INSERT INTO search_index (kind, ref_id, user_id, lecture_id, anchor, title, body)
    SELECT 'inquiry', id, user_id, '', '', title, body FROM search_inquiry_text WHERE id = new.id;
END;
CREATE TRIGGER IF NOT EXISTS search_inquiries_au AFTER UPDATE OF question, content ON inquiries BEGIN
    DELETE FROM search_index WHERE kind = 'inquiry' AND ref_id = old.id;
    INSERT INTO search_index (kind, ref_id, user_id, lecture_id, anchor, title, body)
    SELECT 'inquiry', id, user_id, '', '', title, body FROM search_inquiry_text WHERE id = new.id;
END;
CREATE TRIGGER IF NOT EXISTS search_inquiries_ad AFTER DELETE ON inquiries BEGIN
    DELETE FROM search_index WHERE kind = 'inquiry' AND ref_id = old.id;
END;

-- 이미 있는 자료를 한 번 색인한다.
INSERT INTO search_index (kind, ref_id, user_id, lecture_id, anchor, title, body)
SELECT 'lecture', lecture_id, user_id, lecture_id, anchor, title, body FROM search_lecture_sections;
INSERT INTO search_index (kind, ref_id, user_id, lecture_id, anchor, title, body)
SELECT 'note', id, user_id, COALESCE(lecture_id, ''), anchor, quote, body FROM notes;
INSERT INTO search_index (kind, ref_id, user_id, lecture_id, anchor, title, body)
SELECT 'question', id, user_id, COALESCE(lecture_id, ''), anchor, quote, question || ' ' || answer FROM lecture_questions;
INSERT INTO search_index (kind, ref_id, user_id, lecture_id, anchor, title, body)
SELECT 'inquiry', id, user_id, '', '', title, body FROM search_inquiry_text;
