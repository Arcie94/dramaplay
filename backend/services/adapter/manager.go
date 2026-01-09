package adapter

import (
	"dramabang/models"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

type Manager struct {
	providers    map[string]Provider
	providerList []Provider
	cache        *cache.Cache
}

// Helper for caching detailed response
type CachedDetail struct {
	Drama    *models.Drama
	Episodes []models.Episode
}

func NewManager() *Manager {
	db := NewDramaboxProvider()
	ml := NewMeloloProvider()
	ns := NewNetshortProvider()
	st := NewStarshortProvider()
	// New Providers
	fs := NewFreeShortProvider()
	sm := NewShortMaxProvider()
	dd := NewDramaDashProvider()
	hs := NewHiShortProvider()
	fr := NewFlickReelsProvider()
	dw := NewDramaWaveProvider()

	return &Manager{
		providers: map[string]Provider{
			db.GetID(): db,
			ml.GetID(): ml,
			ns.GetID(): ns,
			st.GetID(): st,
			fs.GetID(): fs,
			sm.GetID(): sm,
			dd.GetID(): dd,
			hs.GetID(): hs,
			fr.GetID(): fr,
			dw.GetID(): dw,
		},
		providerList: []Provider{db, ml, ns, st, fs, sm, dd, hs, fr, dw},
		cache:        cache.New(30*time.Minute, 60*time.Minute),
	}
}

// resolveProvider parses "prefix:id" and returns the provider and raw ID
func (m *Manager) resolveProvider(fullID string) (Provider, string, error) {
	parts := strings.SplitN(fullID, ":", 2)
	if len(parts) < 2 {
		// Legacy fallback: assume Dramabox if no prefix
		if p, ok := m.providers["dramabox"]; ok {
			return p, fullID, nil
		}
		return nil, "", fmt.Errorf("invalid id format")
	}

	prefix := parts[0]
	rawID := parts[1]

	if p, ok := m.providers[prefix]; ok {
		return p, rawID, nil
	}

	return nil, "", fmt.Errorf("unknown provider: %s", prefix)
}

func (m *Manager) GetTrending() ([]models.Drama, error) {
	// Check Cache
	if x, found := m.cache.Get("trending"); found {
		return x.([]models.Drama), nil
	}

	var wg sync.WaitGroup
	results := make([][]models.Drama, len(m.providerList))
	errors := make([]error, len(m.providerList))

	for i, p := range m.providerList {
		wg.Add(1)
		go func(index int, prov Provider) {
			defer wg.Done()
			res, err := prov.GetTrending()
			if err != nil {
				errors[index] = err
				// Log error but continue?
				fmt.Printf("Error fetching trending from %s: %v\n", prov.GetID(), err)
				return
			}
			results[index] = res
		}(i, p)
	}
	wg.Wait()

	// Merge Round Robin
	var merged []models.Drama
	maxLen := 0
	for _, res := range results {
		if len(res) > maxLen {
			maxLen = len(res)
		}
	}

	for i := 0; i < maxLen; i++ {
		for _, res := range results {
			if i < len(res) {
				merged = append(merged, res[i])
			}
		}
	}

	// Set Cache (30 mins) ONLY if not empty
	if len(merged) > 0 {
		m.cache.Set("trending", merged, 30*time.Minute)
	}

	return merged, nil
}

func (m *Manager) Search(query string) ([]models.Drama, error) {
	// Check Cache
	cacheKey := fmt.Sprintf("search:%s", query)
	if x, found := m.cache.Get(cacheKey); found {
		return x.([]models.Drama), nil
	}

	var wg sync.WaitGroup
	results := make([][]models.Drama, len(m.providerList))

	for i, p := range m.providerList {
		wg.Add(1)
		go func(index int, prov Provider) {
			defer wg.Done()
			res, err := prov.Search(query)
			if err != nil {
				fmt.Printf("Error searching %s: %v\n", prov.GetID(), err)
				return
			}
			results[index] = res
		}(i, p)
	}
	wg.Wait()

	// Merge Round Robin
	var merged []models.Drama
	maxLen := 0
	for _, res := range results {
		if len(res) > maxLen {
			maxLen = len(res)
		}
	}

	for i := 0; i < maxLen; i++ {
		for _, res := range results {
			if i < len(res) {
				merged = append(merged, res[i])
			}
		}
	}

	// Set Cache (10 mins)
	if len(merged) > 0 {
		m.cache.Set(cacheKey, merged, 10*time.Minute)
	}

	return merged, nil
}

func (m *Manager) GetLatest(page int) ([]models.Drama, error) {
	// Check Cache
	cacheKey := fmt.Sprintf("latest:%d", page)
	if x, found := m.cache.Get(cacheKey); found {
		return x.([]models.Drama), nil
	}

	var wg sync.WaitGroup
	results := make([][]models.Drama, len(m.providerList))

	for i, p := range m.providerList {
		wg.Add(1)
		go func(index int, prov Provider) {
			defer wg.Done()
			res, err := prov.GetLatest(page)
			if err != nil {
				// Log error but continue
				fmt.Printf("Error fetching latest from %s: %v\n", prov.GetID(), err)
				return
			}
			results[index] = res
		}(i, p)
	}
	wg.Wait()

	// Merge Round Robin
	var merged []models.Drama
	maxLen := 0
	for _, res := range results {
		if len(res) > maxLen {
			maxLen = len(res)
		}
	}

	for i := 0; i < maxLen; i++ {
		for _, res := range results {
			if i < len(res) {
				merged = append(merged, res[i])
			}
		}
	}

	// Set Cache (15 mins)
	if len(merged) > 0 {
		m.cache.Set(cacheKey, merged, 15*time.Minute)
	}

	return merged, nil
}

func (m *Manager) GetLatestFromProvider(providerID string, page int) ([]models.Drama, error) {
	// Check Cache
	cacheKey := fmt.Sprintf("latest:%s:%d", providerID, page)
	if x, found := m.cache.Get(cacheKey); found {
		return x.([]models.Drama), nil
	}

	p, ok := m.providers[providerID]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", providerID)
	}

	res, err := p.GetLatest(page)
	if err != nil {
		return nil, err
	}

	// Set Cache (15 mins)
	if len(res) > 0 {
		m.cache.Set(cacheKey, res, 15*time.Minute)
	}

	return res, nil
}

func (m *Manager) GetDetail(fullID string) (*models.Drama, []models.Episode, error) {
	// Check Cache
	cacheKey := fmt.Sprintf("detail:%s", fullID)
	if x, found := m.cache.Get(cacheKey); found {
		cached := x.(CachedDetail)
		return cached.Drama, cached.Episodes, nil
	}

	p, rawID, err := m.resolveProvider(fullID)
	if err != nil {
		return nil, nil, err
	}
	drama, episodes, err := p.GetDetail(rawID)
	if err == nil {
		// Set Cache (60 mins)
		m.cache.Set(cacheKey, CachedDetail{Drama: drama, Episodes: episodes}, 60*time.Minute)
	}
	return drama, episodes, err
}

func (m *Manager) GetStream(fullID, epIndex string) (*models.StreamData, error) {
	// Check Cache
	cacheKey := fmt.Sprintf("stream:%s:%s", fullID, epIndex)
	if x, found := m.cache.Get(cacheKey); found {
		return x.(*models.StreamData), nil
	}

	p, rawID, err := m.resolveProvider(fullID)
	if err != nil {
		return nil, err
	}
	data, err := p.GetStream(rawID, epIndex)
	if err == nil {
		// Cache successful stream for 30 mins
		m.cache.Set(cacheKey, data, 30*time.Minute)
	}
	return data, err
}
