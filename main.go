package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pterm/pterm"
)

const (
	endpoint = "http://app-homevision-staging.herokuapp.com/api_project/houses"
)

var (
	saveDir   = "./out"
	pageCount = 10
)

type house struct {
	ID        int    `json:"id"`
	Address   string `json:"address"`
	Homeowner string `json:"homeowner"`
	Price     int    `json:"price"`
	PhotoURL  string `json:"photoURL"`
}

type getHomesResponse struct {
	Houses []house `json:"houses"`
	Ok     bool    `json:"ok"`
}

type downloadStatus struct {
	page    int
	total   int
	current int
}

func main() {
	flag.IntVar(&pageCount, "pageCount", pageCount, "The number of pages to download")
	flag.StringVar(&saveDir, "downloadPath", saveDir, "The directory to download the images to")

	flag.Parse()

	// Create the download directory if it doesn't exist
	if _, err := os.Stat(saveDir); os.IsNotExist(err) {
		os.Mkdir(saveDir, 0755)
	}

	// I'm assuming that each page has exactly 10 houses, so the total number of downloads would be 10 * pageCount
	progressBar, _ := pterm.DefaultProgressbar.WithTotal(pageCount * 10).WithTitle("Downloading images...").Start()
	defer func() {
		progressBar.Stop()
		pterm.Success.Printfln("Downloaded %d images", progressBar.Total)
	}()

	var pm sync.Mutex

	logProgress := func(d downloadStatus) {
		// Not entirely sure if the increment function on the progress bar is thread safe,
		// so we'll use a mutex just to be safe
		pm.Lock()
		defer pm.Unlock()
		progressBar.Increment()
	}

	var wg sync.WaitGroup

	for i := 1; i <= pageCount; i++ {
		wg.Add(1)

		// To avoid capturing the loop variable
		pageNumber := i
		go downloadImages(pageNumber, logProgress, &wg)
	}

	wg.Wait()
}

// downloadImages will download the images from the given page number
func downloadImages(pageNumber int, notify func(downloadStatus), wg *sync.WaitGroup) {
	var houses []house

	// First fetch the houses
	retryIndefintely(func() (err error) {
		houses, err = fetchHouses(pageNumber)
		return
	})

	// Keeps track of downloaded images
	var dc uint32
	c := make(chan struct{}, len(houses))

	for _, h := range houses {
		house := h
		go retryIndefintely(func() error {
			ext, err := getFileExtension(house.PhotoURL)
			if err != nil {
				return fmt.Errorf("Error getting file extension: %s", err)
			}
			fileName := fmt.Sprintf("%d-%s-%s.%s", house.ID, house.Homeowner, house.Address, ext)
			filePath := path.Join(saveDir, fileName)
			err = downloadImage(house.PhotoURL, filePath)
			if err != nil {
				return err
			}
			c <- struct{}{}
			return nil
		})
	}

	for i := 0; i < len(houses); i++ {
		<-c
		atomic.AddUint32(&dc, 1)
		notify(downloadStatus{page: pageNumber, total: len(houses), current: int(dc)})
	}

	wg.Done()
}

// fetchHouses will fetch the houses for the given page number
func fetchHouses(pageNumber int) ([]house, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	u.RawQuery = url.Values{
		"page": []string{fmt.Sprintf("%d", pageNumber)},
	}.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Error fetching page: %d", pageNumber)
	}

	body, _ := ioutil.ReadAll(resp.Body)

	var r getHomesResponse
	err = json.Unmarshal(body, &r)
	if err != nil {
		return nil, err
	}

	return r.Houses, nil
}

// downloadImage will download the image from the given url and save it to the given path
func downloadImage(url string, filePath string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// retryIndefintely will retry the given function until it succeeds.
// In a more serious application, it's probably a bad idea to retry indefinitely without some sort of limit,
// but for this usecase, it's fine.
func retryIndefintely(f func() error) {
	for {
		err := f()
		if err == nil {
			return
		}
	}
}

// getFileExtension infers the file extension from a url by probing the content type
func getFileExtension(url string) (string, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	mimeHeader := resp.Header.Get("Content-Type")
	mimeType := mimeHeader[strings.Index(mimeHeader, "/")+1:]

	switch mimeType {
	case "jpeg":
		return "jpg", nil
	}

	return mimeType, nil
}
