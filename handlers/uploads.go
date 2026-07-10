package handlers

import (
	"RTF/database"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type imageInputError struct{ message string }

func (e *imageInputError) Error() string { return e.message }

var imageValidationSlots = make(chan struct{}, 2)
var errUploadBusy = errors.New("Image processing is busy. Try again shortly.")

func removeUpload(name string) {
	if err := os.Remove(filepath.Join(uploadDir(), name)); err != nil && !os.IsNotExist(err) {
		log.Printf("upload cleanup: %v", err)
	}
}

const maxImageSize = 20 << 20 // 20 MiB, independently of multipart overhead.
const maxUploadRequest = maxImageSize + (128 << 10)

func uploadDir() string {
	if path := os.Getenv("UPLOAD_PATH"); path != "" {
		return path
	}
	path := os.Getenv("DATABASE_PATH")
	if path == "" {
		path = "Real-Time-Forum.db"
	}
	return filepath.Join(filepath.Dir(path), strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))+"-uploads")
}

// saveImage is called only after the post's text and topics have been validated.
func saveImage(header *multipart.FileHeader) (name string, err error) {
	if header == nil {
		return "", nil
	}
	select {
	case imageValidationSlots <- struct{}{}:
		defer func() { <-imageValidationSlots }()
	default:
		return "", errUploadBusy
	}
	if header.Size > maxImageSize {
		return "", &imageInputError{"Image exceeds the 20 MiB limit"}
	}
	source, err := header.Open()
	if err != nil {
		return "", err
	}
	defer func() {
		if e := source.Close(); err == nil && e != nil {
			err = e
			if name != "" {
				removeUpload(name)
				name = ""
			}
		}
	}()
	config, format, err := image.DecodeConfig(source)
	if err != nil || (format != "jpeg" && format != "png" && format != "gif") || config.Width < 1 || config.Height < 1 || int64(config.Width)*int64(config.Height) > 40000000 {
		return "", &imageInputError{"Use a valid JPEG, PNG, or GIF image up to 40 million pixels"}
	}
	if _, err = source.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	// Decode the primary raster as well, rejecting truncated or malformed images.
	if _, _, err = image.Decode(source); err != nil {
		return "", &imageInputError{"Malformed image"}
	}
	if _, err = source.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	if err = os.MkdirAll(uploadDir(), 0700); err != nil {
		return "", err
	}
	storageID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	name = storageID.String() + "." + format
	destination, err := os.OpenFile(filepath.Join(uploadDir(), name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	n, copyErr := io.Copy(destination, io.LimitReader(source, maxImageSize+1))
	closeErr := destination.Close()
	if copyErr != nil || closeErr != nil || n > maxImageSize {
		removeUpload(name)
		if n > maxImageSize {
			return "", &imageInputError{"Image exceeds the 20 MiB limit"}
		}
		return "", fmt.Errorf("write image: %w", errors.Join(copyErr, closeErr))
	}
	return name, nil
}

// Only UUID files owned by this application are eligible for cleanup.
func ownedImage(name string) bool {
	extension := filepath.Ext(name)
	_, err := uuid.Parse(strings.TrimSuffix(name, extension))
	return err == nil && filepath.Base(name) == name && (extension == ".jpeg" || extension == ".png" || extension == ".gif")
}
func cleanupImage(name string) {
	if !ownedImage(name) {
		return
	}
	var count int
	if err := database.DBInstance.DB.QueryRow("SELECT COUNT(*) FROM posts WHERE image_path=?", name).Scan(&count); err != nil {
		log.Printf("upload reference check: %v", err)
		return
	}
	if count == 0 {
		removeUpload(name)
	}
}

// ReconcileUploads retries failed deletions and cleans files left by a process
// interruption. The one-hour grace period protects in-flight submissions.
func ReconcileUploads() {
	files, err := os.ReadDir(uploadDir())
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		log.Printf("upload reconciliation: %v", err)
		return
	}
	for _, file := range files {
		if !ownedImage(file.Name()) || file.Type()&os.ModeSymlink != 0 {
			continue
		}
		info, err := file.Info()
		if err != nil {
			log.Printf("upload reconciliation: %v", err)
			continue
		}
		if info.Mode().IsRegular() && time.Since(info.ModTime()) > time.Hour {
			cleanupImage(file.Name())
		}
	}
}

func ServeImage(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/uploads/")
	if !ownedImage(name) {
		http.NotFound(w, r)
		return
	}
	var count int
	if err := database.DBInstance.DB.QueryRow("SELECT COUNT(*) FROM posts p WHERE image_path=? AND "+visibility("p", r), name).Scan(&count); err != nil {
		serverError(w, err)
		return
	}
	if count == 0 {
		http.NotFound(w, r)
		return
	}
	mime := map[string]string{".jpeg": "image/jpeg", ".png": "image/png", ".gif": "image/gif"}[filepath.Ext(name)]
	w.Header().Set("Content-Type", mime)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, filepath.Join(uploadDir(), name))
}
