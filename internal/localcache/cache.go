package localcache

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "os"
    "path/filepath"
)

// Dir returns the base cache directory.
func Dir() string { return ".cache" }

// PhotoPath returns the local path for a given object key.
func PhotoPath(objectKey string) string {
    // Normalize into a stable hashed directory structure to avoid overly deep trees
    sum := sha256.Sum256([]byte(objectKey))
    hexsum := hex.EncodeToString(sum[:])
    // Use first 2 bytes as shard
    shard := hexsum[:2]
    filename := filepath.Base(objectKey)
    if filename == "." || filename == string(os.PathSeparator) || filename == "" {
        filename = hexsum
    }
    return filepath.Join(Dir(), "photos", shard, filename)
}

// ThumbPath returns the local path for a given object key and size spec (e.g., "w480", "w320h240").
func ThumbPath(objectKey, spec string) string {
    sum := sha256.Sum256([]byte(objectKey))
    hexsum := hex.EncodeToString(sum[:])
    shard := hexsum[:2]
    filename := filepath.Base(objectKey)
    if filename == "." || filename == string(os.PathSeparator) || filename == "" {
        filename = hexsum
    }
    // Always encode thumbnail as jpg or png; we keep original filename plus spec suffix for uniqueness
    return filepath.Join(Dir(), "thumbs", spec, shard, filename)
}

// EnsureDir ensures the directory exists.
func EnsureDir(path string) error {
    return os.MkdirAll(filepath.Dir(path), 0o755)
}

// Save writes r to the given path atomically.
func Save(path string, r io.Reader) error {
    if err := EnsureDir(path); err != nil {
        return err
    }
    tmp := fmt.Sprintf("%s.tmp-%d", path, os.Getpid())
    f, err := os.Create(tmp)
    if err != nil {
        return err
    }
    _, werr := io.Copy(f, r)
    cerr := f.Close()
    if werr != nil {
        _ = os.Remove(tmp)
        return werr
    }
    if cerr != nil {
        _ = os.Remove(tmp)
        return cerr
    }
    if err := os.Rename(tmp, path); err != nil {
        _ = os.Remove(tmp)
        return err
    }
    return nil
}

// Exists checks if a file exists at path.
func Exists(path string) bool {
    if st, err := os.Stat(path); err == nil && st.Mode().IsRegular() {
        return true
    }
    return false
}
