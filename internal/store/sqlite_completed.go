package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/wooo-jin/punisher/internal/curriculum"
	"github.com/wooo-jin/punisher/internal/models"
)

// AreaOther groups inquiries with no concept on the map.
const AreaOther = "other"

// completedItemsSQL: 완료한 강의 + 답을 받은 질문 스레드. 분야는 지도의 영역을 그대로 쓴다.
// '더 깊게'로 본 강의는 원 강의의 분야에 넣어 원 강의 바로 아래에 묶인다. $1 = user.
var completedItemsSQL = `
	WITH items AS (
		SELECT 'lecture' AS kind, l.id AS id, l.title AS title, l.concept_id AS concept_id, c.name AS concept_name,
			COALESCE((SELECT pc.area FROM lecture_links k
				JOIN lectures pl ON pl.id = k.parent_lecture_id JOIN concepts pc ON pc.id = pl.concept_id
				WHERE k.user_id = $1 AND ` + linkedChildSQL + ` = l.id ORDER BY k.created_at DESC LIMIT 1), c.area) AS area,
			l.depth AS depth, l.kind AS lecture_kind, ` + parentLectureSQL("l.id") + ` AS parent_id, 0 AS followups,
			p.last_seen_at AS activity_at, l.title || ' ' || c.name AS search
		FROM user_lecture_progress p
		JOIN lectures l ON l.id = p.lecture_id AND (l.kind = 'library' OR l.requested_by = p.user_id)
		JOIN concepts c ON c.id = l.concept_id
		WHERE p.user_id = $1 AND p.completed_at IS NOT NULL
		UNION ALL
		SELECT 'inquiry', i.id, COALESCE(NULLIF(json_extract(i.content, '$.title'), ''), i.question), COALESCE(c.id, ''), COALESCE(c.name, ''),
			COALESCE(c.area, '` + AreaOther + `'), 0, '', '',
			(SELECT COUNT(*) FROM inquiry_followups f WHERE f.inquiry_id = i.id),
			i.updated_at,
			i.question || ' ' || COALESCE(json_extract(i.content, '$.title'), '') || ' ' || COALESCE(
				(SELECT group_concat(n.name, ' ') FROM concepts n WHERE n.id IN (
					SELECT value FROM json_each(i.concept_ids) UNION SELECT value FROM json_each(i.proposed_concept_ids))), '')
		FROM inquiries i
		LEFT JOIN concepts c ON c.id = COALESCE(json_extract(i.concept_ids, '$[0]'), json_extract(i.proposed_concept_ids, '$[0]'))
		WHERE i.user_id = $1 AND i.answered_at IS NOT NULL
	)`

// likePattern escapes a user search term for LIKE ... ESCAPE '\'.
func likePattern(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	return "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q) + "%"
}

// CompletedFilter narrows the completed list (server-side; the client never filters).
type CompletedFilter struct {
	Area  string // "" = all areas
	Query string
	Limit int
}

// ListCompleted returns completed lectures and answered inquiries, most recent first.
func (s *SQLiteStore) ListCompleted(ctx context.Context, userID string, f CompletedFilter, cursor string) ([]models.CompletedItem, string, error) {
	cur, err := decodeCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	items, err := collect(s.c().query(ctx, completedItemsSQL+`
		SELECT kind, id, title, concept_id, concept_name, area, depth, lecture_kind, parent_id, followups, activity_at
		FROM items
		WHERE ($2 = '' OR search LIKE $2 ESCAPE '\') AND ($3 = '' OR area = $3) AND (activity_at, id) < ($4, $5)
		ORDER BY activity_at DESC, id DESC LIMIT $6`, userID, likePattern(f.Query), f.Area, cur.At, cur.ID, f.Limit+1),
		func(r rowScanner) (models.CompletedItem, error) {
			var m models.CompletedItem
			err := r.Scan(&m.Kind, &m.ID, &m.Title, &m.ConceptID, &m.ConceptName, &m.Area, &m.Depth, &m.LectureKind,
				&m.ParentLectureID, &m.FollowupCount, ts(&m.LastActivityAt))
			return m, err
		})
	if err != nil {
		return nil, "", fmt.Errorf("list completed: %w", err)
	}
	next := ""
	if len(items) > f.Limit {
		items = items[:f.Limit]
		last := items[len(items)-1]
		next = encodeCursor(last.LastActivityAt, last.ID)
	}
	return items, next, nil
}

// CompletedAreas counts items per area for the search term, in map area order
// (areas with no items are left out; 기타 last).
func (s *SQLiteStore) CompletedAreas(ctx context.Context, userID, query string) ([]models.CompletedArea, error) {
	type row struct {
		area  string
		count int
	}
	list, err := collect(s.c().query(ctx, completedItemsSQL+`
		SELECT area, COUNT(*) FROM items WHERE ($2 = '' OR search LIKE $2 ESCAPE '\') GROUP BY area`, userID, likePattern(query)),
		func(r rowScanner) (row, error) {
			var c row
			err := r.Scan(&c.area, &c.count)
			return c, err
		})
	if err != nil {
		return nil, fmt.Errorf("completed areas: %w", err)
	}
	counts := make(map[string]int, len(list))
	for _, c := range list {
		counts[c.area] = c.count
	}
	out := []models.CompletedArea{}
	for _, area := range append(append([]string{}, curriculum.Areas...), AreaOther) {
		if counts[area] > 0 {
			out = append(out, models.CompletedArea{Area: area, Count: counts[area]})
		}
	}
	return out, nil
}
