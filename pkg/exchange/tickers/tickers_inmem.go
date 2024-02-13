package tickers

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TickersSourceInMem struct {
	FilePaths    []string
	UseTodayDate bool

	tickersLock *sync.RWMutex
	tickers     []Tick

	channelsLock *sync.RWMutex
	channels     []chan Tick
}

func (d *TickersSourceInMem) Init() error {
	if len(d.FilePaths) < 1 {
		return errors.New("empty list of input files for tickers data")
	}

	d.tickersLock = &sync.RWMutex{}
	d.channelsLock = &sync.RWMutex{}

	d.channels = make([]chan Tick, 0, 2)

	d.tickersLock.Lock()
	defer d.tickersLock.Unlock()
	d.tickers = make([]Tick, 0, 300000)

	//read all to memory
	for _, f := range d.FilePaths {
		file, err := os.Open(f)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Scan() // skip header
		for scanner.Scan() {
			s := strings.Split(scanner.Text(), `,`)

			var ts time.Time
			if d.UseTodayDate {
				ts, err = time.Parse("20060102 150405 MST", fmt.Sprintf("%v %v MSK", time.Now().Format("20060102"), s[3]))
			} else {
				ts, err = time.Parse("20060102 150405 MST", fmt.Sprintf("%v %v MSK", s[2], s[3]))
			}
			if err != nil {
				return err
			}

			l, err := strconv.ParseFloat(strings.Split(s[4], `.`)[0], 32)
			if err != nil {
				return err
			}

			v, err := strconv.ParseInt(s[5], 10, 32)
			if err != nil {
				return err
			}

			t := &Tick{
				Ticker:    s[0],
				Timestamp: ts,
				Last:      float32(l),
				Vol:       int32(v),
			}

			d.tickers = append(d.tickers, *t)
		}
	}

	// sort by timestamp ascending
	sort.SliceStable(d.tickers, func(i, j int) bool {
		return d.tickers[i].Timestamp.Before(d.tickers[j].Timestamp)
	})

	fmt.Println("Historical data load completed")
	fmt.Println("Starting tickers feed")

	go d.feed()

	return nil
}

func (d *TickersSourceInMem) GetFeedChannel() <-chan Tick {
	c := make(chan Tick, 100)

	d.channelsLock.Lock()
	d.channels = append(d.channels, c)
	d.channelsLock.Unlock()

	return c
}

func (d *TickersSourceInMem) CloseFeed() {
	d.channelsLock.Lock()
	defer d.channelsLock.Unlock()

	for _, v := range d.channels {
		close(v)
	}

	//clean up channels, not needed for now since it should happen only on final stop of exchange
	//d.channels = make([]chan Tick, 0, 2)
}

/* another way to feed consumers with tickers
 */
func (d *TickersSourceInMem) feed() {
	// discard everything before exchange startup
	now := time.Now()
	for i, v := range d.tickers {
		if v.Timestamp.After(now) {
			d.tickers = d.tickers[i:]
			break
		}
	}

	interval := time.Second * 1
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// each second push all tickers with appropriate timestamp to listeners
	var l int64
	for range ticker.C {
		d.tickersLock.RLock()

		var maxid int
		tsnow := time.Now()
		for j, k := range d.tickers {
			if k.Timestamp.Before(tsnow) {
				maxid = j + 1

				d.channelsLock.Lock()
				for _, c := range d.channels {
					c <- k
				}
				d.channelsLock.Unlock()
			} else {
				break
			}
		}
		d.tickers = d.tickers[maxid:]
		d.tickersLock.RUnlock()

		l += 1
	}
}
