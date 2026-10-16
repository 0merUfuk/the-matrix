package matrixapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/0merUfuk/the-matrix/internal/cli"
)

// projectStore keeps the user's project list in ~/.the-matrix/projects.json.
type projectStore struct {
	mu   sync.Mutex
	file string
	home string
}

func newProjectStore(home string) *projectStore {
	return &projectStore{file: filepath.Join(home, ".the-matrix", "projects.json"), home: home}
}

func (s *projectStore) list() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read()
}

func (s *projectStore) read() []string {
	var data struct {
		Projects []string `json:"projects"`
	}
	if raw, err := os.ReadFile(s.file); err == nil {
		_ = json.Unmarshal(raw, &data)
	}
	return data.Projects
}

func (s *projectStore) write(list []string) error {
	if err := os.MkdirAll(filepath.Dir(s.file), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(map[string][]string{"projects": list}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.file + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.file)
}

func (s *projectStore) add(dir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.read()
	for _, p := range list {
		if p == dir {
			return nil
		}
	}
	return s.write(append(list, dir))
}

func (s *projectStore) remove(dir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.read()
	out := list[:0]
	for _, p := range list {
		if p != dir {
			out = append(out, p)
		}
	}
	return s.write(out)
}

// resolve turns a path from the browser into a clean absolute folder under
// the home directory. "~/" is expanded; anything else outside home is refused.
func resolve(home, raw string) (string, error) {
	if raw == "" {
		return "", errors.New("choose a folder")
	}
	if raw == "~" {
		raw = home
	} else if strings.HasPrefix(raw, "~/") {
		raw = filepath.Join(home, raw[2:])
	}
	if !filepath.IsAbs(raw) {
		return "", errors.New("use a full folder path")
	}
	p := filepath.Clean(raw)
	if home == "" || (p != home && !strings.HasPrefix(p, home+string(os.PathSeparator))) {
		return "", errors.New("projects must be inside your home folder")
	}
	if info, err := os.Stat(p); err != nil || !info.IsDir() {
		return "", fmt.Errorf("%s is not a folder", cli.PathToHome(p))
	}
	return p, nil
}

type folder struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Project bool   `json:"project"`
}

type folderList struct {
	Path    string   `json:"path"`
	Parent  string   `json:"parent"`
	Project bool     `json:"project"`
	Folders []folder `json:"folders"`
}

// browse lists the visible subfolders of dir for the folder picker.
func browse(home, dir string) (folderList, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return folderList{}, fmt.Errorf("cannot open %s", cli.PathToHome(dir))
	}
	out := folderList{Path: cli.PathToHome(dir), Project: isSetUp(dir), Folders: []folder{}}
	if dir != home {
		out.Parent = cli.PathToHome(filepath.Dir(dir))
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		full := filepath.Join(dir, e.Name())
		out.Folders = append(out.Folders, folder{Name: e.Name(), Path: cli.PathToHome(full), Project: isSetUp(full)})
		if len(out.Folders) == 500 {
			break
		}
	}
	sort.Slice(out.Folders, func(i, j int) bool {
		return strings.ToLower(out.Folders[i].Name) < strings.ToLower(out.Folders[j].Name)
	})
	return out, nil
}
