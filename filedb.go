package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/goinggo/newssearch/rss"
)

const fdbLocation = "/Users/ihar/.feeds"

// FileDatabase is a database of feeds' locations.
type FileDatabase struct {
	Location string
	Urls     []string
}

func (fdb *FileDatabase) init() error {
	_, err := os.Stat(fdb.Location)

	if os.IsNotExist(err) {
		if _, err = os.Create(fdb.Location); err != nil {
			return err
		}
		fmt.Println("New feeds collection has been created.")
	}

	return nil
}

func (fdb *FileDatabase) save(url string) error {
	f, err := os.OpenFile(fdb.Location, os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}

	if _, err = f.WriteAt([]byte(url+"\n"), stat.Size()); err != nil {
		return err
	}

	return nil
}

func (fdb *FileDatabase) read() error {
	f, err := os.Open(fdb.Location)
	if err != nil {
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		fdb.Urls = append(fdb.Urls, s.Text())
	}

	if err = s.Err(); err != nil {
		return err
	}

	return nil
}

func (fdb *FileDatabase) fetch(id int) (*rss.Document, error) {
	err := fdb.read()
	if err != nil {
		return nil, err
	}

	doc, err := rss.RetrieveRssFeed("rssreader", fdb.Urls[id])
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func (fdb *FileDatabase) fetchAll() ([]*rss.Document, error) {
	err := fdb.read()
	if err != nil {
		return nil, err
	}

	docs := []*rss.Document{}

	for _, url := range fdb.Urls {
		doc, err := rss.RetrieveRssFeed("rssreader", url)
		if err != nil {
			return nil, err
		}

		docs = append(docs, doc)
	}

	return docs, nil
}

func (fdb *FileDatabase) list() error {
	docs, err := fdb.fetchAll()
	if err != nil {
		return err
	}

	for i, doc := range docs {
		fmt.Printf("[%d] %s\n", i, strings.Trim(doc.Channel.Title, "\n \t  "))
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

	doc, err := fdb.fetch(n)
	if err != nil {
		return err
	}

	sort.Sort(byTime(doc.Channel.Item))

	for _, item := range doc.Channel.Item {
		fmt.Printf("%s %s (%s)\n", item.PubDate, item.Title, item.Link)
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
