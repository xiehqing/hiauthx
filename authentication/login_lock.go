package authentication

import (
	"context"
	"fmt"
	"github.com/xiehqing/hiauthx/configx"
	"github.com/xiehqing/hiauthx/db/entity"
	"strings"
	"time"

	"github.com/xiehqing/hitoken/htputil"
)

const loginFailCountKey = "loginFailCount"

func isAdminUsername(username string) bool {
	return strings.EqualFold(strings.TrimSpace(username), "admin")
}

func (s *Service) checkLoginLocked(loginID string) error {
	if !htputil.IsDisable(loginID) {
		return nil
	}
	seconds, err := htputil.GetDisableTime(loginID)
	if err != nil || seconds <= 0 {
		return ErrUserLocked
	}
	return fmt.Errorf("%w，请 %d 秒后再试", ErrUserLocked, seconds)
}

func (s *Service) recordLoginFailure(ctx context.Context, loginID string) error {
	config := configx.New(s.queries)
	maxAttempts := config.Int(ctx, entity.SecurityLoginMaxAttempts, 5)
	lockedMinutes := config.Int(ctx, entity.SecurityLoginLockedMinutes, 30)
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	if lockedMinutes <= 0 {
		lockedMinutes = 30
	}

	lockDuration := time.Duration(lockedMinutes) * time.Minute
	session, err := htputil.GetSession(loginID)
	if err != nil {
		return err
	}

	failCount := session.GetInt(loginFailCountKey) + 1
	if failCount >= maxAttempts {
		_ = session.Delete(loginFailCountKey)
		return htputil.Disable(loginID, lockDuration)
	}
	return session.Set(loginFailCountKey, failCount, lockDuration)
}

func (s *Service) clearLoginFailure(loginID string) {
	session, err := htputil.GetSession(loginID)
	if err != nil {
		return
	}
	_ = session.Delete(loginFailCountKey)
}
