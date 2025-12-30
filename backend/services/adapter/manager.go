package adapter

import (
	"dramabang/models"
	"fmt"
	"strings"
	"sync"
)

type Manager struct {
	providers map[string]Provider
	// Slice for ordered iteration (e.g. priority)
	providerList []Provider
}

func NewManager() *Manager {
	db := NewDramaboxProvider()
	ml := NewMeloloProvider()
	ns := NewNetshortProvider()

	return &Manager{
		providers: map[string]Provider{
			db.GetID(): db,
			ml.GetID(): ml,
			ns.GetID(): ns,
		},
		providerList: []Provider{db, ml, ns},
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
	// Loop until all lists are exhausted
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

	return merged, nil
}

func (m *Manager) Search(query string) ([]models.Drama, error) {
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

	return merged, nil
}

func (m *Manager) GetLatest(page int) ([]models.Drama, error) {
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

	return merged, nil
}

func (m *Manager) GetDetail(fullID string) (*models.Drama, []models.Episode, error) {
	p, rawID, err := m.resolveProvider(fullID)
	if err != nil {
		return nil, nil, err
	}
	return p.GetDetail(rawID)
}

func (m *Manager) GetStream(fullID, epIndex string) (*models.StreamData, error) {
	p, rawID, err := m.resolveProvider(fullID)
	if err != nil {
		return nil, err
	}
	return p.GetStream(rawID, epIndex)
}
