package xmlresp

import (
	"encoding/xml"
	"net/http"
	"time"

	"repo3/internal/storage"
)

type errorResponse struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
	Key     string   `xml:"Key,omitempty"`
}

type listBucketsResult struct {
	XMLName xml.Name     `xml:"ListAllMyBucketsResult"`
	Xmlns   string       `xml:"xmlns,attr"`
	Owner   owner        `xml:"Owner"`
	Buckets bucketResult `xml:"Buckets"`
}

type owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type bucketResult struct {
	Buckets []bucketEntry `xml:"Bucket"`
}

type bucketEntry struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

type listBucketResult struct {
	XMLName     xml.Name      `xml:"ListBucketResult"`
	Xmlns       string        `xml:"xmlns,attr"`
	Name        string        `xml:"Name"`
	Prefix      string        `xml:"Prefix"`
	KeyCount    int           `xml:"KeyCount"`
	MaxKeys     int           `xml:"MaxKeys"`
	IsTruncated bool          `xml:"IsTruncated"`
	Contents    []objectEntry `xml:"Contents"`
}

type objectEntry struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

func WriteError(w http.ResponseWriter, status int, code, message, key string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_ = xml.NewEncoder(w).Encode(errorResponse{Code: code, Message: message, Key: key})
}

func WriteListBuckets(w http.ResponseWriter, buckets []storage.Bucket) {
	entries := make([]bucketEntry, 0, len(buckets))
	for _, bucket := range buckets {
		entries = append(entries, bucketEntry{
			Name:         bucket.Name,
			CreationDate: formatS3Time(bucket.CreationDate),
		})
	}

	writeXML(w, listBucketsResult{
		Xmlns:   "http://s3.amazonaws.com/doc/2006-03-01/",
		Owner:   owner{ID: "repo3", DisplayName: "repo3"},
		Buckets: bucketResult{Buckets: entries},
	})
}

func WriteListObjects(w http.ResponseWriter, bucket, prefix string, objects []storage.ObjectInfo) {
	entries := make([]objectEntry, 0, len(objects))
	for _, obj := range objects {
		entries = append(entries, objectEntry{
			Key:          obj.Key,
			LastModified: formatS3Time(obj.LastModified),
			ETag:         `"` + obj.ETag + `"`,
			Size:         obj.Size,
			StorageClass: "STANDARD",
		})
	}

	writeXML(w, listBucketResult{
		Xmlns:       "http://s3.amazonaws.com/doc/2006-03-01/",
		Name:        bucket,
		Prefix:      prefix,
		KeyCount:    len(entries),
		MaxKeys:     1000,
		IsTruncated: false,
		Contents:    entries,
	})
}

func writeXML(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml.Header))
	_ = xml.NewEncoder(w).Encode(value)
}

func formatS3Time(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
