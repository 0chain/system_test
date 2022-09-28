package crypto

import (
	"bytes"
	_ "crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
	"sync"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/herumi/bls-go-binary/bls"
	"github.com/tyler-smith/go-bip39" //nolint
)

var blsLock sync.Mutex

func init() {
	blsLock.Lock()
	defer blsLock.Unlock()

	err := bls.Init(bls.CurveFp254BNb)

	if err != nil {
		panic(err)
	}
}

func GenerateMnemonic(t *testing.T) string {
	entropy, _ := bip39.NewEntropy(256)       //nolint
	mnemonic, _ := bip39.NewMnemonic(entropy) //nolint
	t.Logf("Generated mnemonic [%s]", mnemonic)

	return mnemonic
}

func GenerateKeys(t *testing.T, mnemonic string) *model.KeyPair {
	defer handlePanic(t)
	blsLock.Lock()
	defer func() {
		blsLock.Unlock()
		bls.SetRandFunc(nil)
	}()

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

func SignTransaction(t *testing.T, request *model.Transaction, pair *model.KeyPair) {
	defer handlePanic(t)
	blsLock.Lock()
	defer blsLock.Unlock()

	hashToSign, err := hex.DecodeString(request.Hash)
	require.NoError(t, err, "Error on hash")

	request.Signature = pair.PrivateKey.Sign(string(hashToSign)).SerializeToHexStr()
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

func blankIfNil(obj interface{}) string {
	if obj == nil {
		return ""
	}
	return fmt.Sprintf("%v", obj)
}

func handlePanic(t *testing.T) {
	if err := recover(); err != nil {
		t.Error("panic occurred: ", err)
	}
}

func SignHash(hash string, signatureScheme string, keys []model.KeyPair) (string, error) {
	retSignature := ""
	for _, kv := range keys {
		ss := NewSignatureScheme(signatureScheme)
		err := ss.SetPrivateKey(kv.PrivateKey.GetHexString())
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