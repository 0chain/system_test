package crypto

import (
	"bytes"
	_ "crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/herumi/bls-go-binary/bls"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/sha3"
	"log"
	"sync"
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
	defer handlePanic()
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
	defer handlePanic()
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

func SignHashUsingSignatureScheme(hash string, signatureScheme string, keys []model.RawKeyPair) (string, error) {
	retSignature := ""
	for _, kv := range keys {
		ss, err := NewSignatureScheme(signatureScheme)
		if err != nil {
			return "", err
		}
		err = ss.SetPrivateKey(kv.PrivateKey.SerializeToHexStr())
		if err != nil {
			return "", err
		}

		if len(retSignature) == 0 {
			retSignature, err = ss.Sign(hash)
		} else {
			retSignature, err = ss.Add(retSignature, hash)
		}
		if err != nil {
			return "", err
		}
	}
	return retSignature, nil
}

func SignHash(hash string, pair *model.RawKeyPair) (string, error) {
	defer handlePanic()
	blsLock.Lock()
	defer blsLock.Unlock()

	hashToSign, err := hex.DecodeString(hash)
	if err != nil {
		return "", err
	}

	signature := pair.PrivateKey.Sign(string(hashToSign)).SerializeToHexStr()
	return signature, nil
}

func CreateTransactionHash(request *model.TransactionPutRequest) string {
	return Sha3256([]byte(fmt.Sprintf("%d:%d:%s:%s:%d:%s",
		request.CreationDate,
		request.TransactionNonce,
		request.ClientId,
		request.ToClientId,
		request.TransactionValue,
		Sha3256([]byte(request.TransactionData)))))
}

func handlePanic() {
	if err := recover(); err != nil {
		log.Fatalf("panic occurred: %s", err)
	}
}
