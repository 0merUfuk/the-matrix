// Package registry provides a client for the knowledge registry — a central
// Git-based index of pre-researched oracle knowledge packs.
package registry

// RegistryIndex is the top-level structure of registry.json.
type RegistryIndex struct {
	Version     string     `json:"version"`
	LastUpdated string     `json:"lastUpdated"`
	Packs       []PackMeta `json:"packs"`
}

// PackMeta holds summary metadata for a knowledge pack as listed in the registry index.
type PackMeta struct {
	Name          string `json:"name"`
	DisplayName   string `json:"displayName"`
	Language      string `json:"language"`
	Framework     string `json:"framework"`
	Version       string `json:"version"`
	CreatedDate   string `json:"createdDate"`
	RefreshedDate string `json:"refreshedDate"`
	TopicCount    int    `json:"topicCount"`
	QualityScore  int    `json:"qualityScore"`
	DownloadURL   string `json:"downloadUrl"`
	Description   string `json:"description"`
}

// PackDetail extends PackMeta with full pack metadata including topic list.
type PackDetail struct {
	PackMeta
	ArchStyle    string      `json:"archStyle"`
	ResearchedBy string      `json:"researchedBy"`
	GoldStandard string      `json:"goldStandard"`
	Topics       []PackTopic `json:"topics"`
}

// PackTopic describes a single knowledge doc file within a pack.
type PackTopic struct {
	File         string `json:"file"`
	Lines        int    `json:"lines"`
	Tier1Sources int    `json:"tier1Sources"`
}
