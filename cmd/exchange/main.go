package main

import (
	"context"
	"fmt"

	"github.com/KSerditov/Trading/pkg/exchange/server"
	"github.com/KSerditov/Trading/pkg/exchange/tickers"
)

/* TBD FOR EXCHANGE
1. Use channel to translate new tickers from tickers_inmem to trader and exchange statistics sender
(tickers_inmem has infinite cycle that sends slice elements only if it fits time)

2. Add authentication (brokerid - key based?)

3. Add logging

4. Validate nonunique broker id connections

5. Write tests (consider separationg of trader layer from grpc server)

6. Add configurations

7. Last partial deal should return partial = false

8. Initialize DB instance and store everything there

9. Data types are not consistent between broker and exchange
*/

func main() {
	tickers := &tickers.TickersSourceInMem{
		FilePaths:    []string{`.\assets\SPFB.RTS_190517_190517.txt`, `.\assets\SPFB.Si_190517_190517.txt`},
		UseTodayDate: true,
	}
	err := tickers.Init()
	if err != nil {
		fmt.Println(err)
	}

	/*ch := tickers.GetFeedChannel()
	for {
		select {
		case ticker := <-ch:
			fmt.Printf("NEW TICKER: %v\n", ticker)
		}

	}*/

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = server.Start(ctx, `127.0.0.1:8082`, ``, tickers)
	if err != nil {
		fmt.Println(err)
	}
}
