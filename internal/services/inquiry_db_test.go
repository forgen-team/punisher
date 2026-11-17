package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/store"
)

func inquiryMarkup(a, b, proposedSlug string, withSources bool) string {
	sources := ""
	if withSources {
		sources = "@@sources\n- [doc](https://example.org/doc)\n"
	}
	return "@@lecture\ntitle: 풀 고갈과 CORS\nconcepts: " + a + ", " + b + "\n" +
		"new_concept: " + proposedSlug + " | 헬스체크 설계 | infra-ops | " + a + " | 무엇을 확인하나\n" +
		"@@section summary title=답\n@@markdown\n[개념A](term:" + a + ") 때문입니다.\n" +
		"@@section causes title=원인\n@@markdown\n1 → 2 → [개념B](term:" + b + ")\n" + sources +
		"@@section assumptions title=추정\n@@markdown\n- 추정: 인그레스가 있다\n" +
		"@@section verify title=확인\n@@code lang=bash\ncurl -i x\n" +
		"@@section fix title=대처\n@@markdown\n풀을 늘립니다.\n@@end"
}

func followupMarkup(link, slug string) string {
	return "@@lecture\ntitle: 후속 답\nconcepts: " + link + "\nnew_concept: " + slug + " | 새 개념 | infra-ops\n" +
		"@@section answer title=왜 그런가\n@@markdown\n[개념](term:" + link + ") 때문입니다.\n@@end"
}

func threeConcepts(t *testing.T, s *Service, user string) (a, b, c string) {
	t.Helper()
	catalog, err := s.Store.ConceptCatalog(context.Background(), user, "ko")
	if err != nil || len(catalog) < 3 {
		t.Fatalf("catalog: %d %v", len(catalog), err)
	}
	return catalog[0].ID, catalog[1].ID, catalog[2].ID
}

// Rewrite on invalid output, cache, attach to the map, 다 봤어요, failed regeneration keeps the answer and model.
func TestInquiryRewritesThenCachesAndAttachesToMap(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	a, b, _ := threeConcepts(t, s, user)
	slug := "infra.test-probe"
	fake := &sequenceLLM{texts: []string{inquiryMarkup(a, b, slug, false), inquiryMarkup(a, b, slug, true)}}
	s.LLM = fake
	q, _, err := s.Store.CreateInquiry(ctx, user, "pod가 커넥션 풀을 다 물고 있는데 CORS가 나요", time.Minute, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.runInquiry(ctx, "k", user, q); err != nil {
		t.Fatal(err)
	}
	if len(fake.users) != 2 || !strings.Contains(fake.users[1], "causes must cite") {
		t.Fatal("the rewrite must receive the validation problems")
	}
	got, err := s.GetInquiry(ctx, user, q.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "answered" || strings.Join(got.ConceptIDs, ",") != a+","+b || strings.Join(got.ProposedConceptIDs, ",") != slug ||
		len(got.Concepts) != 3 || got.Concepts[2].Kind != "proposed" {
		t.Fatalf("answered inquiry = %+v", got)
	}
	data := &models.GraphData{States: map[string]string{}}
	if err := s.Store.LoadMapExtras(ctx, user, data); err != nil {
		t.Fatal(err)
	}
	if len(data.Inquiries) != 1 || len(data.Inquiries[0].ConceptIDs) != 3 {
		t.Fatalf("map inquiries = %+v", data.Inquiries)
	}
	byConcept, _ := s.Store.ListInquiries(ctx, user, store.InquiryFilter{ConceptID: slug})
	if len(byConcept) != 1 || strings.Contains(string(byConcept[0].Content), "sections") || !strings.Contains(string(byConcept[0].Content), "풀 고갈과 CORS") {
		t.Fatalf("concept filter = %+v", byConcept)
	}
	if err := s.Store.SetInquiryRead(ctx, user, q.ID, true); err != nil {
		t.Fatal(err)
	}
	open, _ := s.Store.ListInquiries(ctx, user, store.InquiryFilter{Open: true})
	all, _ := s.Store.ListInquiries(ctx, user, store.InquiryFilter{})
	if len(open) != 0 || len(all) != 1 {
		t.Fatalf("open=%d all=%d", len(open), len(all))
	}
	s.LLM = &sequenceLLM{texts: []string{"깨진 출력"}}
	if _, started, err := s.Store.SetInquiryPending(ctx, user, q.ID); err != nil || !started {
		t.Fatal(started, err)
	}
	if _, started, err := s.Store.SetInquiryPending(ctx, user, q.ID); err != nil || started {
		t.Fatal("a second regeneration must not start", started, err)
	}
	if err := s.runInquiry(ctx, "k", user, got); err != nil {
		t.Fatal(err)
	}
	after, _ := s.Store.GetInquiry(ctx, user, q.ID)
	if after.Status != "answered" || after.Error == "" || len(after.Content) < 10 || after.Model != "fake" {
		t.Fatalf("failed regeneration must keep the previous answer: %+v", after)
	}
}

func TestInquiryFailureDedupeAndSweep(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	s.LLM = &sequenceLLM{texts: []string{"@@lecture\ntitle: x\n@@end"}}
	q, _, _ := s.Store.CreateInquiry(ctx, user, "질문", time.Minute, 3)
	if err := s.runInquiry(ctx, "k", user, q); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Store.GetInquiry(ctx, user, q.ID)
	if got.Status != "failed" || got.AnsweredAt != nil || PublicInquiry(got).Error != inquiryFailedMessage {
		t.Fatalf("got %+v", got)
	}
	a, created, _ := s.Store.CreateInquiry(ctx, user, "같은 질문", time.Minute, 3)
	b, again, _ := s.Store.CreateInquiry(ctx, user, "같은 질문", time.Minute, 3)
	if !created || again || a.ID != b.ID {
		t.Fatalf("double submit: %v %v", created, again)
	}
	if n, _ := s.Store.SweepStaleInquiries(ctx, time.Hour); n != 0 {
		t.Fatalf("fresh pending swept: %d", n)
	}
	if _, err := s.Store.SweepStaleInquiries(ctx, -time.Second); err != nil {
		t.Fatal(err)
	}
	if st, _ := s.Store.GetInquiry(ctx, user, a.ID); st.Status != "failed" {
		t.Fatalf("after sweep: %+v", st)
	}
}

func TestFollowupThread(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	a, b, c := threeConcepts(t, s, user)
	s.LLM = &sequenceLLM{texts: []string{inquiryMarkup(a, b, "infra.fq-one", true)}}
	q, _, _ := s.Store.CreateInquiry(ctx, user, "원 질문", time.Minute, 3)
	if err := s.runInquiry(ctx, "k", user, q); err != nil {
		t.Fatal(err)
	}
	_ = s.Store.SetInquiryRead(ctx, user, q.ID, true)
	fake := &sequenceLLM{texts: []string{"깨진 출력", followupMarkup(c, "infra.fq-two")}}
	s.LLM = fake
	f, err := s.Store.CreateFollowup(ctx, user, q.ID, "왜요?", "고른 구절")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.runFollowup(ctx, "k", user, q.ID, f); err != nil {
		t.Fatal(err)
	}
	if len(fake.users) != 2 || !strings.Contains(fake.users[0], "고른 구절") || !strings.Contains(fake.users[0], "원 질문") {
		t.Fatalf("the follow-up prompt must carry the thread and quote: %d", len(fake.users))
	}
	got, err := s.GetInquiry(ctx, user, q.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Followups) != 1 || got.Followups[0].Status != "answered" || got.FollowupCount != 1 || got.ReadAt != nil ||
		!strings.Contains(string(got.Followups[0].Section), "fu-") {
		t.Fatalf("thread = %+v", got)
	}
	if strings.Join(got.ConceptIDs, ",") != a+","+b+","+c || strings.Join(got.ProposedConceptIDs, ",") != "infra.fq-one,infra.fq-two" {
		t.Fatalf("thread concepts = %v %v", got.ConceptIDs, got.ProposedConceptIDs)
	}
	s.LLM = &sequenceLLM{texts: []string{inquiryMarkup(b, a, "infra.fq-one", true)}}
	if err := s.runInquiry(ctx, "k", user, q); err != nil {
		t.Fatal(err)
	}
	if merged, _ := s.Store.GetInquiry(ctx, user, q.ID); strings.Join(merged.ConceptIDs, ",") != b+","+a+","+c {
		t.Fatalf("regeneration lost thread concepts: %v", merged.ConceptIDs)
	}
	next, err := s.Store.CreateFollowup(ctx, user, q.ID, "다음", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Store.CreateFollowup(ctx, user, q.ID, "동시", ""); !errors.Is(err, store.ErrInquiryBusy) {
		t.Fatalf("second pending follow-up: %v", err)
	}
	if _, err := s.Store.SetFollowupPending(ctx, user, "other-thread", next.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("retry must check the thread: %v", err)
	}
	if _, err := s.RegenerateInquiry(ctx, user, q.ID); !errors.Is(err, ErrInquiryBusy) {
		t.Fatalf("no regeneration while a follow-up is being written: %v", err)
	}
	if _, err := s.StartFollowup(ctx, user, q.ID, "또", ""); !errors.Is(err, ErrInquiryBusy) {
		t.Fatalf("one at a time: %v", err)
	}
	areas, err := s.Store.CompletedAreas(ctx, user, "")
	if err != nil || len(areas) != 1 || areas[0].Count != 1 {
		t.Fatalf("areas = %+v %v", areas, err)
	}
	items, _, err := s.Store.ListCompleted(ctx, user, store.CompletedFilter{Area: areas[0].Area, Query: "풀 고갈", Limit: 10}, "")
	if err != nil || len(items) != 1 || items[0].Kind != "inquiry" || items[0].FollowupCount != 2 {
		t.Fatalf("items = %+v %v", items, err)
	}
	if none, _, _ := s.Store.ListCompleted(ctx, user, store.CompletedFilter{Query: "100%없는말", Limit: 10}, ""); len(none) != 0 {
		t.Fatalf("search = %+v", none)
	}
}
