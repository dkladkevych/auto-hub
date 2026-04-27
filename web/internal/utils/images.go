package utils

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
)

const (
	maxImageSize   = 5 * 1024 * 1024
	maxVideoSize   = 15 * 1024 * 1024
	maxFilesPerListing = 10
)

var allowedImageTypes = map[string]bool{
	"image/jpeg": true, "image/jpg": true, "image/png": true,
	"image/webp": true, "image/avif": true,
}
var allowedVideoTypes = map[string]bool{
	"video/mp4": true,
}

// ValidateImages checks uploaded files for size and type limits.
func ValidateImages(files []*multipart.FileHeader) error {
	if len(files) > maxFilesPerListing {
		return fmt.Errorf("maximum %d files per listing", maxFilesPerListing)
	}
	imageCount := 0
	videoCount := 0
	for _, f := range files {
		if f.Size == 0 {
			continue
		}
		ct := f.Header.Get("Content-Type")
		if allowedImageTypes[ct] {
			if f.Size > maxImageSize {
				return fmt.Errorf("image %s exceeds 5MB", f.Filename)
			}
			imageCount++
		} else if allowedVideoTypes[ct] {
			if f.Size > maxVideoSize {
				return fmt.Errorf("video %s exceeds 15MB", f.Filename)
			}
			videoCount++
		} else {
			return fmt.Errorf("unsupported file type: %s", f.Filename)
		}
	}
	if imageCount+videoCount > maxFilesPerListing {
		return fmt.Errorf("maximum %d files per listing", maxFilesPerListing)
	}
	return nil
}

// listingDir returns the filesystem path for a listing's media.
func listingDir(id int) string {
	return fmt.Sprintf("data/listings/%d", id)
}

// SyncListingImages atomically syncs existing and new media files.
func SyncListingImages(id int, keep []string, newFiles []*multipart.FileHeader) error {
	dir := listingDir(id)
	tmpDir := dir + ".tmp"
	if err := os.RemoveAll(tmpDir); err != nil {
		return err
	}
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return err
	}

	// Copy kept files
	copied := make(map[string]bool)
	for _, url := range keep {
		name := filepath.Base(url)
		if name == "" || name == "." || strings.Contains(name, "..") {
			continue
		}
		src := filepath.Join(dir, name)
		dst := filepath.Join(tmpDir, name)
		if err := copyFile(src, dst); err != nil {
			continue
		}
		copied[name] = true
	}

	// Save new files with sequential numbering
	idx := 1
	for _, f := range newFiles {
		if f.Size == 0 {
			continue
		}
		ext := extForMime(f.Header.Get("Content-Type"))
		name := fmt.Sprintf("%02d%s", idx, ext)
		dst := filepath.Join(tmpDir, name)
		if err := saveUploadedFile(f, dst); err != nil {
			return err
		}
		if isImage(f.Header.Get("Content-Type")) {
			thumbPath := filepath.Join(tmpDir, "thumb_"+name+".jpg")
			_ = createThumbnail(dst, thumbPath)
		}
		idx++
	}

	// Replace atomically
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmpDir, dir)
}

// DeleteListingImages removes a listing's media directory.
func DeleteListingImages(id int) error {
	return os.RemoveAll(listingDir(id))
}

func extForMime(ct string) string {
	switch ct {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/avif":
		return ".avif"
	case "video/mp4":
		return ".mp4"
	}
	return ""
}

func isImage(ct string) bool {
	return allowedImageTypes[ct]
}

func saveUploadedFile(fh *multipart.FileHeader, dst string) error {
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func createThumbnail(src, dst string) error {
	img, err := imaging.Open(src)
	if err != nil {
		return err
	}
	thumb := imaging.Fit(img, 640, 480, imaging.Lanczos)
	return imaging.Save(thumb, dst, imaging.JPEGQuality(85))
}

// GetPreviewImage returns the first image URL for a listing.
func GetPreviewImage(id int, thumb bool) string {
	dir := listingDir(id)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "/static/images/empty.png"
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "thumb_") {
			if thumb {
				files = append(files, name)
			}
			continue
		}
		if thumb {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return "/static/images/empty.png"
	}
	return fmt.Sprintf("/data/listings/%d/%s", id, files[0])
}

// ListingHasVideo checks if a listing directory contains an mp4.
func ListingHasVideo(id int) bool {
	dir := listingDir(id)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".mp4") {
			return true
		}
	}
	return false
}

// GetMediaURLs returns image/video URLs for a listing.
func GetMediaURLs(id int, thumb bool) []string {
	dir := listingDir(id)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "thumb_") {
			if thumb {
				files = append(files, fmt.Sprintf("/data/listings/%d/%s", id, name))
			}
			continue
		}
		if thumb {
			continue
		}
		files = append(files, fmt.Sprintf("/data/listings/%d/%s", id, name))
	}
	sort.Strings(files)
	return files
}

// ToInt safely parses an integer from string.
func ToInt(s string) *int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}
