package tickers

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TickersSourceInMem struct {
	FilePaths    []string
	UseTodayDate bool

	TickersLock *sync.RWMutex
	Tickers     []Tick
}

func (d *TickersSourceInMem) GetTickersBeforeTS(ts time.Time, beforeInterval time.Duration) ([]Tick, error) {
	d.TickersLock.RLock()
	defer d.TickersLock.RUnlock()

	tickers := make([]Tick, 0, 100)
	for _, v := range d.Tickers {
		if v.Timestamp.Before(ts) && v.Timestamp.After(ts.Add(-beforeInterval)) {
			tickers = append(tickers, v)
		}
	}

	return tickers, nil
}

func (d *TickersSourceInMem) Init() error {
	if len(d.FilePaths) < 1 {
		return errors.New("empty list of input files for tickers data")
	}

	d.TickersLock.Lock()
	defer d.TickersLock.Unlock()
	d.Tickers = make([]Tick, 0, 300000)

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

			d.Tickers = append(d.Tickers, *t)
		}
	}

	return nil
}
