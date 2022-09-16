package crypto

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	_ "crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/0chain/gosdk/core/encryption"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/herumi/bls-go-binary/bls"
	"github.com/lithammer/shortuuid/v3" //nolint
	"github.com/tyler-smith/go-bip39"   //nolint
)

var blsLock sync.Mutex

const BLS0Chain = "bls0chain"

func GenerateMnemonic(t *testing.T) string {
	entropy, _ := bip39.NewEntropy(256)       //nolint
	mnemonic, _ := bip39.NewMnemonic(entropy) //nolint
	t.Logf("Generated mnemonic [%s]", mnemonic)

	return mnemonic
}

func GenerateKeys(t *testing.T, mnemonic string) *model.KeyPair {
	blsLock.Lock()
	defer blsLock.Unlock()

	_ = bls.Init(bls.CurveFp254BNb)
	seed := bip39.NewSeed(mnemonic, "0chain-client-split-key") //nolint
	random := bytes.NewReader(seed)
	bls.SetRandFunc(random)

	var secretKey bls.SecretKey
	secretKey.SetByCSPRNG()

	publicKey := secretKey.GetPublicKey()
	secretKeyHex := secretKey.SerializeToHexStr()
	publicKeyHex := publicKey.SerializeToHexStr()

	t.Logf("Generated public key [%s] and secret key [%s]", publicKeyHex, secretKeyHex)
	bls.SetRandFunc(nil)

	return &model.KeyPair{PublicKey: *publicKey, PrivateKey: secretKey}
}

func NewConnectionID() string {
	return shortuuid.New() //nolint
}

func HashOfFileSHA1(src *os.File) (string, error) {
	h := sha1.New()
	if _, err := io.Copy(h, src); err != nil {
		return "", err
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func HashOfFileSHA256(src *os.File) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, src); err != nil {
		return "", err
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func HashTransaction(request *model.Transaction) {
	var hashData = blankIfNil(request.CreationDate) + ":" +
		blankIfNil(request.TransactionNonce) + ":" +
		blankIfNil(request.ClientId) + ":" +
		blankIfNil(request.ToClientId) + ":" +
		blankIfNil(request.TransactionValue) + ":" +
		encryption.Hash(request.TransactionData)

	request.Hash = encryption.Hash(hashData)
}

func SignTransaction(request *model.Transaction, pair *model.KeyPair) {
	hashToSign, _ := hex.DecodeString(request.Hash)
	request.Signature = pair.PrivateKey.Sign(string(hashToSign)).SerializeToHexStr()
}

func blankIfNil(obj interface{}) string {
	if obj == nil {
		return ""
	}
	return fmt.Sprintf("%v", obj)
}
