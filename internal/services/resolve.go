package services

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wooo-jin/punisher/internal/curriculum"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/llm"
	"github.com/wooo-jin/punisher/internal/llmjson"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/prompts"
	"github.com/wooo-jin/punisher/internal/store"
)

// Resolution methods.
const (
	ResolvedByAlias    = "alias"
	ResolvedByLLM      = "llm"
	ResolvedByProposal = "proposed"
	ResolvedNone       = "none"
)

// Resolution maps a free query to concepts and a starting point.
type Resolution struct {
	Query      string   `json:"query"`
	Method     string   `json:"method"`
	ConceptIDs []string `json:"conceptIds"`
	// ProposedConceptID is set when a new proposed concept was added.
	ProposedConceptID string `json:"proposedConceptId,omitempty"`
	// StartingConceptID: earliest not-done concept on the path to ConceptIDs[0]
	// whose prerequisites are done ("" when everything is learned).
	StartingConceptID string            `json:"startingConceptId"`
	Path              []models.PathStep `json:"path"`
	Error             string            `json:"error,omitempty"`
}

// ResolveOutput is the haiku resolver JSON.
type ResolveOutput struct {
	ConceptIDs []string                `json:"concept_ids"`
	Confidence float64                 `json:"confidence"`
	Proposed   *models.ProposedConcept `json:"proposed"`
}

var proposedIDRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*\.[a-z0-9]+(-[a-z0-9]+)*$`)

// SanitizeResolveOutput keeps only existing concept ids (max 3) and accepts a
// proposal only when it is well-formed, new, in a known area, and has at
// least one prerequisite that exists. Pure; tested.
func SanitizeResolveOutput(out ResolveOutput, existing map[string]bool) ([]string, *models.ProposedConcept) {
	ids := []string{}
	seen := map[string]bool{}
	for _, id := range out.ConceptIDs {
		id = strings.TrimSpace(id)
		if existing[id] && !seen[id] && len(ids) < 3 {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	p := out.Proposed
	if len(ids) > 0 || p == nil {
		return ids, nil
	}
	p.ID = strings.TrimSpace(strings.ToLower(p.ID))
	if existing[p.ID] {
		return []string{p.ID}, nil
	}
	if !proposedIDRe.MatchString(p.ID) || !curriculum.IsValidArea(p.Area) || strings.TrimSpace(p.Name) == "" {
		return ids, nil
	}
	if p.Depth < 1 || p.Depth > 3 {
		p.Depth = 1
	}
	prereqs := []string{}
	for _, pre := range p.Prerequisites {
		if existing[pre] && len(prereqs) < 3 {
			prereqs = append(prereqs, pre)
		}
	}
	if len(prereqs) == 0 {
		return ids, nil
	}
	p.Prerequisites = prereqs
	objectives := []string{}
	for _, o := range p.Objectives {
		if o = strings.TrimSpace(o); o != "" && utf8.RuneCountInString(o) <= 200 && len(objectives) < 6 {
			objectives = append(objectives, o)
		}
	}
	if len(objectives) == 0 {
		return ids, nil
	}
	p.Objectives = objectives
	return ids, p
}

// Resolve maps a query to concepts: exact alias/name/id first, then the fast
// model (when a key is available). keyFn is called lazily.
func (s *Service) Resolve(ctx context.Context, userID, query string, keyFn func() (string, error)) (Resolution, error) {
	query = strings.TrimSpace(query)
	res := Resolution{Query: query, Method: ResolvedNone, ConceptIDs: []string{}, Path: []models.PathStep{}}
	if query == "" {
		return res, errors.New("query is required")
	}
	index, err := s.Store.AliasIndex(ctx, userID)
	if err != nil {
		return res, err
	}
	if id, ok := learning.ResolveExact(index, query); ok {
		res.Method, res.ConceptIDs = ResolvedByAlias, []string{id}
	} else {
		if !s.AllowUser(userID) {
			return res, ErrNotOwner
		}
		apiKey, err := keyFn()
		if err != nil {
			return res, err
		}
		if err := s.CheckBudget(ctx, userID, false); err != nil {
			return res, err
		}
		if err := s.resolveWithLLM(ctx, userID, apiKey, &res); err != nil {
			return res, err
		}
	}
	if len(res.ConceptIDs) > 0 {
		if err := s.fillPath(ctx, userID, &res); err != nil {
			return res, err
		}
	}
	return res, nil
}

func (s *Service) resolveWithLLM(ctx context.Context, userID, apiKey string, res *Resolution) error {
	catalog, err := s.Store.ConceptCatalog(ctx, userID)
	if err != nil {
		return err
	}
	existing := make(map[string]bool, len(catalog))
	candidates := make([]prompts.ResolveCandidate, 0, len(catalog))
	for _, c := range catalog {
		existing[c.ID] = true
		candidates = append(candidates, prompts.ResolveCandidate{ID: c.ID, Name: c.Name, Area: c.Area, Summary: c.Summary})
	}
	var u usage
	text, err := s.complete(ctx, apiKey, llm.MessagesRequest{
		Model: s.Cfg.FastModel, System: s.UserPrompts(ctx, userID).Text(prompts.BlockResolveSystem, nil),
		User: prompts.Resolve(res.Query, candidates, curriculum.Areas), MaxTokens: 1500, Timeout: 60 * time.Second,
	}, &u)
	if err == nil {
		var out ResolveOutput
		out, err = llmjson.TryParseJSON[ResolveOutput](text)
		if err == nil {
			ids, proposed := SanitizeResolveOutput(out, existing)
			res.ConceptIDs, res.Method = ids, ResolvedByLLM
			if proposed != nil {
				id, perr := s.Store.ProposeConcept(ctx, userID, *proposed)
				if perr != nil {
					err = perr
				} else {
					res.ConceptIDs, res.Method, res.ProposedConceptID = []string{id}, ResolvedByProposal, id
				}
			}
		}
	}
	s.recordUsage(ctx, userID, "resolve", "", u, err)
	return err
}

// fillPath computes the prerequisite path to the first concept and the
// starting point with the user's states.
func (s *Service) fillPath(ctx context.Context, userID string, res *Resolution) error {
	steps, start, err := s.GoalPath(ctx, userID, res.ConceptIDs[0])
	if err != nil {
		return err
	}
	res.Path, res.StartingConceptID = steps, start
	return nil
}

// GoalPath returns the prerequisite chain to target with user states and the
// starting point.
func (s *Service) GoalPath(ctx context.Context, userID, target string) ([]models.PathStep, string, error) {
	catalog, err := s.Store.ConceptCatalog(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	edges, err := s.Store.PrerequisiteEdges(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	byID := make(map[string]store.ConceptRef, len(catalog))
	depthOf := make(map[string]int, len(catalog))
	states := make(map[string]string, len(catalog))
	for _, c := range catalog {
		byID[c.ID], depthOf[c.ID], states[c.ID] = c, c.Depth, c.State
	}
	path := learning.PathTo(target, edges, depthOf)
	start := learning.StartingPoint(path, edges, states)
	prereqs := map[string][]string{}
	for _, e := range edges {
		prereqs[e.To] = append(prereqs[e.To], e.From)
	}
	steps := make([]models.PathStep, 0, len(path))
	for _, id := range path {
		c := byID[id]
		locked := false
		if !learning.IsDone(c.State) {
			for _, p := range prereqs[id] {
				if _, inCatalog := byID[p]; inCatalog && !learning.IsDone(states[p]) {
					locked = true
					break
				}
			}
		}
		steps = append(steps, models.PathStep{ConceptID: id, Name: c.Name, Area: c.Area, Depth: c.Depth,
			State: learning.NormalizeState(c.State), Locked: locked, IsTarget: id == target, IsNext: id == start})
	}
	return steps, start, nil
}

// ResolveStudyRequests is the worker step: resolve the owner's unresolved
// requests (alias, then LLM). Returns how many were resolved.
func (s *Service) ResolveStudyRequests(ctx context.Context) (int, error) {
	n, _, err := s.ResolveStudyRequestsWithStarts(ctx)
	return n, err
}

// ResolveStudyRequestsWithStarts is ResolveStudyRequests that also returns
// the distinct starting concept IDs of the requests resolved in this batch
// (input for PregenerateStudyStarts).
func (s *Service) ResolveStudyRequestsWithStarts(ctx context.Context) (int, []string, error) {
	owner := s.Cfg.OwnerUserID
	if owner == "" {
		return 0, nil, nil
	}
	pending, err := s.Store.UnresolvedStudyRequests(ctx, owner, 3, 20)
	if err != nil {
		return 0, nil, err
	}
	resolved := 0
	var starts []string
	seen := map[string]bool{}
	for _, r := range pending {
		if IsResolving(r.ID) {
			continue // 즉시 해석(ResolveRequestNow)이 이미 처리 중
		}
		res, err := s.Resolve(ctx, owner, r.Query, func() (string, error) { return s.Keys.UserKey(ctx, owner) })
		if errors.Is(err, ErrBudgetExceeded) || errors.Is(err, ErrNoAPIKey) {
			return resolved, starts, err // stop the batch; retry on the next tick without burning attempts
		}
		msg := ""
		if err != nil {
			msg = err.Error()
		} else if len(res.ConceptIDs) == 0 {
			msg = "no matching concept"
		}
		if rerr := s.Store.RecordResolution(ctx, r.ID, res.ConceptIDs, res.StartingConceptID, msg); rerr != nil {
			return resolved, starts, rerr
		}
		if len(res.ConceptIDs) > 0 {
			resolved++
			for _, start := range studyPregenerateTargets(res) {
				if !seen[start] {
					seen[start] = true
					starts = append(starts, start)
				}
			}
		}
	}
	return resolved, starts, nil
}
