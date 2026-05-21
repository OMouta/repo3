package s3api

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"repo3/internal/storage"
	"repo3/internal/xmlresp"
)

type Handler struct {
	store storage.ObjectStore
}

func NewHandler(store storage.ObjectStore) http.Handler {
	return &Handler{store: store}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bucket, key := splitPath(r.URL.Path)

	if bucket == "" {
		if r.Method == http.MethodGet {
			h.listBuckets(w, r)
			return
		}
		writeError(w, r, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}

	if key == "" {
		switch r.Method {
		case http.MethodPut:
			h.createBucket(w, r, bucket)
		case http.MethodGet:
			h.listObjects(w, r, bucket)
		case http.MethodDelete:
			h.deleteBucket(w, r, bucket)
		default:
			writeError(w, r, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		}
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.putObject(w, r, bucket, key)
	case http.MethodGet:
		h.getObject(w, r, bucket, key)
	case http.MethodHead:
		h.headObject(w, r, bucket, key)
	case http.MethodDelete:
		h.deleteObject(w, r, bucket, key)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

func (h *Handler) listBuckets(w http.ResponseWriter, r *http.Request) {
	buckets, err := h.store.ListBuckets(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "InternalError", err.Error())
		return
	}
	xmlresp.WriteListBuckets(w, buckets)
}

func (h *Handler) createBucket(w http.ResponseWriter, r *http.Request, bucket string) {
	err := h.store.CreateBucket(r.Context(), bucket)
	if err != nil && !errors.Is(err, storage.ErrBucketExists) {
		writeMappedError(w, r, err, "")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) deleteBucket(w http.ResponseWriter, r *http.Request, bucket string) {
	if err := h.store.DeleteBucket(r.Context(), bucket); err != nil {
		writeMappedError(w, r, err, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) putObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	result, err := h.store.PutObject(r.Context(), bucket, key, r.Body, storage.Metadata{
		ContentType:  r.Header.Get("Content-Type"),
		UserMetadata: userMetadataFromHeaders(r.Header),
	})
	if err != nil {
		writeMappedError(w, r, err, key)
		return
	}
	w.Header().Set("ETag", strconv.Quote(result.ETag))
	if result.VersionID != "" {
		w.Header().Set("x-amz-version-id", result.VersionID)
	}
	w.WriteHeader(http.StatusOK)
}

func userMetadataFromHeaders(headers http.Header) map[string]string {
	const prefix = "X-Amz-Meta-"
	meta := map[string]string{}
	for key, values := range headers {
		if !strings.HasPrefix(http.CanonicalHeaderKey(key), prefix) || len(values) == 0 {
			continue
		}
		metaKey := strings.TrimPrefix(http.CanonicalHeaderKey(key), prefix)
		if metaKey == "" {
			continue
		}
		meta[strings.ToLower(metaKey)] = values[0]
	}
	if len(meta) == 0 {
		return nil
	}
	return meta
}

func (h *Handler) getObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	obj, err := h.store.GetObject(r.Context(), bucket, key, r.URL.Query().Get("versionId"))
	if err != nil {
		writeMappedError(w, r, err, key)
		return
	}
	defer obj.Body.Close()
	writeObjectHeaders(w, obj.Info)
	_, _ = io.Copy(w, obj.Body)
}

func (h *Handler) headObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	info, err := h.store.HeadObject(r.Context(), bucket, key, r.URL.Query().Get("versionId"))
	if err != nil {
		writeMappedError(w, r, err, key)
		return
	}
	writeObjectHeaders(w, *info)
}

func (h *Handler) deleteObject(w http.ResponseWriter, r *http.Request, bucket, key string) {
	if err := h.store.DeleteObject(r.Context(), bucket, key); err != nil {
		writeMappedError(w, r, err, key)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listObjects(w http.ResponseWriter, r *http.Request, bucket string) {
	if r.URL.Query().Get("list-type") != "2" && r.URL.RawQuery != "" {
		writeError(w, r, http.StatusNotImplemented, "NotImplemented", "only list-type=2 is supported")
		return
	}

	limit := 1000
	if maxKeys := r.URL.Query().Get("max-keys"); maxKeys != "" {
		parsed, err := strconv.Atoi(maxKeys)
		if err == nil && parsed > 0 {
			limit = parsed
		}
	}
	objects, err := h.store.ListObjects(r.Context(), bucket, r.URL.Query().Get("prefix"), limit)
	if err != nil {
		writeMappedError(w, r, err, "")
		return
	}
	xmlresp.WriteListObjects(w, bucket, r.URL.Query().Get("prefix"), objects)
}

func writeObjectHeaders(w http.ResponseWriter, info storage.ObjectInfo) {
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	w.Header().Set("ETag", strconv.Quote(info.ETag))
	w.Header().Set("Last-Modified", info.LastModified.UTC().Format(http.TimeFormat))
	if info.ContentType != "" {
		w.Header().Set("Content-Type", info.ContentType)
	}
	if info.VersionID != "" {
		w.Header().Set("x-amz-version-id", info.VersionID)
	}
}

func writeMappedError(w http.ResponseWriter, r *http.Request, err error, key string) {
	switch {
	case errors.Is(err, storage.ErrNoSuchBucket):
		writeError(w, r, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist.")
	case errors.Is(err, storage.ErrNoSuchKey):
		writeErrorWithKey(w, r, http.StatusNotFound, "NoSuchKey", "The specified key does not exist.", key)
	case errors.Is(err, storage.ErrInvalidBucketName):
		writeError(w, r, http.StatusBadRequest, "InvalidBucketName", "The specified bucket is not valid.")
	case errors.Is(err, storage.ErrInvalidKey):
		writeErrorWithKey(w, r, http.StatusBadRequest, "InvalidKey", "The specified key is not valid.", key)
	case errors.Is(err, storage.ErrBucketNotEmpty):
		writeError(w, r, http.StatusConflict, "BucketNotEmpty", "The bucket you tried to delete is not empty.")
	case errors.Is(err, storage.ErrAccessDenied):
		writeError(w, r, http.StatusForbidden, "AccessDenied", err.Error())
	case errors.Is(err, storage.ErrSlowDown):
		writeError(w, r, http.StatusTooManyRequests, "SlowDown", err.Error())
	case errors.Is(err, storage.ErrOperationAborted):
		writeError(w, r, http.StatusConflict, "OperationAborted", err.Error())
	default:
		writeError(w, r, http.StatusInternalServerError, "InternalError", err.Error())
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeErrorWithKey(w, r, status, code, message, "")
}

func writeErrorWithKey(w http.ResponseWriter, r *http.Request, status int, code, message, key string) {
	if r.Method == http.MethodHead {
		w.WriteHeader(status)
		return
	}
	xmlresp.WriteError(w, status, code, message, key)
}

func splitPath(path string) (string, string) {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return "", ""
	}
	bucket, key, _ := strings.Cut(path, "/")
	return bucket, key
}
