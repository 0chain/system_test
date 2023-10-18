package model

import "time"

func DefaultBlobberRequirements(id, publicKey string) BlobberRequirements {
	return BlobberRequirements{
		Size:           60000,
		DataShards:     3,
		ParityShards:   1,
		ExpirationDate: time.Now().Add(721 * time.Hour).Unix(),
		ReadPriceRange: PriceRange{
			Min: 0,
			Max: 1e10,
		},
		WritePriceRange: PriceRange{
			Min: 0,
			Max: 1e10,
		},
		OwnerId:        id,
		OwnerPublicKey: publicKey,
	}
}
