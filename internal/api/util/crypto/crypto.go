package crypto

import (
	"bytes"
	_ "crypto/sha256"
	"encoding/hex"
	"log"
	"sync"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/herumi/bls-go-binary/bls"
	"github.com/tyler-smith/go-bip39" //nolint
	"golang.org/x/crypto/sha3"
)

var blsLock sync.Mutex

const BLS0Chain = "bls0chain"

func GenerateMnemonics() string {
	entropy, err := bip39.NewEntropy(256) //nolint
	if err != nil {
		log.Fatalln(err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy) //nolint
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Generated mnemonic [%s]\n", mnemonic)

	return mnemonic
}

func GenerateKeys(mnemonics string) *model.RawKeyPair {
	blsLock.Lock()
	defer blsLock.Unlock()

	err := bls.Init(bls.CurveFp254BNb)
	if err != nil {
		log.Fatalln(err)
	}

	seed := bip39.NewSeed(mnemonics, "0chain-client-split-key") //nolint
	random := bytes.NewReader(seed)
	bls.SetRandFunc(random)

	var secretKey bls.SecretKey
	secretKey.SetByCSPRNG()

	publicKey := secretKey.GetPublicKey()
	secretKeyHex := secretKey.SerializeToHexStr()
	publicKeyHex := publicKey.SerializeToHexStr()

	log.Printf("Generated public key [%s] and secret key [%s]\n", publicKeyHex, secretKeyHex)
	bls.SetRandFunc(nil)

	return &model.RawKeyPair{PublicKey: *publicKey, PrivateKey: secretKey}
}

func Sha3256(src []byte) string {
	sha3256 := sha3.New256()
	sha3256.Write(src)
	var buffer []byte
	return hex.EncodeToString(sha3256.Sum(buffer))
}
