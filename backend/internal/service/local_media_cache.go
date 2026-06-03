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
	"strings"
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
