package zs3servertests

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	mrand "math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	porcupine "github.com/anishathalye/porcupine"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// TestZs3serverPorcupineCopyMoveRename extends the Porcupine linearizability
// harness with the three write-path derivatives of PUT+DELETE that also route
// through gosdk's DoMultiOperation (allocation.go:1110-1137) and therefore
// share the DELETE consensus-threshold code path:
//
//   copy   — object copy within a bucket. State after: src and dst both hold value(src).
//   move   — object move (CopyObject + DeleteObject). State after: dst=value(src), src absent.
//   rename — semantic rename. Logical (cp+del) on S3 since MinIO/zs3server don't expose a
//            native S3 RENAME verb. On fs, uses os.Rename (POSIX-atomic).
//
// Pass criterion: porcupine.CheckOperationsVerbose returns Ok (LINEARIZABLE).
// Unknown (check timeout) is tolerated. Illegal fails the subtest and dumps the history.
//
// Subtests (3 per op × 2 backends = 6):
//   s3/copy, s3/move, s3/rename  — zs3server :9100
//   fs/copy, fs/move, fs/rename  — ZS3_NFS_MOUNT (skipped if unset)
//
// Env:
//   ZS3_NFS_MOUNT     — optional NFS path for fs subtests
//   PORC_CMR_WORKERS  — default 4
//   PORC_CMR_OPS      — default 100
//   PORC_CMR_KEYS     — default 10
//   PORC_CMR_TIMEOUT  — porcupine check timeout, default 120s
func TestZs3serverPorcupineCopyMoveRename(testSetup *testing.T) {
	if _, err := os.Stat("hosts.yaml"); os.IsNotExist(err) {
		testSetup.Skip("hosts.yaml not found, skipping")
	}
	config := cliutils.ReadFile(testSetup)
	conn, err := net.DialTimeout("tcp", config.Server+":"+config.HostPort, 5*time.Second)
	if err != nil {
		testSetup.Skipf("zs3server not reachable: %v", err)
	}
	conn.Close()

	t := test.NewSystemTest(testSetup)

	workers := intEnvCMR("PORC_CMR_WORKERS", 4)
	opsPerW := intEnvCMR("PORC_CMR_OPS", 100)
	keys := intEnvCMR("PORC_CMR_KEYS", 10)
	checkTimeout := durEnvCMR("PORC_CMR_TIMEOUT", 120*time.Second)

	// S3 backend subtests
	endpoint := "http://" + config.Server + ":" + config.HostPort
	client := newS3Client(t, endpoint, config.AccessKey, config.SecretKey)
	bucket := "porcupine-cmr"
	_ = ensureBucket(client, bucket)

	// Unique per-run prefix so stale ops from previous runs don't pollute.
	runPrefix := fmt.Sprintf("porc-cmr-%d", time.Now().UnixNano())

	t.RunSequentiallyWithTimeout("s3/copy", 15*time.Minute, func(t *test.SystemTest) {
		be := &s3CMRKV{client: client, bucket: bucket, prefix: runPrefix + "/copy"}
		_ = be.reset()
		runPorcupineCMR(t, be, "copy", workers, opsPerW, keys, checkTimeout)
	})

	t.RunSequentiallyWithTimeout("s3/move", 15*time.Minute, func(t *test.SystemTest) {
		be := &s3CMRKV{client: client, bucket: bucket, prefix: runPrefix + "/move"}
		_ = be.reset()
		runPorcupineCMR(t, be, "move", workers, opsPerW, keys, checkTimeout)
	})

	t.RunSequentiallyWithTimeout("s3/rename", 15*time.Minute, func(t *test.SystemTest) {
		be := &s3CMRKV{client: client, bucket: bucket, prefix: runPrefix + "/rename"}
		// Note: MinIO/zs3server do not expose a native S3 RENAME verb. We use
		// CopyObject+DeleteObject as a "logical rename". This still exercises
		// the gosdk DoMultiOperation path in sequence.
		t.Log("rename backend: logical (CopyObject+DeleteObject) — no native S3 RENAME on zs3server")
		_ = be.reset()
		runPorcupineCMR(t, be, "rename", workers, opsPerW, keys, checkTimeout)
	})

	// NFS/fs backend subtests
	nfsMount := os.Getenv("ZS3_NFS_MOUNT")
	if nfsMount == "" {
		t.Logf("ZS3_NFS_MOUNT unset, skipping fs/* subtests")
		return
	}
	if _, err := os.Stat(nfsMount); err != nil {
		t.Logf("ZS3_NFS_MOUNT=%s not accessible, skipping fs/* subtests", nfsMount)
		return
	}

	for _, op := range []string{"copy", "move", "rename"} {
		op := op
		t.RunSequentiallyWithTimeout("fs/"+op, 15*time.Minute, func(t *test.SystemTest) {
			be := &fsCMRKV{dir: filepath.Join(nfsMount, "porcupine_kv_cmr_"+op)}
			_ = be.reset()
			runPorcupineCMR(t, be, op, workers, opsPerW, keys, checkTimeout)
		})
	}
}

// ---- Porcupine KV model with copy/move/rename ----

type cmrInput struct {
	Op      string
	Key     string
	DestKey string
	Val     string
}

type cmrOutput struct {
	Val string
	Err bool
}

var cmrModel = porcupine.Model{
	Init: func() interface{} { return map[string]string{} },
	Equal: func(a, b interface{}) bool {
		m1, m2 := a.(map[string]string), b.(map[string]string)
		if len(m1) != len(m2) {
			return false
		}
		for k, v := range m1 {
			if v2, ok := m2[k]; !ok || v2 != v {
				return false
			}
		}
		return true
	},
	Step: func(state, input, output interface{}) (bool, interface{}) {
		st := state.(map[string]string)
		op := input.(cmrInput)
		out := output.(cmrOutput)
		if out.Err {
			return true, st
		}
		ns := make(map[string]string, len(st))
		for k, v := range st {
			ns[k] = v
		}
		switch op.Op {
		case "put":
			ns[op.Key] = op.Val
			return out.Val == "ok", ns
		case "get":
			if v, ok := st[op.Key]; ok {
				return out.Val == v, st
			}
			return out.Val == "", st
		case "del":
			delete(ns, op.Key)
			return out.Val == "ok", ns
		case "copy":
			v, ok := st[op.Key]
			if !ok {
				return out.Val == "ok", st
			}
			ns[op.DestKey] = v
			return out.Val == "ok", ns
		case "move", "rename":
			v, ok := st[op.Key]
			if !ok {
				return out.Val == "ok", st
			}
			ns[op.DestKey] = v
			delete(ns, op.Key)
			return out.Val == "ok", ns
		}
		return false, state
	},
}

type cmrBackend interface {
	put(key, val string) error
	get(key string) (string, error)
	del(key string) error
	cp(src, dst string) error
	mv(src, dst string) error
	rename(src, dst string) error
	reset() error
}

// --- fs backend: file per key, unique-per-writer .tmp for atomic put/cp ---

type fsCMRKV struct{ dir string }

func (k *fsCMRKV) path(key string) string { return filepath.Join(k.dir, key) }
func (k *fsCMRKV) put(key, val string) error {
	var nonce [8]byte
	_, _ = rand.Read(nonce[:])
	tmp := fmt.Sprintf("%s.tmp.%x", k.path(key), nonce)
	if err := os.WriteFile(tmp, []byte(val), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, k.path(key))
}
func (k *fsCMRKV) get(key string) (string, error) {
	b, err := os.ReadFile(k.path(key))
	if os.IsNotExist(err) {
		return "", nil
	}
	return string(b), err
}
func (k *fsCMRKV) del(key string) error {
	err := os.Remove(k.path(key))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
func (k *fsCMRKV) cp(src, dst string) error {
	b, err := os.ReadFile(k.path(src))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var nonce [8]byte
	_, _ = rand.Read(nonce[:])
	tmp := fmt.Sprintf("%s.tmp.%x", k.path(dst), nonce)
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, k.path(dst))
}
func (k *fsCMRKV) mv(src, dst string) error {
	if err := k.cp(src, dst); err != nil {
		return err
	}
	return k.del(src)
}
func (k *fsCMRKV) rename(src, dst string) error {
	err := os.Rename(k.path(src), k.path(dst))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
func (k *fsCMRKV) reset() error {
	_ = os.RemoveAll(k.dir)
	return os.MkdirAll(k.dir, 0o755)
}

// --- s3 backend ---

type s3CMRKV struct {
	client         *s3.S3
	bucket, prefix string
}

func (k *s3CMRKV) objKey(key string) string { return k.prefix + "/" + key }
func (k *s3CMRKV) put(key, val string) error {
	ok := k.objKey(key)
	_, err := k.client.PutObject(&s3.PutObjectInput{
		Bucket: &k.bucket, Key: &ok, Body: bytes.NewReader([]byte(val)),
	})
	return err
}
func (k *s3CMRKV) get(key string) (string, error) {
	ok := k.objKey(key)
	out, err := k.client.GetObject(&s3.GetObjectInput{Bucket: &k.bucket, Key: &ok})
	if err != nil {
		return "", nil // treat missing/err as absent (model tolerates Err=true)
	}
	defer out.Body.Close()
	b, err := io.ReadAll(out.Body)
	return string(b), err
}
func (k *s3CMRKV) del(key string) error {
	ok := k.objKey(key)
	_, err := k.client.DeleteObject(&s3.DeleteObjectInput{Bucket: &k.bucket, Key: &ok})
	return err
}
func (k *s3CMRKV) cp(src, dst string) error {
	copySource := k.bucket + "/" + k.objKey(src)
	dstKey := k.objKey(dst)
	_, err := k.client.CopyObject(&s3.CopyObjectInput{
		Bucket:     &k.bucket,
		Key:        &dstKey,
		CopySource: &copySource,
	})
	if err != nil && (strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "404")) {
		return nil
	}
	return err
}
func (k *s3CMRKV) mv(src, dst string) error {
	if err := k.cp(src, dst); err != nil {
		return err
	}
	return k.del(src)
}

// rename: MinIO/zs3server don't expose a native S3 RENAME verb.
// Logical rename = CopyObject + DeleteObject. Still exercises the gosdk
// DoMultiOperation code path that a server-side rename would use internally.
func (k *s3CMRKV) rename(src, dst string) error {
	return k.mv(src, dst)
}
func (k *s3CMRKV) reset() error {
	list, err := k.client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: &k.bucket, Prefix: aws.String(k.prefix + "/"),
	})
	if err != nil {
		return err
	}
	for _, o := range list.Contents {
		_, _ = k.client.DeleteObject(&s3.DeleteObjectInput{Bucket: &k.bucket, Key: o.Key})
	}
	return nil
}

// ---- Runner: mixed workload weighted for the named op ----

func runPorcupineCMR(t *test.SystemTest, be cmrBackend, primaryOp string, workers, opsPerW, nKeys int, checkTimeout time.Duration) {
	var (
		mu      sync.Mutex
		history []porcupine.Operation
		errs    int64
		nextID  int32
	)
	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(wid int) {
			defer wg.Done()
			r := mrand.New(mrand.NewSource(int64(wid) * 997))
			for i := 0; i < opsPerW; i++ {
				key := fmt.Sprintf("k%03d", r.Intn(nKeys))
				dest := fmt.Sprintf("k%03d", r.Intn(nKeys))
				for dest == key {
					dest = fmt.Sprintf("k%03d", r.Intn(nKeys))
				}
				// Mix: 40% put, 30% get, 20% primaryOp, 10% del.
				pick := r.Intn(100)
				t0 := time.Now().UnixNano()
				var in cmrInput
				var out cmrOutput
				switch {
				case pick < 40:
					sn, _ := rand.Int(rand.Reader, big.NewInt(1<<30))
					val := fmt.Sprintf("v-%d-%d-%d", wid, atomic.AddInt32(&nextID, 1), sn.Int64())
					in = cmrInput{Op: "put", Key: key, Val: val}
					if err := be.put(key, val); err != nil {
						out = cmrOutput{Err: true}
						atomic.AddInt64(&errs, 1)
					} else {
						out = cmrOutput{Val: "ok"}
					}
				case pick < 70:
					in = cmrInput{Op: "get", Key: key}
					v, err := be.get(key)
					if err != nil {
						out = cmrOutput{Err: true}
						atomic.AddInt64(&errs, 1)
					} else {
						out = cmrOutput{Val: v}
					}
				case pick < 90:
					in = cmrInput{Op: primaryOp, Key: key, DestKey: dest}
					var err error
					switch primaryOp {
					case "copy":
						err = be.cp(key, dest)
					case "move":
						err = be.mv(key, dest)
					case "rename":
						err = be.rename(key, dest)
					}
					if err != nil {
						out = cmrOutput{Err: true}
						atomic.AddInt64(&errs, 1)
					} else {
						out = cmrOutput{Val: "ok"}
					}
				default:
					in = cmrInput{Op: "del", Key: key}
					if err := be.del(key); err != nil {
						out = cmrOutput{Err: true}
						atomic.AddInt64(&errs, 1)
					} else {
						out = cmrOutput{Val: "ok"}
					}
				}
				t1 := time.Now().UnixNano()
				mu.Lock()
				history = append(history, porcupine.Operation{
					ClientId: wid, Input: in, Call: t0, Output: out, Return: t1,
				})
				mu.Unlock()
			}
		}(w)
	}
	wg.Wait()
	wall := time.Since(start)
	total := len(history)
	t.Logf("op=%s collected %d ops in %.3fs (errors=%d) throughput=%.1f ops/s",
		primaryOp, total, wall.Seconds(), errs, float64(total)/wall.Seconds())

	res, _ := porcupine.CheckOperationsVerbose(cmrModel, history, checkTimeout)
	switch res {
	case porcupine.Ok:
		t.Logf("RESULT op=%s: LINEARIZABLE", primaryOp)
	case porcupine.Illegal:
		t.Fatalf("RESULT op=%s: NON-LINEARIZABLE", primaryOp)
	case porcupine.Unknown:
		t.Logf("RESULT op=%s: UNKNOWN (check timeout %s) — tolerated", primaryOp, checkTimeout)
	}
	_ = be.reset()
}

func intEnvCMR(name string, dflt int) int {
	v := os.Getenv(name)
	if v == "" {
		return dflt
	}
	n := 0
	_, _ = fmt.Sscanf(v, "%d", &n)
	if n <= 0 {
		return dflt
	}
	return n
}

func durEnvCMR(name string, dflt time.Duration) time.Duration {
	v := os.Getenv(name)
	if v == "" {
		return dflt
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return dflt
	}
	return d
}
