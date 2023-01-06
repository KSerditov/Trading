package server

import (
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// какой адрес-порт слушать серверу
	listenAddr string = "127.0.0.1:8082"
)

func getGrpcConn(t *testing.T) *grpc.ClientConn {
	grcpConn, err := grpc.Dial(
		listenAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("cant connect to grpc: %v", err)
	}
	return grcpConn
}

/*
func TestStat(t *testing.T) {
	ctx, finish := context.WithCancel(context.Background())
	err := Start(ctx, `127.0.0.1:8082`, ``, tickers)
	if err != nil {
		t.Fatalf("cant start server initial: %v", err)
	}
	wait(1)
	defer func() {
		finish()
		wait(2)
	}()

	conn := getGrpcConn(t)
	defer conn.Close()

	biz := NewBizClient(conn)
	adm := NewAdminClient(conn)

	statStream1, err := adm.Statistics(getConsumerCtx("stat1"), &StatInterval{IntervalSeconds: 2})
	wait(1)
	statStream2, err := adm.Statistics(getConsumerCtx("stat2"), &StatInterval{IntervalSeconds: 3})

	mu := &sync.Mutex{}
	stat1 := &Stat{}
	stat2 := &Stat{}

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for {
			stat, err := statStream1.Recv()
			if err != nil && err != io.EOF {
				// fmt.Printf("unexpected error %v\n", err)
				return
			} else if err == io.EOF {
				break
			}
			// log.Println("stat1", stat, err)
			mu.Lock()
			// это грязный хак
			// protobuf добавляет к структуре свои поля, которвые не видны при приведении к строке и при reflect.DeepEqual
			// поэтому берем не оригинал сообщения, а только нужные значения
			stat1 = &Stat{
				ByMethod:   stat.ByMethod,
				ByConsumer: stat.ByConsumer,
			}
			mu.Unlock()
		}
	}()
	go func() {
		for {
			stat, err := statStream2.Recv()
			if err != nil && err != io.EOF {
				// fmt.Printf("unexpected error %v\n", err)
				return
			} else if err == io.EOF {
				break
			}
			// log.Println("stat2", stat, err)
			mu.Lock()
			// это грязный хак
			// protobuf добавляет к структуре свои поля, которвые не видны при приведении к строке и при reflect.DeepEqual
			// поэтому берем не оригинал сообщения, а только нужные значения
			stat2 = &Stat{
				ByMethod:   stat.ByMethod,
				ByConsumer: stat.ByConsumer,
			}
			mu.Unlock()
		}
	}()

	wait(1)

	biz.Check(getConsumerCtx("biz_user"), &Nothing{})
	biz.Add(getConsumerCtx("biz_user"), &Nothing{})
	biz.Test(getConsumerCtx("biz_admin"), &Nothing{})

	wait(200) // 2 sec

	expectedStat1 := &Stat{
		ByMethod: map[string]uint64{
			"/main.Biz/Check":        1,
			"/main.Biz/Add":          1,
			"/main.Biz/Test":         1,
			"/main.Admin/Statistics": 1,
		},
		ByConsumer: map[string]uint64{
			"biz_user":  2,
			"biz_admin": 1,
			"stat2":     1,
		},
	}

	mu.Lock()
	if !reflect.DeepEqual(stat1, expectedStat1) {
		t.Fatalf("stat1-1 dont match\nhave %+v\nwant %+v", stat1, expectedStat1)
	}
	mu.Unlock()

	biz.Add(getConsumerCtx("biz_admin"), &Nothing{})

	wait(220) // 2+ sec

	expectedStat1 = &Stat{
		Timestamp: 0,
		ByMethod: map[string]uint64{
			"/main.Biz/Add": 1,
		},
		ByConsumer: map[string]uint64{
			"biz_admin": 1,
		},
	}
	expectedStat2 := &Stat{
		Timestamp: 0,
		ByMethod: map[string]uint64{
			"/main.Biz/Check": 1,
			"/main.Biz/Add":   2,
			"/main.Biz/Test":  1,
		},
		ByConsumer: map[string]uint64{
			"biz_user":  2,
			"biz_admin": 2,
		},
	}

	mu.Lock()
	if !reflect.DeepEqual(stat1, expectedStat1) {
		t.Fatalf("stat1-2 dont match\nhave %+v\nwant %+v", stat1, expectedStat1)
	}
	if !reflect.DeepEqual(stat2, expectedStat2) {
		t.Fatalf("stat2 dont match\nhave %+v\nwant %+v", stat2, expectedStat2)
	}
	mu.Unlock()

	finish()
}
*/
