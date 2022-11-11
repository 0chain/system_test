package crypto

import (
	"bytes"
	_ "crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"testing"

	"github.com/0chain/system_test/internal/api/model"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/herumi/bls-go-binary/bls"
	"github.com/stretchr/testify/require"
	"github.com/tyler-smith/go-bip39" //nolint
	"golang.org/x/crypto/sha3"
)

var blsLock sync.Mutex

const BLS0Chain = "bls0chain"

func init() {
	blsLock.Lock()
	defer blsLock.Unlock()

	err := bls.Init(bls.CurveFp254BNb)

	if err != nil {
		log.Fatalln(err) //nolint
	}
}

func GenerateMnemonics(t *testing.T) string {
	entropy, err := bip39.NewEntropy(256) //nolint
	require.NoError(t, err)
	mnemonic, err := bip39.NewMnemonic(entropy) //nolint
	require.NoError(t, err)
	t.Logf("Generated mnemonic [%s]", mnemonic)

	return mnemonic
}

func GenerateKeys(t *testing.T, mnemonics string) *model.KeyPair {
	defer handlePanic(t)
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
	secretKeyHex := secretKey.SerializeToHexStr()
	publicKeyHex := publicKey.SerializeToHexStr()

	t.Logf("Generated public key [%s] and secret key [%s]", publicKeyHex, secretKeyHex)
	bls.SetRandFunc(nil)

	return &model.KeyPair{PublicKey: *publicKey, PrivateKey: secretKey}
}

func Sha3256(src []byte) string {
	sha3256 := sha3.New256()
	sha3256.Write(src)
	var buffer []byte
	return hex.EncodeToString(sha3256.Sum(buffer))
}

func SignHashUsingSignatureScheme(hash, signatureScheme string, keys []*model.KeyPair) (string, error) {
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

		if retSignature == "" {
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

func ToSecretKey(t *testing.T, wallet *climodel.WalletFile) *bls.SecretKey {
	defer handlePanic(t)
	blsLock.Lock()
	defer blsLock.Unlock()

	var sk bls.SecretKey
	sk.SetByCSPRNG()
	err := sk.DeserializeHexStr(wallet.Keys[0].PrivateKey)
	require.Nil(t, err, "failed to serialize hex of private key")

	return &sk
}

func Sign(t *testing.T, data string, sk *bls.SecretKey) string {
	defer handlePanic(t)
	blsLock.Lock()
	defer blsLock.Unlock()

	sig := sk.Sign(data)

	return sig.SerializeToHexStr()
}

func SignHexString(t *testing.T, data string, sk *bls.SecretKey) string {
	defer handlePanic(t)
	blsLock.Lock()
	defer blsLock.Unlock()

	hashToSign, err := hex.DecodeString(data)
	require.NoError(t, err)

	signature := sk.Sign(string(hashToSign)).SerializeToHexStr()
	return signature
}

func SignTransaction(t *testing.T, request *model.TransactionPutRequest, pair *model.KeyPair) {
	defer handlePanic(t)
	blsLock.Lock()
	defer blsLock.Unlock()

	hashToSign, err := hex.DecodeString(request.Hash)
	require.NoError(t, err, "Error on hash")

	request.Signature = pair.PrivateKey.Sign(string(hashToSign)).SerializeToHexStr()
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
		t.Errorf("panic occurred: %v", err)
	}
}
