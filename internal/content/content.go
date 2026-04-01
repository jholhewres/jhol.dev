package content

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v3"
)

type Post struct {
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Date        string   `json:"date"`
	Tags        []string `json:"tags"`
	Summary     string   `json:"summary"`
	ReadingTime int      `json:"reading_time"`
	Content     string   `json:"content,omitempty"`
}

type Project struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	URL         string   `json:"url"`
	Featured    bool     `json:"featured"`
}

type Experience struct {
	Role        string   `json:"role"`
	Company     string   `json:"company"`
	Period      string   `json:"period"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type AboutContent struct {
	HTML string `json:"html"`
}

type Store struct {
	Posts      map[string][]Post       // lang -> sorted posts
	PostMap   map[string]map[string]Post // lang -> slug -> post
	Projects   map[string][]Project    // lang -> projects
	Experience map[string][]Experience // lang -> experience
	About      map[string]AboutContent // lang -> about HTML
}

var md goldmark.Markdown

func init() {
	md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
}

func Load(contentDir string) (*Store, error) {
	s := &Store{
		Posts:      make(map[string][]Post),
		PostMap:    make(map[string]map[string]Post),
		Projects:   make(map[string][]Project),
		Experience: make(map[string][]Experience),
		About:      make(map[string]AboutContent),
	}

	if err := s.loadPosts(contentDir); err != nil {
		return nil, fmt.Errorf("loading posts: %w", err)
	}
	if err := s.loadYAML(contentDir, "projects", &s.Projects); err != nil {
		return nil, fmt.Errorf("loading projects: %w", err)
	}
	if err := s.loadYAML(contentDir, "experience", &s.Experience); err != nil {
		return nil, fmt.Errorf("loading experience: %w", err)
	}
	if err := s.loadAbout(contentDir); err != nil {
		return nil, fmt.Errorf("loading about: %w", err)
	}

	return s, nil
}

func (s *Store) loadPosts(contentDir string) error {
	blogDir := filepath.Join(contentDir, "blog")
	entries, err := os.ReadDir(blogDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if this is a year directory (e.g. 2024/) or a post directory
		name := entry.Name()
		if isYearDir(name) {
			yearDir := filepath.Join(blogDir, name)
			yearEntries, err := os.ReadDir(yearDir)
			if err != nil {
				return err
			}
			for _, ye := range yearEntries {
				if !ye.IsDir() {
					continue
				}
				if err := s.loadPostDir(filepath.Join(yearDir, ye.Name()), ye.Name()); err != nil {
					return err
				}
			}
		} else {
			if err := s.loadPostDir(filepath.Join(blogDir, name), name); err != nil {
				return err
			}
		}
	}

	// Sort posts by date descending
	for lang := range s.Posts {
		sort.Slice(s.Posts[lang], func(i, j int) bool {
			return s.Posts[lang][i].Date > s.Posts[lang][j].Date
		})
	}

	return nil
}

func isYearDir(name string) bool {
	if len(name) != 4 {
		return false
	}
	for _, c := range name {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (s *Store) loadPostDir(postDir, dirName string) error {
	// Strip date prefix (YYYY-MM-DD-) from slug for URL
	urlSlug := dirName
	if len(dirName) > 11 && dirName[4] == '-' && dirName[7] == '-' && dirName[10] == '-' {
		urlSlug = dirName[11:]
	}

	files, err := os.ReadDir(postDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if !strings.HasPrefix(name, "index.") || !strings.HasSuffix(name, ".md") {
			continue
		}
		// Extract lang from index.{lang}.md
		lang := strings.TrimSuffix(strings.TrimPrefix(name, "index."), ".md")

		data, err := os.ReadFile(filepath.Join(postDir, name))
		if err != nil {
			return err
		}

		post, err := parsePost(data, urlSlug)
		if err != nil {
			return fmt.Errorf("parsing %s/%s: %w", dirName, name, err)
		}

		s.Posts[lang] = append(s.Posts[lang], post)

		if s.PostMap[lang] == nil {
			s.PostMap[lang] = make(map[string]Post)
		}
		s.PostMap[lang][urlSlug] = post
	}

	return nil
}

type frontmatter struct {
	Title       string   `yaml:"title"`
	Date        string   `yaml:"date"`
	Tags        []string `yaml:"tags"`
	Summary     string   `yaml:"summary"`
	ReadingTime int      `yaml:"reading_time"`
}

func parsePost(data []byte, slug string) (Post, error) {
	fm, body, err := parseFrontmatter(data)
	if err != nil {
		return Post{}, err
	}

	var buf bytes.Buffer
	if err := md.Convert(body, &buf); err != nil {
		return Post{}, fmt.Errorf("rendering markdown: %w", err)
	}

	readingTime := fm.ReadingTime
	if readingTime == 0 {
		words := len(strings.Fields(string(body)))
		readingTime = int(math.Max(1, math.Round(float64(words)/200.0)))
	}

	return Post{
		Slug:        slug,
		Title:       fm.Title,
		Date:        fm.Date,
		Tags:        fm.Tags,
		Summary:     fm.Summary,
		ReadingTime: readingTime,
		Content:     buf.String(),
	}, nil
}

func parseFrontmatter(data []byte) (frontmatter, []byte, error) {
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return frontmatter{}, data, nil
	}

	end := strings.Index(content[4:], "\n---")
	if end == -1 {
		return frontmatter{}, data, nil
	}

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(content[4:4+end]), &fm); err != nil {
		return frontmatter{}, nil, fmt.Errorf("parsing frontmatter YAML: %w", err)
	}

	body := []byte(content[4+end+4:])
	return fm, body, nil
}

func (s *Store) loadYAML(contentDir, name string, target interface{}) error {
	for _, lang := range []string{"en", "pt"} {
		path := filepath.Join(contentDir, fmt.Sprintf("%s.%s.yaml", name, lang))
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		switch t := target.(type) {
		case *map[string][]Project:
			var items []Project
			if err := yaml.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("parsing %s: %w", path, err)
			}
			(*t)[lang] = items
		case *map[string][]Experience:
			var items []Experience
			if err := yaml.Unmarshal(data, &items); err != nil {
				return fmt.Errorf("parsing %s: %w", path, err)
			}
			(*t)[lang] = items
		}
	}
	return nil
}

func (s *Store) loadAbout(contentDir string) error {
	for _, lang := range []string{"en", "pt"} {
		path := filepath.Join(contentDir, fmt.Sprintf("about.%s.md", lang))
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		var buf bytes.Buffer
		if err := md.Convert(data, &buf); err != nil {
			return fmt.Errorf("rendering about.%s.md: %w", lang, err)
		}
		s.About[lang] = AboutContent{HTML: buf.String()}
	}
	return nil
}
