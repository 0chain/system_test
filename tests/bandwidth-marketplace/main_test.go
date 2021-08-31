package bandwidth_marketplace

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/0chain/gosdk/zmagmacore/errors"
	clog "github.com/0chain/gosdk/zmagmacore/log"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/config"
	"github.com/0chain/system_test/internal/bandwidth-marketplace/docker"
	"github.com/0chain/system_test/internal/bandwidth-marketplace/log"
)

var (
	testCfg *config.Config
)

func TestMain(m *testing.M) {
	_ = docker.CleanDirs()
	_ = docker.InitDirs()

	var err error
	testCfg, err = config.Read()
	exitIfErr(err)

	log.SetupLogger(testCfg.Log)
	clog.Logger = log.Logger

	dClient, err := docker.NewClient(testCfg)
	exitIfErr(err)

	errCh, err := dClient.PullAndStartAll(context.Background())
	exitIfErr(err)

	var (
		mainCtx, mainCancel = context.WithCancel(context.Background())

		waitShutdownCh = make(chan struct{})
	)
	go func() {
		dClient.HandleShutdown(mainCtx, errCh)
		waitShutdownCh <- struct{}{}
	}()

	waitNodes()
	st := m.Run()
	mainCancel()
	<-waitShutdownCh
	os.Exit(st)
}

func waitNodes() {
	sleepTime := time.Duration(testCfg.BuildingWaitTime) * time.Second
	log.Logger.Info(fmt.Sprintf("Wait %s for starting all containers.", sleepTime.String()))
	time.Sleep(sleepTime)
}

func exitIfErr(err error) {
	if err != nil {
		errors.ExitMsg(err.Error(), 2)
	}
}
