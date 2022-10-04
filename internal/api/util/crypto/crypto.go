package crypto

import (
	"bytes"
	_ "crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/herumi/bls-go-binary/bls"
	"github.com/tyler-smith/go-bip39" //nolint
	"golang.org/x/crypto/sha3"
	"log"
	"sync"
	"testing"
)

const BLS0Chain = "bls0chain"

var blsLock sync.Mutex

func init() {
	blsLock.Lock()
	defer blsLock.Unlock()

	err := bls.Init(bls.CurveFp254BNb)

	if err != nil {
		panic(err)
	}
}

func GenerateMnemonics() string {
	entropy, err := bip39.NewEntropy(256) //nolint
	if err != nil {
		log.Fatalln(err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy) //nolint
	if err != nil {
		log.Fatalln(err)
	}

	return mnemonic
}

func GenerateKeys(mnemonics string) *model.RawKeyPair {
	blsLock.Lock()
	defer func() {
		blsLock.Unlock()
		bls.SetRandFunc(nil)
	}()

	seed := bip39.NewSeed(mnemonics, "0chain-client-split-key") //nolint
	random := bytes.NewReader(seed)
	bls.SetRandFunc(random)

	var secretKey bls.SecretKey
	secretKey.SetByCSPRNG()

	publicKey := secretKey.GetPublicKey()
	bls.SetRandFunc(nil)

	return &model.RawKeyPair{PublicKey: *publicKey, PrivateKey: secretKey}
}

func Sha3256(src []byte) string {
	sha3256 := sha3.New256()
	sha3256.Write(src)
	var buffer []byte
	return hex.EncodeToString(sha3256.Sum(buffer))
}

func SignTransaction(hash string, pair *model.RawKeyPair) (string, error) {
	blsLock.Lock()
	defer blsLock.Unlock()

	hashToSign, err := hex.DecodeString(hash)
	if err != nil {
		return "", err
	}

	signature := pair.PrivateKey.Sign(string(hashToSign)).SerializeToHexStr()
	return signature, nil
}

func HashTransaction(request *model.TransactionEntity) {
	var hashData = blankIfNil(request.CreationDate) + ":" +
		blankIfNil(request.TransactionNonce) + ":" +
		blankIfNil(request.ClientId) + ":" +
		blankIfNil(request.ToClientId) + ":" +
		blankIfNil(request.TransactionValue) + ":" +
		Sha3256([]byte(request.TransactionData))

	request.Hash = Sha3256([]byte(hashData))
}

func blankIfNil(obj interface{}) string {
	if obj == nil {
		return ""
	}
	return fmt.Sprintf("%v", obj)
}

func handlePanic(t *testing.T) {
	if err := recover(); err != nil {
		t.Errorf("panic occurred: ", err)
	}
}
