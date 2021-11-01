package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/0chain/gosdk/zmagmacore/errors"
	"github.com/0chain/gosdk/zmagmacore/log"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.uber.org/zap"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/config"
)

type (
	// Client represents local docker client implementation.
	Client struct {
		dClient *client.Client

		containers []*dContainer
	}
)

// NewClient creates Client with configured magma, consumer, and provider containers.
func NewClient(cfg *config.Config) (*Client, error) {
	cl, err := client.NewClientWithOpts()
	if err != nil {
		return nil, err
	}

	magmaCont, err := newMagmaContainer(cfg.Magma)
	if err != nil {
		return nil, err
	}
	consCont, err := newConsumerContainer(cfg.Consumer)
	if err != nil {
		return nil, err
	}
	provCont, err := newProviderContainer(cfg.Provider)
	if err != nil {
		return nil, err
	}

	return &Client{
		dClient: cl,
		containers: []*dContainer{
			magmaCont,
			consCont,
			provCont,
		},
	}, nil
}

// PullAndStartAll pulls and starts all needed for tests containers.
//
// Returns error channel for reporting containers exit status or docker errors.
func (cl *Client) PullAndStartAll(ctx context.Context) (<-chan error, error) {
	var (
		mainErrCh = make(chan error)

		errCode = "pull_and_start"
	)

	for _, cont := range cl.containers {
		stCh, errCh, err := cl.pullAndStart(ctx, cont)
		if err != nil {
			return nil, err
		}

		go func(cont *dContainer) {
			select {
			case <-ctx.Done():
				break

			case err := <-errCh:
				msg := "error while starting" + cont.name + "container: " + err.Error()
				mainErrCh <- errors.New(errCode, msg)

			case st := <-stCh:
				msg := fmt.Sprintf("got %s removing exit status err: %v; exit code: %d", cont.name, st.Error, st.StatusCode)
				mainErrCh <- errors.New(errCode, msg)
			}

		}(cont)
	}
	return mainErrCh, nil
}

func (cl *Client) pullAndStart(ctx context.Context, cont *dContainer) (<-chan container.ContainerWaitOKBody, <-chan error, error) {
	if err := cl.pull(ctx, cont.ref); err != nil {
		return nil, nil, err
	}

	_ = cl.stopAndRemoveContainer(ctx, cont.name)

	stCh, errCh, err := cl.start(ctx, cont)
	if err != nil {
		return nil, nil, err
	}

	return stCh, errCh, err
}

func (cl *Client) pull(ctx context.Context, ref string) error {
	reader, err := cl.dClient.ImagePull(ctx, ref, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer func(r io.ReadCloser) { _ = r.Close() }(reader)

	if err := checkBuilding(reader); err != nil {
		return err
	}

	log.Logger.Info(fmt.Sprintf("%s is pulled.\n", ref))

	return nil
}

func (cl *Client) start(ctx context.Context, cont *dContainer) (<-chan container.ContainerWaitOKBody, <-chan error, error) {
	resp, err := cl.dClient.ContainerCreate(ctx, cont.cfg, cont.hostCfg, cont.networkCfg, cont.name)
	if err != nil {
		return nil, nil, err
	}

	err = cl.dClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return nil, nil, err
	}

	statusCh, errCh := cl.dClient.ContainerWait(ctx, resp.ID, container.WaitConditionNextExit)

	return statusCh, errCh, nil
}

// StopAndRemoveAllContainers stops and removes all used containers.
func (cl *Client) StopAndRemoveAllContainers(ctx context.Context) error {
	for _, cont := range cl.containers {
		stCh, errCh := cl.dClient.ContainerWait(ctx, cont.name, container.WaitConditionRemoved)
		go reportRemoving(stCh, errCh, cont.name)
		if err := cl.stopAndRemoveContainer(ctx, cont.name); err != nil {
			return err
		}
	}

	return nil
}

func (cl *Client) stopAndRemoveContainer(ctx context.Context, name string) error {
	if err := cl.dClient.ContainerStop(ctx, name, nil); err != nil {
		return err
	}

	removeOptions := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	return cl.dClient.ContainerRemove(ctx, name, removeOptions)
}

// HandleShutdown stops and removes all active containers when the context is closed or error is received.
//
// If error is received, os.Exit with code 2 will be called.
func (cl *Client) HandleShutdown(ctx context.Context, errCh <-chan error) {
	var (
		exit = false
	)
	select {
	case <-ctx.Done():

	case err := <-errCh:
		log.Logger.Warn(
			"Got error from containers, trying to stop all ...",
			zap.String("err", err.Error()),
		)

		exit = true
	}

	if err := cl.StopAndRemoveAllContainers(context.Background()); err != nil {
		log.Logger.Warn(
			"Got error while stopping and removing all containers. Stop and remove manually",
			zap.String("err", err.Error()),
		)
	}

	if exit {
		os.Exit(2)
	}
}

func reportRemoving(stCh <-chan container.ContainerWaitOKBody, errCh <-chan error, containerName string) {
	select {
	case st := <-stCh:
		log.Logger.Info("Got removing exit status.",
			zap.String("container", containerName),
			zap.Int64("exit_status", st.StatusCode),
		)

	case err := <-errCh:
		log.Logger.Info("Got removing err",
			zap.String("container", containerName),
			zap.String("error", err.Error()),
		)
	}
}
