package crypto

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	_ "crypto/sha256"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/sha3"
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
	defer func() {
		if err := recover(); err != nil {
			t.Errorf("panic occurred: ", err)
		}
	}()
	blsLock.Lock()
	defer func() {
		blsLock.Unlock()
		bls.SetRandFunc(nil)
	}()

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
		Sha3256([]byte(request.TransactionData))

	request.Hash = Sha3256([]byte(hashData))
}

func Sha3256(publicKeyBytes []byte) string {
	sha3256 := sha3.New256()
	sha3256.Write(publicKeyBytes)
	var buffer []byte
	clientId := hex.EncodeToString(sha3256.Sum(buffer))
	return clientId
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
