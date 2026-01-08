package adapter

import (
	"dramabang/models"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type MovieProvider struct {
	scriptPath string
}

func NewMovieProvider() *MovieProvider {
	return &MovieProvider{
		scriptPath: "python/scraper.py",
	}
}

func (p *MovieProvider) GetID() string {
	return "movie"
}

func (p *MovieProvider) IsCompatibleID(id string) bool {
	return strings.HasPrefix(id, "movie:")
}

// --- Python Exec Helper ---
func (p *MovieProvider) runPython(args ...string) ([]byte, error) {
	// args[0] = cmd (latest, detail, stream)
	// args[1...] = flags
	fullArgs := append([]string{p.scriptPath}, args...)
	cmd := exec.Command("python3", fullArgs...)

	// Capture Combined Output (stdout + stderr)?
	// No, we designed scraper to print JSON to stdout and logs to stderr.
	// So we only want Output().
	output, err := cmd.Output()
	if err != nil {
		// Exit status mismatch
		if exitError, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("python script failed: %s, stderr: %s", err, string(exitError.Stderr))
		}
		return nil, err
	}
	return output, nil
}

// --- Implementation ---

func (p *MovieProvider) GetTrending() ([]models.Drama, error) {
	// Alias to Latest page 1
	return p.GetLatest(1)
}

func (p *MovieProvider) Search(query string) ([]models.Drama, error) {
	// NOT SUPPORTED in Python Script yet?
	// The provided scraper.py didn't have search.
	// We can return empty for now or implement search in python later.
	// User only asked for "New Movies".
	return []models.Drama{}, nil
}

func (p *MovieProvider) GetLatest(page int) ([]models.Drama, error) {
	out, err := p.runPython("latest", "--page", fmt.Sprintf("%d", page))
	if err != nil {
		return nil, err
	}

	var rawMovies []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Cover string `json:"cover"`
		Genre string `json:"genre"`
	}

	if err := json.Unmarshal(out, &rawMovies); err != nil {
		return nil, fmt.Errorf("failed to parse python output: %v", err)
	}

	var movies []models.Drama
	for _, rm := range rawMovies {
		movies = append(movies, models.Drama{
			BookID: rm.ID,
			Judul:  rm.Title,
			Cover:  rm.Cover,
			Genre:  rm.Genre,
		})
	}
	return movies, nil
}

func (p *MovieProvider) GetDetail(id string) (*models.Drama, []models.Episode, error) {
	// ID is "movie:slug", python expects "slug"
	rawID := strings.TrimPrefix(id, "movie:")
	out, err := p.runPython("detail", "--url", rawID)
	if err != nil {
		return nil, nil, err
	}

	var res struct {
		Drama struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			Cover       string `json:"cover"`
			Genre       string `json:"genre"`
			TotalEps    string `json:"total_episodes"`
		} `json:"drama"`
		Episodes []struct {
			ID    string `json:"id"`
			Index int    `json:"index"`
			Label string `json:"label"`
		} `json:"episodes"`
	}

	if err := json.Unmarshal(out, &res); err != nil {
		return nil, nil, fmt.Errorf("failed to parse detail: %v", err)
	}

	drama := &models.Drama{
		BookID:       res.Drama.ID,
		Judul:        res.Drama.Title,
		Deskripsi:    res.Drama.Description,
		Cover:        res.Drama.Cover,
		Genre:        res.Drama.Genre,
		TotalEpisode: res.Drama.TotalEps,
	}

	var episodes []models.Episode
	for _, ep := range res.Episodes {
		episodes = append(episodes, models.Episode{
			BookID:       ep.ID,
			EpisodeIndex: ep.Index,
			EpisodeLabel: ep.Label,
		})
	}

	return drama, episodes, nil
}

func (p *MovieProvider) GetStream(id, epIndex string) (*models.StreamData, error) {
	rawID := strings.TrimPrefix(id, "movie:")
	out, err := p.runPython("stream", "--url", rawID)
	if err != nil {
		return nil, err
	}

	var stream models.StreamData
	if err := json.Unmarshal(out, &stream); err != nil {
		return nil, fmt.Errorf("failed to parse stream: %v", err)
	}

	return &stream, nil
}
