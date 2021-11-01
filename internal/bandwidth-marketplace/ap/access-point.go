package ap

import (
	"context"
	"time"

	"github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/0chain/gosdk/zmagmacore/magmasc/pb"
	"github.com/0chain/gosdk/zmagmacore/node"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/config"
	"github.com/0chain/system_test/internal/bandwidth-marketplace/log"
	"github.com/0chain/system_test/internal/bandwidth-marketplace/zsdk"
)

func RegisterAndStake(keysFile, nodeDir string, cfg *config.Config) (ap *magmasc.AccessPoint, err error) {
	log.Logger.Info("Registering Access Point ...")

	if err = zsdk.Init(keysFile, nodeDir, "", cfg); err != nil {
		return nil, err
	}

	ap = testAccessPoint()

	registered, err := magmasc.IsAccessPointRegisteredRP(node.ID())
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if !registered {
		if ap, err = magmasc.ExecuteAccessPointRegister(ctx, ap); err != nil {
			return nil, err
		}
	} else {
		if ap, err = magmasc.ExecuteAccessPointUpdateTerms(ctx, ap); err != nil {
			return nil, err
		}
	}

	minStake, err := magmasc.AccessPointMinStakeFetch()
	if err != nil {
		return nil, err
	}
	err = zsdk.Pour(minStake)
	if err != nil {
		return nil, err
	}

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err = magmasc.ExecuteAccessPointStake(ctx); err != nil {
		return nil, err
	}

	log.Logger.Info("Ap staked")

	return ap, nil
}

func testAccessPoint() *magmasc.AccessPoint {
	return &magmasc.AccessPoint{
		AccessPoint: &pb.AccessPoint{
			Terms: &pb.Terms{
				Price:           0.000000001,
				PriceAutoUpdate: 0.000000001,
				MinCost:         0.000000001,
				Volume:          0,
				Qos: &pb.QoS{
					DownloadMbps: 1,
					UploadMbps:   1,
				},
				QosAutoUpdate: &pb.QoSAutoUpdate{
					DownloadMbps: 0.001,
					UploadMbps:   0.001,
				},
				ProlongDuration: &durationpb.Duration{
					Seconds: int64((60 * time.Minute).Seconds()),
				},
				ExpiredAt: timestamppb.New(time.Now().Add(180 * time.Minute)),
			},
		},
	}
}
