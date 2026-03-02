package model

import "time"

func DefaultBlobberRequirements(id, publicKey string) BlobberRequirements {
	return BlobberRequirements{
		Size:           64 * 1024 * 4 * 50,
		DataShards:     2,
		ParityShards:   1,
		ExpirationDate: time.Now().Add(8760 * time.Hour).Unix(), // 1 year
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
		StorageVersion: 1,
	}
}
