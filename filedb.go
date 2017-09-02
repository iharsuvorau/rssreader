package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/iharsuvorau/rssreader/rss"
)

// FileDatabase is a database of feeds' locations.
type FileDatabase struct {
	Location string
	Feeds    []Feed
}

// Feed is an internal type for feeds.
type Feed struct {
	Loc  string // URL
	Kind string // xml, json, json:gunzip
}

func (fdb *FileDatabase) init() error {
	_, err := os.Stat(fdb.Location)

	if os.IsNotExist(err) {
		if _, err = os.Create(fdb.Location); err != nil {
			return fmt.Errorf("can't create the file at %s: %s", fdb.Location, err)
		}
		fmt.Println("New feeds collection has been created.")
	}

	return nil
}

func (fdb *FileDatabase) save(url, kind string) error {
	f, err := os.OpenFile(fdb.Location, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("can't open the file at %s: %s", fdb.Location, err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}

	if len(kind) > 0 {
		url = fmt.Sprintf("%s:%s", kind, url)
	}

	if _, err = f.WriteAt([]byte(url+"\n"), stat.Size()); err != nil {
		return err
	}

	return nil
}

func (fdb *FileDatabase) read() error {
	f, err := os.Open(fdb.Location)
	if err != nil {
		return fmt.Errorf("can't read the file at %s: %s", fdb.Location, err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		encodedURL := s.Text()
		parts := strings.Split(encodedURL, ":")
		switch parts[0] {
		case "xml", "json", "parsehub":
			fdb.Feeds = append(fdb.Feeds, Feed{strings.Join(parts[1:], ":"), parts[0]})
			continue
		default:
			fdb.Feeds = append(fdb.Feeds, Feed{encodedURL, "xml"})
			continue
		}
	}

	if err = s.Err(); err != nil {
		return err
	}

	return nil
}

func (fdb *FileDatabase) fetchAt(id int) (doc *rss.Document, err error) {
	if err = fdb.read(); err != nil {
		return nil, err
	}

	errs := make(chan error, len(fdb.Feeds))
	docs := make(chan *rss.Document, len(fdb.Feeds))
	fetch(fdb.Feeds[id], errs, docs)

	select {
	case err = <-errs:
		return doc, err
	case doc = <-docs:
		return doc, nil
	}

	// doc, err := rss.RetrieveRssFeed("rssreader", fdb.Feeds[id].Loc)
	// if err != nil {
	// 	return nil, err
	// }
}

func (fdb *FileDatabase) fetchAll() ([]*rss.Document, error) {
	err := fdb.read()
	if err != nil {
		return nil, err
	}

	errs := make(chan error, len(fdb.Feeds))
	docs := make(chan *rss.Document, len(fdb.Feeds))
	var wg sync.WaitGroup

	for _, feed := range fdb.Feeds {
		wg.Add(1)
		go func(feed Feed) {
			fetch(feed, errs, docs)
			wg.Done()
		}(feed)
	}

	wg.Wait()
	close(errs)
	close(docs)

	collection := []*rss.Document{}
	for doc := range docs {
		collection = append(collection, doc)
	}
	for err = range errs {
		if err != nil {
			return collection, err
		}
	}

	return collection, err
}

func (fdb *FileDatabase) list() error {
	docs, err := fdb.fetchAll()
	if err != nil {
		return err
	}

	for i, feed := range fdb.Feeds {
		for _, doc := range docs {
			if feed.Loc == doc.Channel.Link {
				fmt.Printf("[%d] %s\n", i, strings.Trim(doc.Channel.Title, "\n \t  "))
			}
		}
	}

	return nil
}

func (fdb *FileDatabase) listAt(id string) error {
	n, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	err = fdb.read()
	if err != nil {
		return err
	}

	doc, err := fdb.fetchAt(n)
	if err != nil {
		return err
	}

	sort.Sort(byTime(doc.Channel.Item))

	for _, item := range doc.Channel.Item {
		if len(item.PubDate) > 0 {
			fmt.Printf("%s %s (%s)\n", item.PubDate, item.Title, item.Link)
		} else {
			fmt.Printf("%s (%s)\n", item.Title, item.Link)
		}
	}

	return nil
}

// byTime implements sort.Interface for []rss.item based in PubDate
type byTime []rss.Item

func (a byTime) Len() int {
	return len(a)
}

func (a byTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a byTime) Less(i, j int) bool {
	t1, err := time.Parse(time.RFC1123Z, a[i].PubDate)
	if err != nil {
		return false
	}

	t2, err := time.Parse(time.RFC1123Z, a[j].PubDate)
	if err != nil {
		return false
	}

	return !t1.Before(t2)
}

//

func fetch(feed Feed, errs chan error, docs chan *rss.Document) {
	switch feed.Kind {
	case "xml":
		doc, err := rss.RetrieveRssFeed("rssreader", feed.Loc)
		if err != nil {
			errs <- err
		}
		doc.Channel.Link = feed.Loc

		docs <- doc
		return
	case "json":
		doc := &rss.Document{}
		doc.Channel.Link = feed.Loc

		url, err := url.Parse(feed.Loc)
		if err != nil {
			errs <- err
			return
		}
		doc.Channel.Title = url.Host

		resp, err := http.Get(feed.Loc)
		if err != nil {
			errs <- err
			return
		}

		defer resp.Body.Close()
		d := json.NewDecoder(resp.Body)

		data := make(map[string][]map[string]string)
		if err = d.Decode(&data); err != nil {
			errs <- err
			return
		}

		// JSON template: {"collection1": [{"name": "", "url": ""}], "collection2": ""}
		for _, items := range data {
			for _, item := range items {
				doc.Channel.Item = append(doc.Channel.Item, rss.Item{
					Title: item["name"],
					Link:  item["url"],
				})
			}
		}

		docs <- doc
		return
	case "parsehub":
		doc := &rss.Document{}
		doc.Channel.Link = feed.Loc

		url, err := url.Parse(feed.Loc)
		if err != nil {
			errs <- err
			return
		}
		doc.Channel.Title = url.Host

		// getting the project data
		resp, err := http.Get(feed.Loc)
		if err != nil {
			errs <- err
			return
		}

		d := json.NewDecoder(resp.Body)

		projectData := make(map[string]interface{})
		if err = d.Decode(&projectData); err != nil {
			errs <- err
			return
		}

		doc.Channel.Title = projectData["title"].(string)
		resp.Body.Close()

		// getting items
		url, err = url.Parse(feed.Loc)
		if err != nil {
			errs <- err
			return
		}
		url.Path += "/last_ready_run/data"

		if resp, err = http.Get(url.String()); err != nil {
			errs <- err
			return
		}

		d = json.NewDecoder(resp.Body)

		data := make(map[string][]map[string]string)
		if err = d.Decode(&data); err != nil {
			errs <- err
			return
		}

		resp.Body.Close()

		// JSON template: {"collection1": [{"name": "", "url": ""}], "collection2": ""}
		for _, items := range data {
			for _, item := range items {
				doc.Channel.Item = append(doc.Channel.Item, rss.Item{
					Title: item["name"],
					Link:  item["url"],
				})
			}
		}

		docs <- doc
		return
	}
}

func fetchParseHubProject(loc string) (doc *rss.Document, err error) {
	doc = new(rss.Document)
	doc.Channel.Link = loc

	url, err := url.Parse(loc)
	if err != nil {
		return
	}

	doc.Channel.Title = url.Host

	// getting the project data
	resp, err := http.Get(loc)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	projectData := make(map[string]interface{})

	if err = json.NewDecoder(resp.Body).Decode(&projectData); err != nil {
		return
	}

	doc.Channel.Title = projectData["title"].(string)
	return doc, nil
}
