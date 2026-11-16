package backup

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// LockName is the lock file in the data folder: the running app holds it, so
// an import (or a second app on the same folder) is refused.
const LockName = "punisher.lock"

// ErrLocked: another Punisher uses this data folder.
var ErrLocked = errors.New("다른 Punisher가 이 데이터 폴더를 쓰고 있어요. 앱을 끄고(폰 연결을 켰다면 punisher phone off) 다시 해 주세요")

// Lock takes the data folder lock (non-blocking). Release with Unlock.
func Lock(dataDir string) (*os.File, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dataDir, LockName), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrLocked
		}
		return nil, err
	}
	return f, nil
}

// Unlock releases a lock taken with Lock.
func Unlock(f *os.File) {
	if f == nil {
		return
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()
}
