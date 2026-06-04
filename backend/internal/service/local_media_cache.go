package service

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	localMediaImage = "image"
	localMediaVideo = "video"
)

var localMediaIDRE = regexp.MustCompile(`^[0-9a-fA-F-]{16,64}$`)

type LocalMediaFile struct {
	Path        string
	ContentType string
}

type LocalMediaCacheItem struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	FileName    string `json:"file_name"`
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	ModifiedAt  int64  `json:"modified_at"`
	URL         string `json:"url"`
}

type LocalMediaCacheListFilter struct {
	Type          string
	Search        string
	Before        time.Time
	Page          int
	PageSize      int
	IncludeImages bool
	IncludeVideos bool
}

type LocalMediaCacheListResult struct {
	Items    []LocalMediaCacheItem `json:"items"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

type LocalMediaCacheCleanupInput struct {
	Type   string
	Before time.Time
	Limit  int
}

type LocalMediaCacheCleanupResult struct {
	Deleted int      `json:"deleted"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

func saveLocalImage(raw []byte, contentType, seed string) (string, error) {
	if len(raw) == 0 {
		return "", errors.New("image content is empty")
	}
	id := localMediaID(seed)
	ext := imageExtFromContentType(contentType)
	path := filepath.Join(localMediaDir(localMediaImage), id+ext)
	if err := writeLocalMediaFile(path, raw); err != nil {
		return "", err
	}
	return id, nil
}

func saveLocalVideo(raw []byte, seed string) (string, error) {
	if len(raw) == 0 {
		return "", errors.New("video content is empty")
	}
	id := localMediaID(seed)
	path := filepath.Join(localMediaDir(localMediaVideo), id+".mp4")
	if err := writeLocalMediaFile(path, raw); err != nil {
		return "", err
	}
	return id, nil
}

func LocalImageFile(id string) (*LocalMediaFile, error) {
	id = strings.TrimSpace(id)
	if !localMediaIDRE.MatchString(id) {
		return nil, errors.New("invalid image file id")
	}
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp"} {
		path := filepath.Join(localMediaDir(localMediaImage), id+ext)
		if fileExists(path) {
			contentType := mime.TypeByExtension(ext)
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			return &LocalMediaFile{Path: path, ContentType: contentType}, nil
		}
	}
	return nil, os.ErrNotExist
}

func LocalVideoFile(id string) (*LocalMediaFile, error) {
	id = strings.TrimSpace(id)
	if !localMediaIDRE.MatchString(id) {
		return nil, errors.New("invalid video file id")
	}
	path := filepath.Join(localMediaDir(localMediaVideo), id+".mp4")
	if !fileExists(path) {
		return nil, os.ErrNotExist
	}
	return &LocalMediaFile{Path: path, ContentType: "video/mp4"}, nil
}

func localImageURL(id string) string {
	return "/v1/files/image?id=" + id
}

func localVideoURL(id string) string {
	return "/v1/files/video?id=" + id
}

func localMediaID(seed string) string {
	seed = strings.TrimSpace(seed)
	if localMediaIDRE.MatchString(seed) {
		return strings.ToLower(seed)
	}
	sum := sha1.Sum([]byte(seed))
	return hex.EncodeToString(sum[:])
}

func localMediaDir(mediaType string) string {
	dataDir := strings.TrimSpace(os.Getenv("DATA_DIR"))
	if dataDir == "" {
		dataDir = filepath.Join(".", "data")
	}
	switch mediaType {
	case localMediaImage:
		return filepath.Join(dataDir, "files", "images")
	case localMediaVideo:
		return filepath.Join(dataDir, "files", "videos")
	default:
		return filepath.Join(dataDir, "files")
	}
}

func writeLocalMediaFile(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if fileExists(path) {
		return nil
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func imageExtFromContentType(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if ext, err := mime.ExtensionsByType(contentType); err == nil && len(ext) > 0 {
		switch ext[0] {
		case ".jpe":
			return ".jpg"
		default:
			return ext[0]
		}
	}
	switch contentType {
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/bmp":
		return ".bmp"
	default:
		return ".jpg"
	}
}

func responseContentType(resp *http.Response, fallback string) string {
	if resp != nil {
		if ct := strings.TrimSpace(resp.Header.Get("Content-Type")); ct != "" {
			return ct
		}
	}
	if fallback != "" {
		return fallback
	}
	return "application/octet-stream"
}

func localMediaNotFound(mediaType, id string) error {
	return fmt.Errorf("%s file %q not found", mediaType, id)
}

func (s *OpenAIGatewayService) GetLocalImageFile(id string) (*LocalMediaFile, error) {
	file, err := LocalImageFile(id)
	if errors.Is(err, os.ErrNotExist) {
		return nil, localMediaNotFound(localMediaImage, id)
	}
	return file, err
}

func (s *OpenAIGatewayService) GetLocalVideoFile(id string) (*LocalMediaFile, error) {
	file, err := LocalVideoFile(id)
	if errors.Is(err, os.ErrNotExist) {
		return nil, localMediaNotFound(localMediaVideo, id)
	}
	return file, err
}

func (s *OpenAIGatewayService) ListLocalMediaCache(filter LocalMediaCacheListFilter) (*LocalMediaCacheListResult, error) {
	items, err := listLocalMediaCacheItems(filter)
	if err != nil {
		return nil, err
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return &LocalMediaCacheListResult{
		Items:    items[start:end],
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *OpenAIGatewayService) DeleteLocalMediaCacheItem(mediaType, id string) error {
	path, err := localMediaCachePath(mediaType, id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return localMediaNotFound(mediaType, id)
		}
		return err
	}
	return nil
}

func (s *OpenAIGatewayService) CleanupLocalMediaCache(input LocalMediaCacheCleanupInput) (*LocalMediaCacheCleanupResult, error) {
	items, err := listLocalMediaCacheItems(LocalMediaCacheListFilter{
		Type:   input.Type,
		Before: input.Before,
	})
	if err != nil {
		return nil, err
	}
	if input.Limit > 0 && len(items) > input.Limit {
		items = items[:input.Limit]
	}
	result := &LocalMediaCacheCleanupResult{}
	for _, item := range items {
		if err := os.Remove(item.Path); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, err.Error())
			continue
		}
		result.Deleted++
	}
	return result, nil
}

func (s *OpenAIGatewayService) CleanupLocalMediaOrphans() (*LocalMediaCacheCleanupResult, error) {
	result := &LocalMediaCacheCleanupResult{}
	for _, mediaType := range []string{localMediaImage, localMediaVideo} {
		dir := localMediaDir(mediaType)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			name := entry.Name()
			id := strings.TrimSuffix(name, filepath.Ext(name))
			orphan := strings.HasSuffix(name, ".tmp") || !localMediaIDRE.MatchString(id)
			if !orphan {
				result.Skipped++
				continue
			}
			if err := os.Remove(path); err != nil {
				result.Skipped++
				result.Errors = append(result.Errors, err.Error())
				continue
			}
			result.Deleted++
		}
	}
	return result, nil
}

func listLocalMediaCacheItems(filter LocalMediaCacheListFilter) ([]LocalMediaCacheItem, error) {
	mediaTypes, err := localMediaCacheTypes(filter)
	if err != nil {
		return nil, err
	}
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	items := []LocalMediaCacheItem{}
	for _, mediaType := range mediaTypes {
		dir := localMediaDir(mediaType)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() || strings.HasSuffix(entry.Name(), ".tmp") {
				continue
			}
			id := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			if !localMediaIDRE.MatchString(id) {
				continue
			}
			if search != "" && !strings.Contains(strings.ToLower(entry.Name()), search) && !strings.Contains(strings.ToLower(id), search) {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				return nil, err
			}
			if !filter.Before.IsZero() && !info.ModTime().Before(filter.Before) {
				continue
			}
			ext := filepath.Ext(entry.Name())
			contentType := mime.TypeByExtension(ext)
			if contentType == "" {
				if mediaType == localMediaVideo {
					contentType = "video/mp4"
				} else {
					contentType = "application/octet-stream"
				}
			}
			item := LocalMediaCacheItem{
				ID:          strings.ToLower(id),
				Type:        mediaType,
				FileName:    entry.Name(),
				Path:        filepath.Join(dir, entry.Name()),
				ContentType: contentType,
				Size:        info.Size(),
				ModifiedAt:  info.ModTime().Unix(),
			}
			if mediaType == localMediaVideo {
				item.URL = localVideoURL(item.ID)
			} else {
				item.URL = localImageURL(item.ID)
			}
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ModifiedAt == items[j].ModifiedAt {
			return items[i].FileName < items[j].FileName
		}
		return items[i].ModifiedAt > items[j].ModifiedAt
	})
	return items, nil
}

func localMediaCacheTypes(filter LocalMediaCacheListFilter) ([]string, error) {
	mediaType := strings.ToLower(strings.TrimSpace(filter.Type))
	if filter.IncludeImages || filter.IncludeVideos {
		out := []string{}
		if filter.IncludeImages {
			out = append(out, localMediaImage)
		}
		if filter.IncludeVideos {
			out = append(out, localMediaVideo)
		}
		return out, nil
	}
	switch mediaType {
	case "", "all":
		return []string{localMediaImage, localMediaVideo}, nil
	case localMediaImage, localMediaVideo:
		return []string{mediaType}, nil
	default:
		return nil, fmt.Errorf("media type must be one of [%s, %s, all]", localMediaImage, localMediaVideo)
	}
}

func localMediaCachePath(mediaType, id string) (string, error) {
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	id = strings.ToLower(strings.TrimSpace(id))
	if !localMediaIDRE.MatchString(id) {
		return "", errors.New("invalid media file id")
	}
	switch mediaType {
	case localMediaImage:
		file, err := LocalImageFile(id)
		if err != nil {
			return "", err
		}
		return file.Path, nil
	case localMediaVideo:
		file, err := LocalVideoFile(id)
		if err != nil {
			return "", err
		}
		return file.Path, nil
	default:
		return "", fmt.Errorf("media type must be one of [%s, %s]", localMediaImage, localMediaVideo)
	}
}
