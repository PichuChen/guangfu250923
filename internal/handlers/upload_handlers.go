package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"log/slog"

	"guangfu250923/internal/localcache"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"image"
	"image/jpeg"
	"image/png"
)

// UploadPhoto accepts multipart/form-data with a file field named "file" and uploads to S3.
// Returns 201 with JSON: { url, key, content_type, size } when successful.
func (h *Handler) UploadPhoto(c *gin.Context) {
	slog.Info("UploadPhoto: start", "content_type", c.GetHeader("Content-Type"))
	if h.s3 == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "upload unavailable"})
		return
	}

	// Limit reader size according to configured max
	// Gin's MaxMultipartMemory can be tuned, but here we rely on streaming + S3 uploader limiter
	// Accept only multipart/form-data
	ctReq := c.ContentType()
	if !strings.HasPrefix(ctReq, "multipart/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content type must be multipart/form-data"})
		return
	}

	// Ensure form parsing occurs and capture any error for debugging
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		slog.Error("UploadPhoto: ParseMultipartForm error", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		// surface exact error for debugging
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Enforce maximum size if known
	if h.s3.MaxBytes() > 0 && fileHeader.Size > 0 && fileHeader.Size > h.s3.MaxBytes() {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file too large"})
		return
	}

	// Open the file for streaming upload
	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer f.Close()

	// Basic validation: filename and size
	filename := sanitizeFilename(fileHeader.Filename)
	if filename == "" {
		filename = fmt.Sprintf("upload-%d", time.Now().UnixNano())
	}
	// Detect content type by sniffing first 512 bytes when possible
	var sniffBuf [512]byte
	n, _ := io.ReadFull(f, sniffBuf[:])
	// Prepare reader for upload by concatenating sniffed bytes back
	var uploadReader io.Reader = io.MultiReader(bytes.NewReader(sniffBuf[:n]), f)

	ctype := http.DetectContentType(sniffBuf[:n])
	// Fallback to header or extension if DetectContentType returned generic type
	if ctype == "application/octet-stream" || ctype == "binary/octet-stream" || ctype == "text/plain; charset=utf-8" {
		if h := fileHeader.Header.Get("Content-Type"); h != "" {
			ctype = h
		} else {
			ext := strings.ToLower(filepath.Ext(filename))
			switch ext {
			case ".jpg", ".jpeg":
				ctype = "image/jpeg"
			case ".png":
				ctype = "image/png"
			case ".webp":
				ctype = "image/webp"
			case ".heic":
				ctype = "image/heic"
			default:
				ctype = "application/octet-stream"
			}
		}
	}

	// Only allow images
	if !strings.HasPrefix(strings.ToLower(ctype), "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only image uploads are allowed"})
		return
	}

	// Generate a uuidv7 for public-facing id and object key path
	newID, err := uuid.NewV7()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate id"})
		return
	}
	// Object key does not expose original filename to the URL path to reduce risk, but we keep extension
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".bin"
	}
	key := "photos/" + newID.String() + ext

	// Use a context with timeout for the upload
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	url, objectKey, err := h.s3.Upload(ctx, key, uploadReader, ctype)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	size := fileHeader.Size
	// size left as provided by multipart header; streaming prevents accurate recount here

	// Persist metadata
	if _, err := h.pool.Exec(c.Request.Context(),
		`insert into photos(id, object_key, original_filename, content_type, size, public_url) values($1,$2,$3,$4,$5,$6)`,
		newID.String(), objectKey, filename, ctype, size, url,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return the user-facing path and metadata; clients will GET /photos/{id} to fetch/redirect
	c.JSON(http.StatusCreated, gin.H{
		"id":           newID.String(),
		"path":         "/photos/" + newID.String(),
		"content_type": ctype,
		"size":         size,
	})
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "..", "-")
	// simple normalization
	return name
}

// GetPhoto resolves a public photo ID to its actual public URL.
// Current behavior: 302 redirect to stored public_url.
func (h *Handler) GetPhoto(c *gin.Context) {
	id := c.Param("id")
	var url string
	var objectKey string
	var contentType string
	if err := h.pool.QueryRow(c.Request.Context(), `select public_url, object_key, content_type from photos where id=$1`, id).Scan(&url, &objectKey, &contentType); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	// Thumbnail selector via query param: small(w100), medium(w300, default), large(w1200), original
	thumbSel := strings.TrimSpace(strings.ToLower(c.Query("thumbnail")))
	var targetWidth int
	switch thumbSel {
	case "", "medium":
		targetWidth = 300
	case "small":
		targetWidth = 100
	case "large":
		targetWidth = 1200
	case "original":
		targetWidth = 0
	default:
		// 未知值時以預設 medium
		targetWidth = 300
	}

	if targetWidth > 0 {
		// Serve/generate thumbnail
		spec := fmt.Sprintf("w%d", targetWidth)
		thumbPath := localcache.ThumbPath(objectKey, spec)
		if localcache.Exists(thumbPath) {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
			c.File(thumbPath)
			return
		}
		// Need source image
		srcPath := localcache.PhotoPath(objectKey)
		var src io.ReadCloser
		if localcache.Exists(srcPath) {
			if f, err := os.Open(srcPath); err == nil { src = f }
		}
		if src == nil && h.s3 != nil {
			if rc, _, _, err := h.s3.GetObject(c.Request.Context(), objectKey); err == nil { src = rc }
		}
		if src == nil {
			// Fallback: presign to original if source unavailable
			if h.s3 != nil {
				if presigned, perr := h.s3.PresignGet(c.Request.Context(), objectKey, 5*time.Minute); perr == nil {
					c.Header("Cache-Control", "private, max-age=60")
					c.Redirect(http.StatusFound, presigned)
					return
				}
			}
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "source unavailable"})
			return
		}
		defer src.Close()

		data, err := io.ReadAll(io.LimitReader(src, 32<<20))
		if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"read failed"}); return }
		img, format, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			// Unsupported by stdlib: fallback to presigned
			if h.s3 != nil {
				if presigned, perr := h.s3.PresignGet(c.Request.Context(), objectKey, 5*time.Minute); perr == nil {
					c.Header("Cache-Control", "private, max-age=60")
					c.Redirect(http.StatusFound, presigned)
					return
				}
			}
			c.JSON(http.StatusBadRequest, gin.H{"error":"decode failed"})
			return
		}
		b := img.Bounds()
		if b.Dx() <= targetWidth {
			// No upscale; cache original bytes into thumb path for consistency
			if err := localcache.Save(thumbPath, bytes.NewReader(data)); err == nil {
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
				c.File(thumbPath)
				return
			}
			// If saving failed, just return original bytes
			ct := contentType
			if ct == "" { ct = http.DetectContentType(data) }
			c.Data(http.StatusOK, ct, data)
			return
		}
		height := int(float64(b.Dy()) * (float64(targetWidth) / float64(b.Dx())))
		if height <= 0 { height = 1 }
		dst := image.NewRGBA(image.Rect(0,0,targetWidth,height))
		for y := 0; y < height; y++ {
			sy := y * b.Dy() / height
			for x := 0; x < targetWidth; x++ {
				sx := x * b.Dx() / targetWidth
				dst.Set(x, y, img.At(b.Min.X+sx, b.Min.Y+sy))
			}
		}
		buf := new(bytes.Buffer)
		ct := "image/jpeg"
		if format == "png" {
			if err := png.Encode(buf, dst); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"encode failed"}); return }
			ct = "image/png"
		} else {
			if err := jpeg.Encode(buf, dst, &jpeg.Options{Quality: 75}); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"encode failed"}); return }
		}
		if err := localcache.Save(thumbPath, bytes.NewReader(buf.Bytes())); err == nil {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
			c.Data(http.StatusOK, ct, buf.Bytes())
			return
		}
		c.Header("Cache-Control", "private, max-age=60")
		c.Data(http.StatusOK, ct, buf.Bytes())
		return
	}

	// Original path (no thumbnail)
	// Determine local cache path
	cachePath := localcache.PhotoPath(objectKey)
	if localcache.Exists(cachePath) {
		if contentType != "" { c.Header("Content-Type", contentType) }
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.File(cachePath)
		return
	}
	if h.s3 != nil {
		if rc, s3CT, _, err := h.s3.GetObject(c.Request.Context(), objectKey); err == nil {
			defer rc.Close()
			if werr := localcache.Save(cachePath, rc); werr == nil {
				if contentType == "" { contentType = s3CT }
				if contentType != "" { c.Header("Content-Type", contentType) }
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
				c.File(cachePath)
				return
			}
			// If saving failed, re-fetch and stream without cache
			if rc2, s3CT2, _, err2 := h.s3.GetObject(c.Request.Context(), objectKey); err2 == nil {
				defer rc2.Close()
				if contentType == "" { contentType = s3CT2 }
				if contentType != "" { c.Header("Content-Type", contentType) }
				c.Header("Cache-Control", "private, max-age=60")
				if _, copyErr := io.Copy(c.Writer, rc2); copyErr == nil { c.Status(http.StatusOK); return }
			}
		}
		if presigned, perr := h.s3.PresignGet(c.Request.Context(), objectKey, 5*time.Minute); perr == nil {
			c.Header("Cache-Control", "private, max-age=60")
			c.Redirect(http.StatusFound, presigned)
			return
		}
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Redirect(http.StatusFound, url)
}

// GetPhotoThumbnail generates/serves a cached thumbnail for a photo.
// Route example: GET /photos/:id/thumb/:w where :w is like "w480" (width in px).
func (h *Handler) GetPhotoThumbnail(c *gin.Context) {
	id := c.Param("id")
	spec := c.Param("w")
	if spec == "" || !strings.HasPrefix(spec, "w") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid thumbnail spec"})
		return
	}
	// Parse width
	widthStr := strings.TrimPrefix(spec, "w")
	if widthStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing width"})
		return
	}
	var width int
	for _, ch := range widthStr {
		if ch < '0' || ch > '9' { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid width"}); return }
	}
	// Simple Atoi without importing strconv
	for i := 0; i < len(widthStr); i++ { width = width*10 + int(widthStr[i]-'0') }
	if width <= 0 || width > 4096 { c.JSON(http.StatusBadRequest, gin.H{"error":"width out of range"}); return }

	var objectKey, contentType string
	if err := h.pool.QueryRow(c.Request.Context(), `select object_key, content_type from photos where id=$1`, id).Scan(&objectKey, &contentType); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	thumbPath := localcache.ThumbPath(objectKey, spec)
	if localcache.Exists(thumbPath) {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.File(thumbPath)
		return
	}

	// Need source image: prefer local original cache first
	srcPath := localcache.PhotoPath(objectKey)
	var src io.ReadCloser
	if localcache.Exists(srcPath) {
		f, err := os.Open(srcPath)
		if err == nil { src = f }
	}
	// Else fetch from S3
	if src == nil && h.s3 != nil {
		rc, _, _, err := h.s3.GetObject(c.Request.Context(), objectKey)
		if err == nil { src = rc }
	}
	if src == nil {
		// As a last resort, presign redirect to original (client can downscale)
		if h.s3 != nil {
			if presigned, perr := h.s3.PresignGet(c.Request.Context(), objectKey, 5*time.Minute); perr == nil {
				c.Header("Cache-Control", "private, max-age=60")
				c.Redirect(http.StatusFound, presigned)
				return
			}
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "source unavailable"})
		return
	}
	defer src.Close()

	// Decode, resize, and encode JPEG/PNG output depending on original type
	// We use the standard library for decode( png/jpeg ) and a simple nearest-neighbor scale to avoid heavy deps.
	// If performance/quality is insufficient, we can swap to github.com/disintegration/imaging later.
	data, err := io.ReadAll(io.LimitReader(src, 32<<20)) // limit 32MB decode for safety
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"read failed"}); return }

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		// Unsupported format by stdlib: fallback to presigned
		if h.s3 != nil {
			if presigned, perr := h.s3.PresignGet(c.Request.Context(), objectKey, 5*time.Minute); perr == nil {
				c.Header("Cache-Control", "private, max-age=60")
				c.Redirect(http.StatusFound, presigned)
				return
			}
		}
		c.JSON(http.StatusBadRequest, gin.H{"error":"decode failed"})
		return
	}
	b := img.Bounds()
	if b.Dx() <= width {
		// No upscale; just return original cached or newly cached original
		// Save original to thumbPath (copy) to unify caching
		if err := localcache.Save(thumbPath, bytes.NewReader(data)); err == nil {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
			c.File(thumbPath)
			return
		}
		c.Data(http.StatusOK, contentType, data)
		return
	}
	// Compute proportional height
	height := int(float64(b.Dy()) * (float64(width) / float64(b.Dx())))
	if height <= 0 { height = 1 }

	// Simple nearest-neighbor scaling
	dst := image.NewRGBA(image.Rect(0,0,width,height))
	for y := 0; y < height; y++ {
		sy := y * b.Dy() / height
		for x := 0; x < width; x++ {
			sx := x * b.Dx() / width
			dst.Set(x, y, img.At(b.Min.X+sx, b.Min.Y+sy))
		}
	}

	// Encode as JPEG for wide compatibility unless original was PNG with transparency
	buf := new(bytes.Buffer)
	ct := "image/jpeg"
	if format == "png" {
		// Try to preserve PNG if likely transparency; for simplicity, encode PNG always here
		if err := png.Encode(buf, dst); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"encode failed"}); return }
		ct = "image/png"
	} else {
		// Use default quality ~75
		if err := jpeg.Encode(buf, dst, &jpeg.Options{Quality: 75}); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"encode failed"}); return }
	}

	// Cache and serve
	if err := localcache.Save(thumbPath, bytes.NewReader(buf.Bytes())); err == nil {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Data(http.StatusOK, ct, buf.Bytes())
		return
	}
	c.Header("Cache-Control", "private, max-age=60")
	c.Data(http.StatusOK, ct, buf.Bytes())
}
 
