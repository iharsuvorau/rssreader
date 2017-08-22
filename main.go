// rssreader manages feeds' locations and show items for a particular feed.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/goinggo/newssearch/rss"
)

const fdbLocation = "/Users/ihar/.feeds"

// FileDatabase is database of feeds' locations.
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
		fmt.Printf("[%d] %s\n", i, doc.Channel.Title)
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

	for _, item := range doc.Channel.Item {
		fmt.Printf("%s %s (%s)\n", item.PubDate, item.Title, item.Link)
	}

	return nil
}

func main() {
	url := flag.String("a", "", "add a feed's URL to fetch")
	list := flag.Bool("l", false, "show a list of current feeds")
	show := flag.String("s", "", "show a list of items for a feed with the specified index")
	flag.Parse()

	fdb := FileDatabase{
		Location: fdbLocation,
		Urls:     []string{},
	}

	var err error

	if err = fdb.init(); err != nil {
		fmt.Println(err)
	}

	if len(*url) > 0 {
		if err = fdb.save(*url); err != nil {
			fmt.Println(err)
		}
	}

	if *list {
		if err = fdb.list(); err != nil {
			fmt.Println(err)
		}
	}

	if len(*show) > 0 {
		if err = fdb.listAt(*show); err != nil {
			fmt.Println(err)
		}
	}
}
