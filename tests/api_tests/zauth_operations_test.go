package api_tests

import (
	"encoding/json"
	"testing"

	"github.com/0chain/gosdk_common/core/transaction"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/crypto"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

const (
	CLIENT_ID_A       = "03df8919e4d76f6ffa9e23c388576f0baa02360d6e903a84d69689e0b2b5a28c"
	PUBLIC_KEY_I      = "ff132aaf4eeb517478c7bdd19ba887dbd909c4527b78ac989f723e5b5c349f03dc5a737071040edd5ba7a593e12264d895ad9cace1a50321886653ca8c366e13"
	PRIVATE_KEY_B     = "863f94d6deacc75c658db3e22d27c31944fc562aeef6e5b97a31d9e25294111a"
	PEER_PUBLIC_KEY_I = "f06fe5513831714965ca33080052b3873bb2e785c5d2abd88a2f958e74872617adf6e8c5fe5cb8ed0e0c218cde796707c0a9ef8f7e77756032eada0d5f3c9d00"
	HASH              = "7a8950d472c3a5a7cf0f27019c4507275d24e0bd3a97e1dedc7d77a132d9d6d3"
	SIGNATURE         = "5d65c31f8bcf64c259c3e155b78bb4c7c5ea2dae1d0d6da1a8b735f2bda2db84"

	CANCEL_ALLOCATION_TRANSACTION_TYPE                 = "cancel_allocation"
	ALLOCATION_STORAGE_OPERATIONS_TRANSACTION_TYPE_SET = "allocation_storage_operations"
)

func TestZauthOperations(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("Sign transaction with not allowed restrictions", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, PEER_PUBLIC_KEY_I)

		response, err = zauthClient.Setup(t, &model.SetupWallet{
			UserID:        client.X_APP_USER_ID,
			ClientID:      CLIENT_ID_A,
			ClientKey:     PUBLIC_KEY_I,
			PublicKey:     PUBLIC_KEY_I,
			PrivateKey:    PRIVATE_KEY_B,
			PeerPublicKey: PEER_PUBLIC_KEY_I,
			ExpiredAt:     EXPIRES_AT,
		}, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		transactionData := transaction.SmartContractTxnData{
			Name: CANCEL_ALLOCATION_TRANSACTION_TYPE,
		}

		var data []byte

		data, err = json.Marshal(transactionData)
		require.NoError(t, err)

		signTransactionPayload := &transaction.Transaction{
			Hash:            HASH,
			ClientID:        CLIENT_ID_A,
			Signature:       SIGNATURE,
			TransactionData: string(data),
		}

		_, response, err = zauthClient.SignTransaction(t, signTransactionPayload, headers)
		require.Error(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zauthClient.Delete(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Sign transaction with allowed restrictions", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, PEER_PUBLIC_KEY_I)

		response, err = zauthClient.Setup(t, &model.SetupWallet{
			UserID:        client.X_APP_USER_ID,
			ClientID:      CLIENT_ID_A,
			ClientKey:     PUBLIC_KEY_I,
			PublicKey:     PUBLIC_KEY_I,
			PrivateKey:    PRIVATE_KEY_B,
			Restrictions:  []string{ALLOCATION_STORAGE_OPERATIONS_TRANSACTION_TYPE_SET},
			PeerPublicKey: PEER_PUBLIC_KEY_I,
			ExpiredAt:     EXPIRES_AT,
		}, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		transactionData := transaction.SmartContractTxnData{
			Name: CANCEL_ALLOCATION_TRANSACTION_TYPE,
		}

		var data []byte

		data, err = json.Marshal(transactionData)
		require.NoError(t, err)

		signTransactionPayload := &transaction.Transaction{
			Hash:            HASH,
			ClientID:        CLIENT_ID_A,
			Signature:       SIGNATURE,
			TransactionData: string(data),
		}

		_, response, err = zauthClient.SignTransaction(t, signTransactionPayload, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zauthClient.Delete(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Sign transaction with the missing signature", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, PEER_PUBLIC_KEY_I)

		response, err = zauthClient.Setup(t, &model.SetupWallet{
			UserID:        client.X_APP_USER_ID,
			ClientID:      CLIENT_ID_A,
			ClientKey:     PUBLIC_KEY_I,
			PublicKey:     PUBLIC_KEY_I,
			PrivateKey:    PRIVATE_KEY_B,
			PeerPublicKey: PEER_PUBLIC_KEY_I,
			Restrictions:  []string{ALLOCATION_STORAGE_OPERATIONS_TRANSACTION_TYPE_SET},
			ExpiredAt:     EXPIRES_AT,
		}, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		transactionData := transaction.SmartContractTxnData{
			Name: CANCEL_ALLOCATION_TRANSACTION_TYPE,
		}

		var data []byte

		data, err = json.Marshal(transactionData)
		require.NoError(t, err)

		signTransactionPayload := &transaction.Transaction{
			Hash:            HASH,
			ClientID:        CLIENT_ID_A,
			Signature:       "",
			TransactionData: string(data),
		}

		_, response, err = zauthClient.SignTransaction(t, signTransactionPayload, headers)
		require.Error(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zauthClient.Delete(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Sign transaction with the missing peer public key", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, "")

		response, err = zauthClient.Setup(t, &model.SetupWallet{
			UserID:        client.X_APP_USER_ID,
			ClientID:      CLIENT_ID_A,
			ClientKey:     PUBLIC_KEY_I,
			PublicKey:     PUBLIC_KEY_I,
			PrivateKey:    PRIVATE_KEY_B,
			PeerPublicKey: PEER_PUBLIC_KEY_I,
			Restrictions:  []string{ALLOCATION_STORAGE_OPERATIONS_TRANSACTION_TYPE_SET},
			ExpiredAt:     EXPIRES_AT,
		}, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		transactionData := transaction.SmartContractTxnData{
			Name: CANCEL_ALLOCATION_TRANSACTION_TYPE,
		}

		var data []byte

		data, err = json.Marshal(transactionData)
		require.NoError(t, err)

		signTransactionPayload := &transaction.Transaction{
			Hash:            HASH,
			ClientID:        CLIENT_ID_A,
			Signature:       SIGNATURE,
			TransactionData: string(data),
		}

		_, response, err = zauthClient.SignTransaction(t, signTransactionPayload, headers)
		require.Error(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zauthClient.Delete(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Sign transaction with the missing split key", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, PEER_PUBLIC_KEY_I)

		transactionData := transaction.SmartContractTxnData{
			Name: CANCEL_ALLOCATION_TRANSACTION_TYPE,
		}

		var data []byte

		data, err = json.Marshal(transactionData)
		require.NoError(t, err)

		signTransactionPayload := &transaction.Transaction{
			Hash:            HASH,
			ClientID:        CLIENT_ID_A,
			Signature:       SIGNATURE,
			TransactionData: string(data),
		}

		_, response, err = zauthClient.SignTransaction(t, signTransactionPayload, headers)
		require.Error(t, err)
		require.Equal(t, 500, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Sign transaction with correct payload", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, PEER_PUBLIC_KEY_I)

		response, err = zauthClient.Setup(t, &model.SetupWallet{
			UserID:        client.X_APP_USER_ID,
			ClientID:      CLIENT_ID_A,
			ClientKey:     PUBLIC_KEY_I,
			PublicKey:     PUBLIC_KEY_I,
			PrivateKey:    PRIVATE_KEY_B,
			PeerPublicKey: PEER_PUBLIC_KEY_I,
			Restrictions:  []string{ALLOCATION_STORAGE_OPERATIONS_TRANSACTION_TYPE_SET},
			ExpiredAt:     EXPIRES_AT,
		}, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		transactionData := transaction.SmartContractTxnData{
			Name: CANCEL_ALLOCATION_TRANSACTION_TYPE,
		}

		var data []byte

		data, err = json.Marshal(transactionData)
		require.NoError(t, err)

		signTransactionPayload := &transaction.Transaction{
			Hash:            HASH,
			ClientID:        CLIENT_ID_A,
			Signature:       SIGNATURE,
			TransactionData: string(data),
		}

		signature, response, err := zauthClient.SignTransaction(t, signTransactionPayload, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		ok, err := crypto.Verify(t, PUBLIC_KEY_I, signature, HASH)
		require.NoError(t, err)
		require.True(t, ok)

		response, err = zauthClient.Delete(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Sign message with the missing peer public key", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, "")

		response, err = zauthClient.Setup(t, &model.SetupWallet{
			UserID:        client.X_APP_USER_ID,
			ClientID:      CLIENT_ID_A,
			ClientKey:     PUBLIC_KEY_I,
			PublicKey:     PUBLIC_KEY_I,
			PrivateKey:    PRIVATE_KEY_B,
			PeerPublicKey: PEER_PUBLIC_KEY_I,
			Restrictions:  []string{ALLOCATION_STORAGE_OPERATIONS_TRANSACTION_TYPE_SET},
			ExpiredAt:     EXPIRES_AT,
		}, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		signMessageRequest := &model.SignMessageRequest{
			Hash:      HASH,
			ClientID:  CLIENT_ID_A,
			Signature: SIGNATURE,
		}

		_, response, err = zauthClient.SignMessage(t, signMessageRequest, headers)
		require.Error(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zauthClient.Delete(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Sign message with the missing split key", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, PEER_PUBLIC_KEY_I)

		signMessageRequest := &model.SignMessageRequest{
			Hash:      HASH,
			ClientID:  CLIENT_ID_A,
			Signature: SIGNATURE,
		}

		_, response, err = zauthClient.SignMessage(t, signMessageRequest, headers)
		require.Error(t, err)
		require.Equal(t, 500, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Sign message with invalid client id in the payload", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, PEER_PUBLIC_KEY_I)

		response, err = zauthClient.Setup(t, &model.SetupWallet{
			UserID:        client.X_APP_USER_ID,
			ClientID:      CLIENT_ID_A,
			ClientKey:     PUBLIC_KEY_I,
			PublicKey:     PUBLIC_KEY_I,
			PrivateKey:    PRIVATE_KEY_B,
			PeerPublicKey: PEER_PUBLIC_KEY_I,
			Restrictions:  []string{ALLOCATION_STORAGE_OPERATIONS_TRANSACTION_TYPE_SET},
			ExpiredAt:     EXPIRES_AT,
		}, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		signMessageRequest := &model.SignMessageRequest{
			Hash:      HASH,
			ClientID:  CLIENT_ID,
			Signature: SIGNATURE,
		}

		_, response, err = zauthClient.SignMessage(t, signMessageRequest, headers)
		require.Error(t, err)
		require.Equal(t, 500, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zauthClient.Delete(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Sign message with correct payload", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, PEER_PUBLIC_KEY_I)

		response, err = zauthClient.Setup(t, &model.SetupWallet{
			UserID:        client.X_APP_USER_ID,
			ClientID:      CLIENT_ID_A,
			ClientKey:     PUBLIC_KEY_I,
			PublicKey:     PUBLIC_KEY_I,
			PrivateKey:    PRIVATE_KEY_B,
			PeerPublicKey: PEER_PUBLIC_KEY_I,
			Restrictions:  []string{ALLOCATION_STORAGE_OPERATIONS_TRANSACTION_TYPE_SET},
			ExpiredAt:     EXPIRES_AT,
		}, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		signMessageRequest := &model.SignMessageRequest{
			Hash:      HASH,
			ClientID:  CLIENT_ID_A,
			Signature: SIGNATURE,
		}

		message, response, err := zauthClient.SignMessage(t, signMessageRequest, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		ok, err := crypto.Verify(t, PUBLIC_KEY_I, message.Sig, HASH)
		require.NoError(t, err)
		require.True(t, ok)

		response, err = zauthClient.Delete(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Revoke not existing split key", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, PEER_PUBLIC_KEY_I)

		response, err = zauthClient.Revoke(t, CLIENT_ID_A, PEER_PUBLIC_KEY_I, headers)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Revoke existing split key", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, PEER_PUBLIC_KEY_I)

		response, err = zauthClient.Setup(t, &model.SetupWallet{
			UserID:        client.X_APP_USER_ID,
			ClientID:      CLIENT_ID_A,
			ClientKey:     PUBLIC_KEY_I,
			PublicKey:     PUBLIC_KEY_I,
			PrivateKey:    PRIVATE_KEY_B,
			PeerPublicKey: PEER_PUBLIC_KEY_I,
			Restrictions:  []string{ALLOCATION_STORAGE_OPERATIONS_TRANSACTION_TYPE_SET},
			ExpiredAt:     EXPIRES_AT,
		}, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zauthClient.Revoke(t, CLIENT_ID_A, PEER_PUBLIC_KEY_I, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zauthClient.Delete(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Delete not existing split key", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, PEER_PUBLIC_KEY_I)

		response, err = zauthClient.Delete(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Delete existing split key", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, PEER_PUBLIC_KEY_I)

		response, err = zauthClient.Setup(t, &model.SetupWallet{
			UserID:        client.X_APP_USER_ID,
			ClientID:      CLIENT_ID_A,
			ClientKey:     PUBLIC_KEY_I,
			PublicKey:     PUBLIC_KEY_I,
			PrivateKey:    PRIVATE_KEY_B,
			PeerPublicKey: PEER_PUBLIC_KEY_I,
			Restrictions:  []string{ALLOCATION_STORAGE_OPERATIONS_TRANSACTION_TYPE_SET},
			ExpiredAt:     EXPIRES_AT,
		}, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		response, err = zauthClient.Delete(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Retrieve details for not existing split key", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, PEER_PUBLIC_KEY_I)

		_, response, err = zauthClient.GetKeyDetails(t, CLIENT_ID_A, headers)
		require.Error(t, err)
		require.Equal(t, 500, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Retrieve split key details for existing split key", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, PEER_PUBLIC_KEY_I)

		response, err = zauthClient.Setup(t, &model.SetupWallet{
			UserID:        client.X_APP_USER_ID,
			ClientID:      CLIENT_ID_A,
			ClientKey:     PUBLIC_KEY_I,
			PublicKey:     PUBLIC_KEY_I,
			PrivateKey:    PRIVATE_KEY_B,
			PeerPublicKey: PEER_PUBLIC_KEY_I,
			Restrictions:  []string{ALLOCATION_STORAGE_OPERATIONS_TRANSACTION_TYPE_SET},
			ExpiredAt:     EXPIRES_AT,
		}, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		var keyDetails *model.KeyDetailsResponse

		headers["X-Peer-Public-Key"] = PUBLIC_KEY_I

		keyDetails, response, err = zauthClient.GetKeyDetails(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, keyDetails.LastUsed, int64(0))

		response, err = zauthClient.Delete(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Retrieve split key details last used field to be updated after message signing operation", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		jwtToken, response, err := zboxClient.CreateJwtToken(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers = zauthClient.NewZauthHeaders(jwtToken.JwtToken, PEER_PUBLIC_KEY_I)

		response, err = zauthClient.Setup(t, &model.SetupWallet{
			UserID:        client.X_APP_USER_ID,
			ClientID:      CLIENT_ID_A,
			ClientKey:     PUBLIC_KEY_I,
			PublicKey:     PUBLIC_KEY_I,
			PrivateKey:    PRIVATE_KEY_B,
			PeerPublicKey: PEER_PUBLIC_KEY_I,
			Restrictions:  []string{ALLOCATION_STORAGE_OPERATIONS_TRANSACTION_TYPE_SET},
			ExpiredAt:     EXPIRES_AT,
		}, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers["X-Peer-Public-Key"] = PUBLIC_KEY_I

		var keyDetails *model.KeyDetailsResponse

		keyDetails, response, err = zauthClient.GetKeyDetails(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, keyDetails.LastUsed, int64(0))

		headers["X-Peer-Public-Key"] = PEER_PUBLIC_KEY_I

		signMessageRequest := &model.SignMessageRequest{
			Hash:      HASH,
			ClientID:  CLIENT_ID_A,
			Signature: SIGNATURE,
		}

		_, response, err = zauthClient.SignMessage(t, signMessageRequest, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		headers["X-Peer-Public-Key"] = PUBLIC_KEY_I

		keyDetails, response, err = zauthClient.GetKeyDetails(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.NotEqual(t, keyDetails.LastUsed, int64(0))

		response, err = zauthClient.Delete(t, CLIENT_ID_A, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})
}
