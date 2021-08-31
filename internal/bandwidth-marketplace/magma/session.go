package magma

import (
	"context"
	"time"

	"github.com/0chain/gosdk/zmagmacore/errors"
	magma "github.com/magma/augmented-networks/accounting/protos"
	"go.uber.org/zap"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/config"
	"github.com/0chain/system_test/internal/bandwidth-marketplace/log"
)

// SessionStart calls protos.AccountingServer Start procedure to configured magma GRPC address.
func SessionStart(userIMSI, providerAPID, sessionID string, cfg *config.Config) error {
	const errCode = "session_start"

	cl, err := Client(cfg.Magma.GRPCAddress())
	if err != nil {
		return errors.Wrap(errCode, "error while making magma client", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.DefaultUserGRPCTimeout)*time.Second)
	defer cancel()
	req := magma.Session{
		User:        &magma.Session_IMSI{IMSI: userIMSI},
		ConsumerId:  cfg.Consumer.ExtID,
		SessionId:   sessionID,
		ProviderId:  cfg.Provider.ExtID,
		ProviderApn: providerAPID,
	}
	log.Logger.Debug("Sending Start", zap.Any("req", req))
	resp, err := cl.Start(ctx, &req)
	if err != nil {
		return errors.Wrap(errCode, "error while requesting", err)
	}
	log.Logger.Debug("Start completed", zap.Any("resp", resp), zap.Any("req", req))

	return nil
}

// SessionUpdate calls protos.AccountingServer Update procedure to configured magma GRPC address.
func SessionUpdate(userIMSI, providerAPID, sessionID string, sessTime uint32,
	octetsIn, octetsOut uint64, cfg *config.Config) error {

	const errCode = "session_update"

	cl, err := Client(cfg.Magma.GRPCAddress())
	if err != nil {
		return errors.Wrap(errCode, "error while making magma client", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.GetDefaultUserGRPCTimeout())
	defer cancel()
	req := magma.UpdateReq{
		Session: &magma.Session{
			User:        &magma.Session_IMSI{IMSI: userIMSI},
			ConsumerId:  cfg.Consumer.ExtID,
			SessionId:   sessionID,
			ProviderId:  cfg.Provider.ExtID,
			ProviderApn: providerAPID,
		},
		OctetsIn:    octetsIn,
		OctetsOut:   octetsOut,
		SessionTime: sessTime,
	}
	log.Logger.Debug("Sending Update", zap.Any("req", req))
	resp, err := cl.Update(ctx, &req)
	if err != nil {
		return errors.Wrap(errCode, "error while requesting", err)
	}
	log.Logger.Debug("Update completed", zap.Any("resp", resp), zap.Any("req", req))

	return nil
}

// SessionStop calls protos.AccountingServer Stop procedure to configured magma GRPC address.
func SessionStop(userIMSI, providerAPID, sessionID string,
	sessTime uint32, octetsIn, octetsOut uint64, cfg *config.Config) error {

	const errCode = "session_stop"

	cl, err := Client(cfg.Magma.GRPCAddress())
	if err != nil {
		return errors.Wrap(errCode, "error while making magma client", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.GetDefaultUserGRPCTimeout())
	defer cancel()
	req := magma.UpdateReq{
		Session: &magma.Session{
			User:        &magma.Session_IMSI{IMSI: userIMSI},
			ConsumerId:  cfg.Consumer.ExtID,
			SessionId:   sessionID,
			ProviderId:  cfg.Provider.ExtID,
			ProviderApn: providerAPID,
		},
		OctetsIn:    octetsIn,
		OctetsOut:   octetsOut,
		SessionTime: sessTime,
	}
	log.Logger.Debug("Sending Stop", zap.Any("req", req))
	resp, err := cl.Stop(ctx, &req)
	if err != nil {
		return errors.Wrap(errCode, "error while requesting", err)
	}
	log.Logger.Debug("Stop completed", zap.Any("resp", resp), zap.Any("req", req))

	return nil
}
