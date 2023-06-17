package model

import "time"

func DefaultBlobberRequirements(id, publicKey string) BlobberRequirements {
	return BlobberRequirements{
		Size:           120000,
		DataShards:     2,
		ParityShards:   2,
		ExpirationDate: time.Now().Add(721 * time.Hour).Unix(),
		ReadPriceRange: PriceRange{
			Min: 0,
			Max: 9223372036854775807,
		},
		WritePriceRange: PriceRange{
			Min: 0,
			Max: 9223372036854775807,
		},
		OwnerId:        id,
		OwnerPublicKey: publicKey,
	}
}
