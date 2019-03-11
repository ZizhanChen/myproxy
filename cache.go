package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
)

// CachedResponse respresents a response to be cached.
type CachedResponse struct {
	StatusCode int
	Body       []byte
	Headers    map[string]string
}

// SpecResponse represents a specification for a response.
type SpecResponse struct {
	StatusCode  int               `json:"status_code"`
	ContentFile string            `json:"content"`
	Headers     map[string]string `json:"headers"`
}

// Spec represents a full specification to describe a response and how to look up its index.
type Spec struct {
	SpecResponse `json:"response"`
	Key          string `json:"key"`
}

// A FileSystem interface is used to provide a mechanism of storing and retreiving files to/from disk.
type FileSystem interface {
	WriteFile(path string, content []byte) error
	ReadFile(path string) ([]byte, error)
}

// DefaultFileSystem provides a default implementation of a filesystem on disk.
type DefaultFileSystem struct {
}

// WriteFile writes content to p2p network.
func (fs DefaultFileSystem) WriteFile(myPath string, content []byte) error {
	// first write the content to the local file
	err := ioutil.WriteFile(myPath, content, 0644)
	if err != nil {
		panic(err)
	}
	// if the content is nil, just return
	if len(content) == 0{
		return err
	}

	awsPath := "s3://mybucket" + myPath
	// use the awscli command store the file to the p2p network
	cmd := exec.Command("aws",
		"s3",
		"--endpoint=http://localhost:9000/",
		"cp",
		myPath,
		awsPath)
	res := cmd.Run()
	// remove the local content
	err = os.Remove(myPath)
	if err != nil {
		panic(err)
	}
	return res
}

// ReadFile reads content from the p2p network
func (fs DefaultFileSystem) ReadFile(myPath string) ([]byte, error) {
	// get the current file path
	awsPath := "s3://mybucket" + myPath
	// use the awscli command get the file from the p2p network
	cmd := exec.Command("aws",
		"s3",
		"--endpoint=http://localhost:9000/",
		"cp",
		awsPath,
		myPath)
	cmd.Run()
	content, err := ioutil.ReadFile(myPath)
	os.Remove(myPath)
	return content, err
}

// A Cacher interface is used to provide a mechanism of storage for a given request and response.
type Cacher interface {
	Get(key string) *CachedResponse
	Put(key string, r *httptest.ResponseRecorder) *CachedResponse
}

// DiskCacher is the default cacher which writes to disk
type DiskCacher struct {
	cache    map[string]*CachedResponse
	dataDir  string
	specPath string
	mutex    *sync.RWMutex
	FileSystem
}

// NewDiskCacher creates a new disk cacher for a given data directory.
func NewDiskCacher(dataDir string) DiskCacher {
	absPath, err := os.Getwd()
	if err != nil{
		panic (err)
	}
	return DiskCacher{
		cache:      make(map[string]*CachedResponse),
		dataDir:    path.Join(absPath, dataDir),
		specPath:   path.Join(absPath, dataDir, "spec.json"),
		mutex:      new(sync.RWMutex),
		FileSystem: DefaultFileSystem{},
	}
}

// SeedCache populates the DiskCacher with entries from disk.
func (c *DiskCacher) SeedCache() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	specs := c.loadSpecs()

	for _, spec := range specs {
		body, err := c.FileSystem.ReadFile(path.Join(c.dataDir, spec.SpecResponse.ContentFile))
		if err != nil {
			panic(err)
		}
		response := &CachedResponse{
			StatusCode: spec.StatusCode,
			Headers:    spec.Headers,
			Body:       body,
		}
		c.cache[spec.Key] = response
	}
}

// Get fetches a CachedResponse for a given key
func (c DiskCacher) Get(key string) *CachedResponse {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.cache[key]
}

func (c DiskCacher) loadSpecs() []Spec {
	specContent, err := c.FileSystem.ReadFile(c.specPath)
	if err != nil {
		specContent = []byte{'[', ']'}
	}

	var specs []Spec
	err = json.Unmarshal(specContent, &specs)
	if err != nil {
		panic(err)
	}

	return specs
}

// Put stores a CachedResponse for a given key and response
func (c DiskCacher) Put(key string, resp *httptest.ResponseRecorder) *CachedResponse {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	skipDisk := resp.Header().Get("_chameleon-seeded-skip-disk") != ""
	if skipDisk {
		resp.Header().Del("_chameleon-seeded-skip-disk")
	}

	specHeaders := make(map[string]string)
	for k, v := range resp.Header() {
		specHeaders[k] = strings.Join(v, ", ")
	}

	if !skipDisk {
		specs := c.loadSpecs()

		newSpec := Spec{
			Key: key,
			SpecResponse: SpecResponse{
				StatusCode:  resp.Code,
				ContentFile: key,
				Headers:     specHeaders,
			},
		}

		specs = append(specs, newSpec)

		contentFilePath := path.Join(c.dataDir, key)
		err := c.FileSystem.WriteFile(contentFilePath, resp.Body.Bytes())
		if err != nil {
			panic(err)
		}

		specBytes, err := json.MarshalIndent(specs, "", "    ")
		err = c.FileSystem.WriteFile(c.specPath, specBytes)
		if err != nil {
			panic(err)
		}
	}

	c.cache[key] = &CachedResponse{
		StatusCode: resp.Code,
		Headers:    specHeaders,
		Body:       resp.Body.Bytes(),
	}

	return c.cache[key]
}
