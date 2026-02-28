package reporter

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/kullanici-adin/aegis-ai-asm/internal/discovery"
)

// Reporter writes discovery outputs to disk in different formats.
type Reporter struct {
	baseDir string
}

// New creates a Reporter, ensuring the base directory exists.
func New(baseDir string) (*Reporter, error) {
	if baseDir == "" {
		baseDir = "outputs"
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}
	return &Reporter{baseDir: baseDir}, nil
}

// WriteJSON saves assets as prettified JSON. Returns the file path.
func (r *Reporter) WriteJSON(assets []discovery.Asset) (string, error) {
	path := filepath.Join(r.baseDir, fmt.Sprintf("discovery-%s.json", timestamp()))
	content, err := json.MarshalIndent(assets, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// WriteCSV saves assets as CSV with common columns.
func (r *Reporter) WriteCSV(assets []discovery.Asset) (string, error) {
	path := filepath.Join(r.baseDir, fmt.Sprintf("discovery-%s.csv", timestamp()))
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"id", "domain", "kind", "owner", "risk_score"}
	if err := writer.Write(headers); err != nil {
		return "", err
	}

	for _, asset := range assets {
		row := []string{
			asset.ID,
			asset.Domain,
			asset.Kind,
			asset.Owner,
			strconv.FormatFloat(asset.RiskScore, 'f', 2, 64),
		}
		if err := writer.Write(row); err != nil {
			return "", err
		}
	}

	return path, nil
}

func timestamp() string {
	return time.Now().UTC().Format("20060102-150405")
}
